package langfuse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ezardev-team/langfuse-go/model"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

// TestEncoder_GenerationCreate_EndTimePropagated verifies that calling
// Generation(g, nil) once with both StartTime and EndTime set produces an
// OTLP span whose end_time_unix_nano > start_time_unix_nano. Prior to the
// fix, applyGenerationCreate dropped EndTime and the span fell back to
// endTime = startTime, surfacing in Langfuse as latency=0.
func TestEncoder_GenerationCreate_EndTimePropagated(t *testing.T) {
	body, _ := captureOTLP(t, func(lf *Langfuse) {
		start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		end := start.Add(750 * time.Millisecond)
		tr, err := lf.Trace(&model.Trace{Name: "encoder-endtime", Timestamp: &start})
		if err != nil {
			t.Fatalf("Trace: %v", err)
		}
		if _, err := lf.Generation(&model.Generation{
			TraceID:   tr.ID,
			Name:      "gen-endtime",
			StartTime: &start,
			EndTime:   &end,
			Model:     "test-model",
			Output:    "ok",
		}, nil); err != nil {
			t.Fatalf("Generation: %v", err)
		}
	})

	span := findSpan(t, body, "gen-endtime")
	wantStart := uint64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano())
	wantEnd := wantStart + uint64(750*time.Millisecond)

	if span.StartTimeUnixNano != wantStart {
		t.Errorf("StartTimeUnixNano=%d, want %d", span.StartTimeUnixNano, wantStart)
	}
	if span.EndTimeUnixNano != wantEnd {
		t.Errorf("EndTimeUnixNano=%d, want %d (regression: pre-fix this equalled start=%d)",
			span.EndTimeUnixNano, wantEnd, wantStart)
	}
	if span.EndTimeUnixNano <= span.StartTimeUnixNano {
		t.Fatalf("latency must be > 0; got end=%d start=%d", span.EndTimeUnixNano, span.StartTimeUnixNano)
	}
}

// TestEncoder_SpanCreate_EndTimePropagated mirrors the generation test for
// applySpanCreate.
func TestEncoder_SpanCreate_EndTimePropagated(t *testing.T) {
	body, _ := captureOTLP(t, func(lf *Langfuse) {
		start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		end := start.Add(123 * time.Millisecond)
		tr, err := lf.Trace(&model.Trace{Name: "encoder-endtime-span", Timestamp: &start})
		if err != nil {
			t.Fatalf("Trace: %v", err)
		}
		if _, err := lf.Span(&model.Span{
			TraceID:   tr.ID,
			Name:      "span-endtime",
			StartTime: &start,
			EndTime:   &end,
		}, nil); err != nil {
			t.Fatalf("Span: %v", err)
		}
	})

	span := findSpan(t, body, "span-endtime")
	wantDelta := uint64(123 * time.Millisecond)
	if got := span.EndTimeUnixNano - span.StartTimeUnixNano; got != wantDelta {
		t.Fatalf("end-start=%d ns, want %d ns", got, wantDelta)
	}
}

// captureOTLP starts an httptest server, points the Langfuse client at it,
// runs the caller-supplied operations, flushes, and returns the captured
// raw OTLP request body.
func captureOTLP(t *testing.T, do func(*Langfuse)) ([]byte, *httptest.Server) {
	t.Helper()
	var captured []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read OTLP body: %v", err)
		}
		captured = b
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("LANGFUSE_HOST", srv.URL)
	t.Setenv("LANGFUSE_PUBLIC_KEY", "test-pk")
	t.Setenv("LANGFUSE_SECRET_KEY", "test-sk")
	// Defensive: ensure no stray env from the host shell leaks in.
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	lf := New(context.Background())
	do(lf)
	lf.Flush(context.Background())

	if len(captured) == 0 {
		t.Fatal("no OTLP body captured")
	}
	return captured, srv
}

func findSpan(t *testing.T, body []byte, name string) *tracev1.Span {
	t.Helper()
	var req coltracepb.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal OTLP body: %v", err)
	}
	for _, rs := range req.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				if s.Name == name {
					return s
				}
			}
		}
	}
	t.Fatalf("span %q not found in OTLP body", name)
	return nil
}
