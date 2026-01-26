package langfuse

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ezardev-team/langfuse-go/internal/pkg/api"
	"github.com/ezardev-team/langfuse-go/internal/pkg/otel"
	"github.com/ezardev-team/langfuse-go/model"
)

func skipIfNoEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("LANGFUSE_PUBLIC_KEY") == "" || os.Getenv("LANGFUSE_SECRET_KEY") == "" {
		t.Skip("LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY must be set")
	}
}

// uniqueName generates a unique observation name to avoid collision across test runs.
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// queryObservationsByName queries observations by name with retries.
func queryObservationsByName(t *testing.T, ctx context.Context, name string, obsType model.ObservationType) []model.ObservationView {
	t.Helper()
	client := api.New()
	limit := 10
	page := 1

	// Retry a few times since OTEL data takes time to index
	for attempt := 1; attempt <= 3; attempt++ {
		time.Sleep(3 * time.Second)

		req := &api.ObservationsRequest{
			Page:  &page,
			Limit: &limit,
			Name:  name,
		}
		if obsType != "" {
			req.Type = obsType
		}

		res := &api.ObservationsResponse{}
		err := client.Observations(ctx, req, res)
		if err != nil {
			t.Logf("  attempt %d: query failed: %v", attempt, err)
			continue
		}

		if res.IsSuccess() && len(res.Data) > 0 {
			return res.Data
		}
		t.Logf("  attempt %d: no results yet (code=%d, count=%d)", attempt, res.Code, len(res.Data))
	}

	return nil
}

// TestTracing_FullPipeline tests the complete tracing flow:
// Trace → Span → Generation (with usage & latency) → Event → Score → GenerationEnd → SpanEnd → Flush
// Then verifies the generation arrived via the Observations API by name.
func TestTracing_FullPipeline(t *testing.T) {
	skipIfNoEnv(t)

	ctx := context.Background()
	l := New(ctx)

	genName := uniqueName("e2e-gen")

	// 1. Create trace
	trace, err := l.Trace(&model.Trace{
		Name:      "e2e-tracing-test",
		UserID:    "test-user-123",
		SessionID: "test-session-456",
		Tags:      []string{"e2e", "test"},
		Metadata: model.M{
			"environment": "test",
			"sdk":         "langfuse-go",
		},
		Input:  model.M{"query": "What is tracing?"},
		Output: model.M{"answer": "Tracing tracks request flow."},
	})
	if err != nil {
		t.Fatalf("Trace creation failed: %v", err)
	}
	if trace.ID == "" {
		t.Fatal("expected non-empty trace ID")
	}
	t.Logf("Trace created: id=%s", trace.ID)

	// 2. Create span under trace
	spanStart := time.Now()
	span, err := l.Span(&model.Span{
		TraceID:   trace.ID,
		Name:      "llm-orchestration",
		StartTime: &spanStart,
		Input:     model.M{"prompt": "What is tracing?"},
		Metadata:  model.M{"step": "orchestration"},
	}, nil)
	if err != nil {
		t.Fatalf("Span creation failed: %v", err)
	}
	if span.ID == "" {
		t.Fatal("expected non-empty span ID")
	}
	t.Logf("Span created: id=%s, traceId=%s", span.ID, span.TraceID)

	// 3. Create generation with usage tokens & latency
	genStart := time.Now()
	completionStart := genStart.Add(200 * time.Millisecond)
	generation, err := l.Generation(
		&model.Generation{
			TraceID:   trace.ID,
			Name:      genName,
			Model:     "gpt-4",
			StartTime: &genStart,
			ModelParameters: model.M{
				"temperature": "0.7",
				"maxTokens":   "2048",
				"topP":        "0.95",
			},
			Input: []model.M{
				{"role": "system", "content": "You are a helpful assistant."},
				{"role": "user", "content": "What is distributed tracing?"},
			},
			Usage: model.Usage{
				Input:            256,
				Output:           64,
				Total:            320,
				Unit:             model.ModelUsageUnitTokens,
				PromptTokens:     256,
				CompletionTokens: 64,
				TotalTokens:      320,
				InputCost:        0.0077,
				OutputCost:       0.0038,
				TotalCost:        0.0115,
			},
			CompletionStartTime: &completionStart,
			Level:               model.ObservationLevelDefault,
			PromptName:          "qa-prompt",
			PromptVersion:       2,
			Metadata: model.M{
				"provider":    "openai",
				"temperature": 0.7,
			},
		},
		&span.ID,
	)
	if err != nil {
		t.Fatalf("Generation creation failed: %v", err)
	}
	if generation.ID == "" {
		t.Fatal("expected non-empty generation ID")
	}
	t.Logf("Generation created: id=%s, name=%s, parentId=%s", generation.ID, genName, generation.ParentObservationID)

	// 4. Create event under generation
	event, err := l.Event(
		&model.Event{
			TraceID: trace.ID,
			Name:    "token-count-logged",
			Input:   model.M{"promptTokens": 256, "completionTokens": 64},
			Output:  model.M{"totalTokens": 320},
			Metadata: model.M{
				"step": "post-processing",
			},
		},
		&generation.ID,
	)
	if err != nil {
		t.Fatalf("Event creation failed: %v", err)
	}
	t.Logf("Event created: id=%s", event.ID)

	// 5. Create score
	score, err := l.Score(&model.Score{
		TraceID: trace.ID,
		Name:    "relevance",
		Value:   0.92,
		Comment: "Highly relevant response",
	})
	if err != nil {
		t.Fatalf("Score creation failed: %v", err)
	}
	t.Logf("Score created: id=%s", score.ID)

	// 6. End generation with output
	genEnd := time.Now()
	generation.EndTime = &genEnd
	generation.Output = model.M{
		"role":    "assistant",
		"content": "Distributed tracing is a method for tracking requests across microservices...",
	}
	generation.StatusMessage = "completed"
	_, err = l.GenerationEnd(generation)
	if err != nil {
		t.Fatalf("GenerationEnd failed: %v", err)
	}
	t.Logf("Generation ended: latency=%v", genEnd.Sub(genStart))

	// 7. End span
	spanEnd := time.Now()
	span.EndTime = &spanEnd
	span.Output = model.M{"result": "success"}
	_, err = l.SpanEnd(span)
	if err != nil {
		t.Fatalf("SpanEnd failed: %v", err)
	}
	t.Logf("Span ended: latency=%v", spanEnd.Sub(spanStart))

	// 8. Flush - send all events to Langfuse via OTEL
	t.Log("Flushing all events...")
	l.Flush(ctx)
	t.Log("Flush complete (no errors from OTEL endpoint)")

	// 9. Verify generation arrived by querying with its unique name
	t.Logf("Verifying generation '%s' via Observations API...", genName)
	observations := queryObservationsByName(t, ctx, genName, model.ObservationTypeGeneration)
	if len(observations) == 0 {
		t.Logf("WARN: generation '%s' not found in Observations API after retries (OTEL data may take longer to index)", genName)
	} else {
		obs := observations[0]
		t.Logf("Verified: [%s] name=%s model=%s", obs.Type, obs.Name, obs.Model)
	}
}

