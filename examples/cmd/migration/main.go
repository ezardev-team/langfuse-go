package main

import (
	"context"
	"log"
	"time"

	"github.com/ezardev-team/langfuse-go"
	"github.com/ezardev-team/langfuse-go/model"
)

// Migration example:
//   - If you previously sent events to /api/public/ingestion manually,
//     switch to the high-level Langfuse API below. It now sends OTLP/HTTP
//     to /api/public/otel/v1/traces by default.
func main() {
	ctx := context.Background()
	client := langfuse.New(ctx)

	trace, err := client.Trace(&model.Trace{
		Name:      "migration-trace",
		UserID:    "user-123",
		SessionID: "session-abc",
		Metadata:  model.M{"source": "migration-example"},
	})
	if err != nil {
		log.Fatalf("trace error: %v", err)
	}

	spanStart := time.Now().Add(-250 * time.Millisecond)
	spanEnd := time.Now()
	span, err := client.Span(&model.Span{
		TraceID:   trace.ID,
		Name:      "migration-span",
		StartTime: &spanStart,
		EndTime:   &spanEnd,
		Input:     model.M{"prompt": "summarize"},
		Output:    model.M{"result": "summary"},
	}, nil)
	if err != nil {
		log.Fatalf("span error: %v", err)
	}

	genStart := time.Now().Add(-200 * time.Millisecond)
	genEnd := time.Now()
	generation, err := client.Generation(&model.Generation{
		TraceID:   trace.ID,
		Name:      "migration-generation",
		Model:     "gpt-4o-mini",
		StartTime: &genStart,
		EndTime:   &genEnd,
		Input:     []model.M{{"role": "user", "content": "Summarize this."}},
		Output:    model.M{"role": "assistant", "content": "Summary."},
		Usage: model.Usage{
			Input:  64,
			Output: 32,
			Total:  96,
			Unit:   model.ModelUsageUnitTokens,
		},
	}, &span.ID)
	if err != nil {
		log.Fatalf("generation error: %v", err)
	}

	_, err = client.Event(&model.Event{
		TraceID: trace.ID,
		Name:    "migration-event",
		Input:   model.M{"payload": "value"},
		Output:  model.M{"status": "ok"},
	}, &generation.ID)
	if err != nil {
		log.Fatalf("event error: %v", err)
	}

	_, err = client.Score(&model.Score{
		TraceID: trace.ID,
		Name:    "migration-score",
		Value:   0.95,
	})
	if err != nil {
		log.Fatalf("score error: %v", err)
	}

	client.Flush(ctx)
}