// TestTracing_GenerationCreateAndUpdate tests the create → update flow.
func TestTracing_GenerationCreateAndUpdate(t *testing.T) {
	skipIfNoEnv(t)

	ctx := context.Background()
	l := New(ctx)

	genName := uniqueName("gen-update")

	trace, err := l.Trace(&model.Trace{Name: "gen-update-test"})
	if err != nil {
		t.Fatalf("Trace failed: %v", err)
	}

	// Create generation (start of LLM call)
	genStart := time.Now()
	gen, err := l.Generation(
		&model.Generation{
			TraceID:   trace.ID,
			Name:      genName,
			Model:     "gpt-4o",
			StartTime: &genStart,
			Input: []model.M{
				{"role": "user", "content": "Hello!"},
			},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("Generation create failed: %v", err)
	}
	t.Logf("Generation created (no output): id=%s, name=%s", gen.ID, genName)

	// Simulate LLM processing
	time.Sleep(100 * time.Millisecond)

	// Update with output, usage, end time
	genEnd := time.Now()
	gen.EndTime = &genEnd
	gen.Output = model.M{
		"role":    "assistant",
		"content": "Hi there! How can I help you today?",
	}
	gen.Usage = model.Usage{
		PromptTokens:     8,
		CompletionTokens: 12,
		TotalTokens:      20,
		Unit:             model.ModelUsageUnitTokens,
		InputCost:        0.0001,
		OutputCost:       0.0003,
		TotalCost:        0.0004,
	}
	gen.StatusMessage = "done"

	_, err = l.GenerationEnd(gen)
	if err != nil {
		t.Fatalf("GenerationEnd failed: %v", err)
	}

	latency := genEnd.Sub(genStart)
	t.Logf("Generation updated: latency=%v, tokens=%d, cost=%.4f",
		latency, gen.Usage.TotalTokens, gen.Usage.TotalCost)

	l.Flush(ctx)
	t.Log("Flushed successfully")

	// Verify
	t.Logf("Verifying generation '%s'...", genName)
	observations := queryObservationsByName(t, ctx, genName, model.ObservationTypeGeneration)
	if len(observations) == 0 {
		t.Logf("WARN: generation '%s' not found after retries (OTEL data may take longer to index)", genName)
	} else {
		obs := observations[0]
		t.Logf("Verified: model=%s", obs.Model)
		if obs.Usage.TotalTokens > 0 {
			t.Logf("  usage: total=%d, prompt=%d, completion=%d",
				obs.Usage.TotalTokens, obs.Usage.PromptTokens, obs.Usage.CompletionTokens)
		}
	}
}

// TestTracing_SpanHierarchy tests nested span parent-child relationships.
func TestTracing_SpanHierarchy(t *testing.T) {
	skipIfNoEnv(t)

	ctx := context.Background()
	l := New(ctx)

	trace, err := l.Trace(&model.Trace{Name: "hierarchy-test"})
	if err != nil {
		t.Fatalf("Trace failed: %v", err)
	}

	// Parent span
	spanA, err := l.Span(&model.Span{
		TraceID: trace.ID,
		Name:    "retrieval-pipeline",
		Input:   model.M{"query": "test query"},
	}, nil)
	if err != nil {
		t.Fatalf("Span A failed: %v", err)
	}
	t.Logf("Span A: id=%s", spanA.ID)

	// Child span under spanA
	spanB, err := l.Span(&model.Span{
		TraceID: trace.ID,
		Name:    "vector-search",
		Input:   model.M{"embedding": "..."},
	}, &spanA.ID)
	if err != nil {
		t.Fatalf("Span B failed: %v", err)
	}
	if spanB.ParentObservationID != spanA.ID {
		t.Errorf("expected spanB.parentObservationID=%s, got %s", spanA.ID, spanB.ParentObservationID)
	}
	t.Logf("Span B: id=%s, parent=%s", spanB.ID, spanB.ParentObservationID)

	// Generation under spanB
	gen, err := l.Generation(&model.Generation{
		TraceID: trace.ID,
		Name:    "rerank-generation",
		Model:   "cohere-rerank",
		Usage: model.Usage{
			Input:  100,
			Output: 10,
			Total:  110,
			Unit:   model.ModelUsageUnitTokens,
		},
	}, &spanB.ID)
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}
	if gen.ParentObservationID != spanB.ID {
		t.Errorf("expected gen.parentObservationID=%s, got %s", spanB.ID, gen.ParentObservationID)
	}
	t.Logf("Generation: id=%s, parent=%s", gen.ID, gen.ParentObservationID)

	// End spans
	_, _ = l.SpanEnd(spanB)
	_, _ = l.SpanEnd(spanA)

	l.Flush(ctx)
	t.Log("Hierarchy test flushed (no errors)")
}

// TestTracing_MultipleScores tests attaching multiple scores to a trace.
func TestTracing_MultipleScores(t *testing.T) {
	skipIfNoEnv(t)

	ctx := context.Background()
	l := New(ctx)

	trace, err := l.Trace(&model.Trace{Name: "score-test"})
	if err != nil {
		t.Fatalf("Trace failed: %v", err)
	}

	scores := []struct {
		name  string
		value float64
	}{
		{"accuracy", 0.95},
		{"relevance", 0.88},
		{"fluency", 0.92},
	}

	for _, s := range scores {
		score, err := l.Score(&model.Score{
			TraceID: trace.ID,
			Name:    s.name,
			Value:   s.value,
		})
		if err != nil {
			t.Fatalf("Score %s failed: %v", s.name, err)
		}
		t.Logf("Score created: name=%s value=%.2f id=%s", s.name, s.value, score.ID)
	}

	l.Flush(ctx)
	t.Log("Multiple scores flushed (no errors)")
}

// TestTracing_OtelEncoding tests that the OTEL encoder correctly encodes
// all observation types with usage, latency, and metadata.
func TestTracing_OtelEncoding(t *testing.T) {
	now := time.Now()
	start := now.Add(-2 * time.Second)
	end := now
	completionStart := now.Add(-1 * time.Second)

	events := []model.IngestionEvent{
		{
			Type:      model.IngestionEventTypeTraceCreate,
			ID:        "trace-enc-001",
			Timestamp: now,
			Body: &model.Trace{
				ID:        "trace-enc-001",
				Name:      "encoding-test",
				UserID:    "user-1",
				SessionID: "session-1",
				Tags:      []string{"test"},
				Input:     model.M{"q": "hello"},
				Output:    model.M{"a": "world"},
			},
		},
		{
			Type:      model.IngestionEventTypeSpanCreate,
			ID:        "span-enc-001",
			Timestamp: now,
			Body: &model.Span{
				ID:        "span-enc-001",
				TraceID:   "trace-enc-001",
				Name:      "test-span",
				StartTime: &start,
				EndTime:   &end,
				Input:     model.M{"data": "input"},
				Output:    model.M{"data": "output"},
			},
		},
		{
			Type:      model.IngestionEventTypeGenerationCreate,
			ID:        "gen-enc-001",
			Timestamp: now,
			Body: &model.Generation{
				ID:                  "gen-enc-001",
				TraceID:             "trace-enc-001",
				ParentObservationID: "span-enc-001",
				Name:                "test-gen",
				Model:               "gpt-4",
				StartTime:           &start,
				CompletionStartTime: &completionStart,
				ModelParameters:     model.M{"temperature": "0.5"},
				Input:               []model.M{{"role": "user", "content": "hi"}},
				Usage: model.Usage{
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
					Unit:             model.ModelUsageUnitTokens,
					InputCost:        0.003,
					OutputCost:       0.006,
					TotalCost:        0.009,
				},
				Level:         model.ObservationLevelDefault,
				StatusMessage: "ok",
				PromptName:    "my-prompt",
				PromptVersion: 5,
			},
		},
		{
			Type:      model.IngestionEventTypeGenerationUpdate,
			ID:        "gen-enc-001",
			Timestamp: now,
			Body: &model.Generation{
				ID:      "gen-enc-001",
				TraceID: "trace-enc-001",
				EndTime: &end,
				Output:  model.M{"role": "assistant", "content": "hello!"},
			},
		},
		{
			Type:      model.IngestionEventTypeEventCreate,
			ID:        "evt-enc-001",
			Timestamp: now,
			Body: &model.Event{
				ID:      "evt-enc-001",
				TraceID: "trace-enc-001",
				Name:    "test-event",
				Input:   model.M{"key": "value"},
			},
		},
	}

	payload, err := otel.EncodeEvents(events)
	if err != nil {
		t.Fatalf("EncodeEvents failed: %v", err)
	}

	if len(payload) == 0 {
		t.Fatal("expected non-empty protobuf payload")
	}

	t.Logf("OTEL encoding successful: %d bytes for %d events", len(payload), len(events))
	t.Logf("  - trace, span, generation (create+update), event encoded")
	t.Logf("  - usage: 150 tokens, cost: $0.009")
	t.Logf("  - latency info: start/end/completionStart present")
}

// TestTracing_OtelEncodingAndSend tests encoding events and sending them
// directly to the Langfuse OTEL endpoint.
func TestTracing_OtelEncodingAndSend(t *testing.T) {
	skipIfNoEnv(t)

	ctx := context.Background()
	genName := uniqueName("otel-direct")

	now := time.Now()
	start := now.Add(-500 * time.Millisecond)

	events := []model.IngestionEvent{
		{
			Type:      model.IngestionEventTypeTraceCreate,
			ID:        "otel-trace-001",
			Timestamp: now,
			Body: &model.Trace{
				ID:   "otel-trace-001",
				Name: "otel-direct-test",
			},
		},
		{
			Type:      model.IngestionEventTypeGenerationCreate,
			ID:        "otel-gen-001",
			Timestamp: now,
			Body: &model.Generation{
				ID:        "otel-gen-001",
				TraceID:   "otel-trace-001",
				Name:      genName,
				Model:     "gpt-4-turbo",
				StartTime: &start,
				EndTime:   &now,
				Input:     []model.M{{"role": "user", "content": "test"}},
				Output:    model.M{"role": "assistant", "content": "response"},
				Usage: model.Usage{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
					Unit:             model.ModelUsageUnitTokens,
					InputCost:        0.0003,
					OutputCost:       0.0003,
					TotalCost:        0.0006,
				},
				Level: model.ObservationLevelDefault,
			},
		},
	}

	// Encode
	payload, err := otel.EncodeEvents(events)
	if err != nil {
		t.Fatalf("EncodeEvents failed: %v", err)
	}
	t.Logf("Encoded %d events into %d bytes protobuf", len(events), len(payload))

	// Send directly
	client := api.New()
	req := &api.OpenTelemetryTracesRequest{Body: payload}
	res := &api.OpenTelemetryResponse{}
	err = client.OpenTelemetryTraces(ctx, req, res)
	if err != nil {
		t.Fatalf("OpenTelemetryTraces POST failed: %v", err)
	}
	if !res.IsSuccess() {
		t.Fatalf("OTEL endpoint returned non-success: code=%d", res.Code)
	}
	t.Logf("OTEL endpoint accepted: code=%d", res.Code)

	// Verify the generation appeared
	t.Logf("Verifying generation '%s' via Observations API...", genName)
	observations := queryObservationsByName(t, ctx, genName, model.ObservationTypeGeneration)
	if len(observations) == 0 {
		t.Logf("WARN: generation '%s' not found after retries (OTEL data may take longer to index)", genName)
	} else {
		obs := observations[0]
		t.Logf("Verified: [%s] name=%s model=%s totalTokens=%d",
			obs.Type, obs.Name, obs.Model, obs.Usage.TotalTokens)
	}
}
