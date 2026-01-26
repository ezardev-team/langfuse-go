package otel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ezardev-team/langfuse-go/model"
	coltrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

const instrumentationName = "langfuse-go"

type observationKind string

const (
	observationKindTrace      observationKind = "trace"
	observationKindSpan       observationKind = "span"
	observationKindGeneration observationKind = "generation"
	observationKindEvent      observationKind = "event"
)

type observationState struct {
	kind                observationKind
	id                  string
	traceID             string
	parentID            string
	name                string
	startTime           *time.Time
	endTime             *time.Time
	fallbackTime        time.Time
	completionStartTime *time.Time
	level               model.ObservationLevel
	statusMessage       string
	input               any
	output              any
	metadata            any
	version             string
	model               string
	modelParameters     any
	usage               model.Usage
	promptName          string
	promptVersion       int
	hasPromptVersion    bool
	public              *bool
}

type traceContext struct {
	traceID        string
	traceIDBytes   []byte
	rootSpanID     []byte
	propagateAttrs []*commonv1.KeyValue
	rootAttrs      []*commonv1.KeyValue
}

func EncodeEvents(events []model.IngestionEvent) ([]byte, error) {
	traceContexts := map[string]*traceContext{}
	for _, event := range events {
		if event.Type != model.IngestionEventTypeTraceCreate {
			continue
		}
		trace, ok := asTrace(event.Body)
		if !ok {
			continue
		}
		traceID := trace.ID
		if traceID == "" {
			traceID = event.ID
		}
		if traceID == "" {
			traceID = uuidFallback(event.Timestamp)
		}
		traceContexts[traceID] = buildTraceContext(traceID, trace)
	}

	observations := map[string]*observationState{}
	for _, event := range events {
		switch event.Type {
		case model.IngestionEventTypeTraceCreate:
			trace, ok := asTrace(event.Body)
			if !ok {
				continue
			}
			traceID := trace.ID
			if traceID == "" {
				traceID = event.ID
			}
			if traceID == "" {
				traceID = uuidFallback(event.Timestamp)
			}
			state := getOrCreateObservation(observations, "trace:"+traceID, observationKindTrace, event.Timestamp)
			state.id = traceID
			state.traceID = traceID
			state.name = coalesce(state.name, trace.Name)
			state.startTime = coalesceTime(state.startTime, trace.Timestamp)
			state.input = coalesceAny(state.input, trace.Input)
			state.output = coalesceAny(state.output, trace.Output)
			state.version = coalesce(state.version, trace.Version)
			if state.public == nil {
				state.public = &trace.Public
			}
		case model.IngestionEventTypeGenerationCreate:
			gen, ok := asGeneration(event.Body)
			if !ok {
				continue
			}
			applyGenerationCreate(observations, gen, event.Timestamp)
		case model.IngestionEventTypeGenerationUpdate:
			gen, ok := asGeneration(event.Body)
			if !ok {
				continue
			}
			applyGenerationUpdate(observations, gen, event.Timestamp)
		case model.IngestionEventTypeSpanCreate:
			span, ok := asSpan(event.Body)
			if !ok {
				continue
			}
			applySpanCreate(observations, span, event.Timestamp)
		case model.IngestionEventTypeSpanUpdate:
			span, ok := asSpan(event.Body)
			if !ok {
				continue
			}
			applySpanUpdate(observations, span, event.Timestamp)
		case model.IngestionEventTypeEventCreate:
			ev, ok := asEvent(event.Body)
			if !ok {
				continue
			}
			applyEventCreate(observations, ev, event.Timestamp)
		}
	}

	spans := make([]*tracev1.Span, 0, len(observations))
	for _, obs := range observations {
		span, err := buildSpan(obs, traceContexts[obs.traceID])
		if err != nil {
			return nil, err
		}
		spans = append(spans, span)
	}

	request := &coltrace.ExportTraceServiceRequest{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						attrString("service.name", instrumentationName),
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Scope: &commonv1.InstrumentationScope{
							Name: instrumentationName,
						},
						Spans: spans,
					},
				},
			},
		},
	}

	return proto.Marshal(request)
}

func buildTraceContext(traceID string, trace *model.Trace) *traceContext {
	propagate := make([]*commonv1.KeyValue, 0)
	rootAttrs := make([]*commonv1.KeyValue, 0)

	if trace.UserID != "" {
		propagate = append(propagate, attrString("langfuse.user.id", trace.UserID))
	}
	if trace.SessionID != "" {
		propagate = append(propagate, attrString("langfuse.session.id", trace.SessionID))
	}
	if trace.Release != "" {
		propagate = append(propagate, attrString("langfuse.release", trace.Release))
	}
	if trace.Version != "" {
		propagate = append(propagate, attrString("langfuse.version", trace.Version))
	}
	if len(trace.Tags) > 0 {
		propagate = append(propagate, attrStringArray("langfuse.trace.tags", trace.Tags))
	}

	propagate = appendMetadataAttrs("langfuse.trace.metadata.", trace.Metadata, propagate)

	if trace.Name != "" {
		rootAttrs = append(rootAttrs, attrString("langfuse.trace.name", trace.Name))
	}
	if trace.Input != nil {
		rootAttrs = append(rootAttrs, attrString("langfuse.trace.input", jsonString(trace.Input)))
	}
	if trace.Output != nil {
		rootAttrs = append(rootAttrs, attrString("langfuse.trace.output", jsonString(trace.Output)))
	}
	rootAttrs = append(rootAttrs, attrBool("langfuse.trace.public", trace.Public))

	return &traceContext{
		traceID:        traceID,
		traceIDBytes:   traceIDFromString(traceID),
		rootSpanID:     spanIDFromString(traceID),
		propagateAttrs: propagate,
		rootAttrs:      rootAttrs,
	}
}

func buildSpan(state *observationState, traceCtx *traceContext) (*tracev1.Span, error) {
	traceID := state.traceID
	if traceID == "" && traceCtx != nil {
		traceID = traceCtx.traceID
	}
	if traceID == "" {
		traceID = uuidFallback(state.fallbackTime)
	}

	traceIDBytes := traceIDFromString(traceID)
	if traceCtx != nil && traceCtx.traceID == traceID {
		traceIDBytes = traceCtx.traceIDBytes
	}

	spanID := spanIDFromString(state.id)
	if state.kind == observationKindTrace && traceCtx != nil {
		spanID = traceCtx.rootSpanID
	}

	var parentSpanID []byte
	if state.kind != observationKindTrace {
		if state.parentID != "" {
			parentSpanID = spanIDFromString(state.parentID)
		} else if traceCtx != nil {
			parentSpanID = traceCtx.rootSpanID
		}
	}

	startTime := state.startTime
	if startTime == nil {
		startTime = &state.fallbackTime
	}
	endTime := state.endTime
	if endTime == nil {
		endTime = startTime
	}

	attrs := make([]*commonv1.KeyValue, 0)
	if traceCtx != nil {
		attrs = append(attrs, traceCtx.propagateAttrs...)
	}
	if state.kind == observationKindTrace && traceCtx != nil {
		attrs = append(attrs, traceCtx.rootAttrs...)
	}

	attrs = append(attrs, observationAttributes(state)...)

	span := &tracev1.Span{
		TraceId:           traceIDBytes,
		SpanId:            spanID,
		ParentSpanId:      parentSpanID,
		Name:              coalesce(state.name, string(state.kind)),
		StartTimeUnixNano: uint64(startTime.UnixNano()),
		EndTimeUnixNano:   uint64(endTime.UnixNano()),
		Attributes:        attrs,
	}

	if state.statusMessage != "" {
		span.Status = &tracev1.Status{
			Message: state.statusMessage,
		}
	}

	return span, nil
}

func observationAttributes(state *observationState) []*commonv1.KeyValue {
	attrs := make([]*commonv1.KeyValue, 0)
	obsType := observationType(state.kind)
	if obsType != "" {
		attrs = append(attrs, attrString("langfuse.observation.type", obsType))
	}
	if state.level != "" {
		attrs = append(attrs, attrString("langfuse.observation.level", string(state.level)))
	}
	if state.statusMessage != "" {
		attrs = append(attrs, attrString("langfuse.observation.status_message", state.statusMessage))
	}
	if state.input != nil {
		attrs = append(attrs, attrString("langfuse.observation.input", jsonString(state.input)))
	}
	if state.output != nil {
		attrs = append(attrs, attrString("langfuse.observation.output", jsonString(state.output)))
	}

	attrs = appendMetadataAttrs("langfuse.observation.metadata.", state.metadata, attrs)

	if state.model != "" {
		attrs = append(attrs, attrString("langfuse.observation.model.name", state.model))
	}
	if state.modelParameters != nil {
		attrs = append(attrs, attrString("langfuse.observation.model.parameters", jsonString(state.modelParameters)))
	}
	if !isZeroUsage(state.usage) {
		attrs = append(attrs, attrString("langfuse.observation.usage_details", jsonString(usageDetails(state.usage))))
	}
	if hasCost(state.usage) {
		attrs = append(attrs, attrString("langfuse.observation.cost_details", jsonString(costDetails(state.usage))))
	}
	if state.promptName != "" {
		attrs = append(attrs, attrString("langfuse.observation.prompt.name", state.promptName))
	}
	if state.hasPromptVersion {
		attrs = append(attrs, attrInt("langfuse.observation.prompt.version", int64(state.promptVersion)))
	}
	if state.completionStartTime != nil {
		attrs = append(attrs, attrString("langfuse.observation.completion_start_time", state.completionStartTime.UTC().Format(time.RFC3339Nano)))
	}
	if state.version != "" {
		attrs = append(attrs, attrString("langfuse.version", state.version))
	}
	if state.kind == observationKindTrace && state.public != nil {
		attrs = append(attrs, attrBool("langfuse.trace.public", *state.public))
	}

	return attrs
}

func observationType(kind observationKind) string {
	switch kind {
	case observationKindGeneration:
		return "generation"
	case observationKindEvent:
		return "event"
	case observationKindTrace, observationKindSpan:
		return "span"
	default:
		return ""
	}
}

func applyGenerationCreate(observations map[string]*observationState, gen *model.Generation, fallback time.Time) {
	id := coalesce(gen.ID, uuidFallback(fallback))
	key := "generation:" + id
	state := getOrCreateObservation(observations, key, observationKindGeneration, fallback)
	state.id = id
	state.traceID = coalesce(state.traceID, gen.TraceID)
	state.parentID = coalesce(state.parentID, gen.ParentObservationID)
	state.name = coalesce(state.name, gen.Name)
	state.startTime = coalesceTime(state.startTime, gen.StartTime)
	state.input = coalesceAny(state.input, gen.Input)
	state.output = coalesceAny(state.output, gen.Output)
	state.metadata = coalesceAny(state.metadata, gen.Metadata)
	if state.level == "" {
		state.level = gen.Level
	}
	state.statusMessage = coalesce(state.statusMessage, gen.StatusMessage)
	state.version = coalesce(state.version, gen.Version)
	state.model = coalesce(state.model, gen.Model)
	state.modelParameters = coalesceAny(state.modelParameters, gen.ModelParameters)
	if isZeroUsage(state.usage) {
		state.usage = gen.Usage
	}
	state.promptName = coalesce(state.promptName, gen.PromptName)
	if !state.hasPromptVersion && gen.PromptVersion != 0 {
		state.promptVersion = gen.PromptVersion
		state.hasPromptVersion = true
	}
	state.completionStartTime = coalesceTime(state.completionStartTime, gen.CompletionStartTime)
}

func applyGenerationUpdate(observations map[string]*observationState, gen *model.Generation, fallback time.Time) {
	id := coalesce(gen.ID, uuidFallback(fallback))
	key := "generation:" + id
	state := getOrCreateObservation(observations, key, observationKindGeneration, fallback)
	state.id = id
	state.traceID = coalesce(state.traceID, gen.TraceID)
	state.parentID = coalesce(state.parentID, gen.ParentObservationID)
	state.name = coalesce(state.name, gen.Name)
	state.startTime = coalesceTime(state.startTime, gen.StartTime)
	state.endTime = coalesceTime(state.endTime, gen.EndTime)
	if gen.Output != nil {
		state.output = gen.Output
	}
	if gen.Metadata != nil {
		state.metadata = gen.Metadata
	}
	if gen.Level != "" {
		state.level = gen.Level
	}
	if gen.StatusMessage != "" {
		state.statusMessage = gen.StatusMessage
	}
	if gen.Version != "" {
		state.version = gen.Version
	}
	if gen.Model != "" {
		state.model = gen.Model
	}
	if gen.ModelParameters != nil {
		state.modelParameters = gen.ModelParameters
	}
	if !isZeroUsage(gen.Usage) {
		state.usage = gen.Usage
	}
	if gen.PromptName != "" {
		state.promptName = gen.PromptName
	}
	if gen.PromptVersion != 0 {
		state.promptVersion = gen.PromptVersion
		state.hasPromptVersion = true
	}
	state.completionStartTime = coalesceTime(state.completionStartTime, gen.CompletionStartTime)
}

func applySpanCreate(observations map[string]*observationState, span *model.Span, fallback time.Time) {
	id := coalesce(span.ID, uuidFallback(fallback))
	key := "span:" + id
	state := getOrCreateObservation(observations, key, observationKindSpan, fallback)
	state.id = id
	state.traceID = coalesce(state.traceID, span.TraceID)
	state.parentID = coalesce(state.parentID, span.ParentObservationID)
	state.name = coalesce(state.name, span.Name)
	state.startTime = coalesceTime(state.startTime, span.StartTime)
	state.input = coalesceAny(state.input, span.Input)
	state.output = coalesceAny(state.output, span.Output)
	state.metadata = coalesceAny(state.metadata, span.Metadata)
	if state.level == "" {
		state.level = span.Level
	}
	state.statusMessage = coalesce(state.statusMessage, span.StatusMessage)
	state.version = coalesce(state.version, span.Version)
}

func applySpanUpdate(observations map[string]*observationState, span *model.Span, fallback time.Time) {
	id := coalesce(span.ID, uuidFallback(fallback))
	key := "span:" + id
	state := getOrCreateObservation(observations, key, observationKindSpan, fallback)
	state.id = id
	state.traceID = coalesce(state.traceID, span.TraceID)
	state.parentID = coalesce(state.parentID, span.ParentObservationID)
	state.name = coalesce(state.name, span.Name)
	state.startTime = coalesceTime(state.startTime, span.StartTime)
	state.endTime = coalesceTime(state.endTime, span.EndTime)
	if span.Output != nil {
		state.output = span.Output
	}
	if span.Metadata != nil {
		state.metadata = span.Metadata
	}
	if span.Level != "" {
		state.level = span.Level
	}
	if span.StatusMessage != "" {
		state.statusMessage = span.StatusMessage
	}
	if span.Version != "" {
		state.version = span.Version
	}
}

func applyEventCreate(observations map[string]*observationState, ev *model.Event, fallback time.Time) {
	id := coalesce(ev.ID, uuidFallback(fallback))
	key := "event:" + id
	state := getOrCreateObservation(observations, key, observationKindEvent, fallback)
	state.id = id
	state.traceID = coalesce(state.traceID, ev.TraceID)
	state.parentID = coalesce(state.parentID, ev.ParentObservationID)
	state.name = coalesce(state.name, ev.Name)
	state.startTime = coalesceTime(state.startTime, ev.StartTime)
	state.input = coalesceAny(state.input, ev.Input)
	state.output = coalesceAny(state.output, ev.Output)
	state.metadata = coalesceAny(state.metadata, ev.Metadata)
	if state.level == "" {
		state.level = ev.Level
	}
	state.statusMessage = coalesce(state.statusMessage, ev.StatusMessage)
	state.version = coalesce(state.version, ev.Version)
}

func getOrCreateObservation(observations map[string]*observationState, key string, kind observationKind, fallback time.Time) *observationState {
	if existing, ok := observations[key]; ok {
		return existing
	}
	state := &observationState{
		kind:         kind,
		fallbackTime: fallback,
	}
	observations[key] = state
	return state
}

func attrString(key, value string) *commonv1.KeyValue {
	if key == "" {
		return nil
	}
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: value},
		},
	}
}

func attrBool(key string, value bool) *commonv1.KeyValue {
	if key == "" {
		return nil
	}
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_BoolValue{BoolValue: value},
		},
	}
}

func attrInt(key string, value int64) *commonv1.KeyValue {
	if key == "" {
		return nil
	}
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_IntValue{IntValue: value},
		},
	}
}

func attrStringArray(key string, values []string) *commonv1.KeyValue {
	if key == "" || len(values) == 0 {
		return nil
	}
	arrayValues := make([]*commonv1.AnyValue, 0, len(values))
	for _, value := range values {
		arrayValues = append(arrayValues, &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: value},
		})
	}
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_ArrayValue{
				ArrayValue: &commonv1.ArrayValue{Values: arrayValues},
			},
		},
	}
}

func appendMetadataAttrs(prefix string, metadata any, attrs []*commonv1.KeyValue) []*commonv1.KeyValue {
	if metadata == nil {
		return attrs
	}

	switch m := metadata.(type) {
	case map[string]string:
		for key, value := range m {
			if key == "" {
				continue
			}
			attrs = append(attrs, attrString(prefix+key, value))
		}
	case map[string]any:
		for key, value := range m {
			if key == "" {
				continue
			}
			attrs = append(attrs, attrString(prefix+key, jsonString(value)))
		}
	default:
		attrs = append(attrs, attrString(prefix+"raw", jsonString(metadata)))
	}

	return attrs
}

func jsonString(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}

func usageDetails(usage model.Usage) map[string]any {
	details := map[string]any{}
	if usage.Input != 0 {
		details["input"] = usage.Input
	}
	if usage.Output != 0 {
		details["output"] = usage.Output
	}
	if usage.Total != 0 {
		details["total"] = usage.Total
	}
	if usage.Unit != "" {
		details["unit"] = usage.Unit
	}
	if usage.PromptTokens != 0 {
		details["promptTokens"] = usage.PromptTokens
	}
	if usage.CompletionTokens != 0 {
		details["completionTokens"] = usage.CompletionTokens
	}
	if usage.TotalTokens != 0 {
		details["totalTokens"] = usage.TotalTokens
	}
	return details
}

func costDetails(usage model.Usage) map[string]any {
	details := map[string]any{}
	if usage.InputCost != 0 {
		details["input"] = usage.InputCost
	}
	if usage.OutputCost != 0 {
		details["output"] = usage.OutputCost
	}
	if usage.TotalCost != 0 {
		details["total"] = usage.TotalCost
	}
	return details
}

func hasCost(usage model.Usage) bool {
	return usage.InputCost != 0 || usage.OutputCost != 0 || usage.TotalCost != 0
}

func isZeroUsage(usage model.Usage) bool {
	return usage == model.Usage{}
}

func coalesceTime(current *time.Time, candidate *time.Time) *time.Time {
	if current != nil {
		return current
	}
	return candidate
}

func coalesce(current, candidate string) string {
	if current != "" {
		return current
	}
	return candidate
}

func coalesceAny(current any, candidate any) any {
	if current != nil {
		return current
	}
	if candidate == nil {
		return current
	}
	return candidate
}

func asTrace(value any) (*model.Trace, bool) {
	switch t := value.(type) {
	case *model.Trace:
		return t, true
	case model.Trace:
		return &t, true
	default:
		return nil, false
	}
}

func asGeneration(value any) (*model.Generation, bool) {
	switch g := value.(type) {
	case *model.Generation:
		return g, true
	case model.Generation:
		return &g, true
	default:
		return nil, false
	}
}

func asSpan(value any) (*model.Span, bool) {
	switch s := value.(type) {
	case *model.Span:
		return s, true
	case model.Span:
		return &s, true
	default:
		return nil, false
	}
}

func asEvent(value any) (*model.Event, bool) {
	switch e := value.(type) {
	case *model.Event:
		return e, true
	case model.Event:
		return &e, true
	default:
		return nil, false
	}
}

func traceIDFromString(id string) []byte {
	if id == "" {
		return randomBytes(16, time.Now())
	}
	normalized := normalizeHex(id)
	if len(normalized) == 32 {
		if decoded, err := hex.DecodeString(normalized); err == nil {
			return decoded
		}
	}
	sum := sha256.Sum256([]byte(id))
	return sum[:16]
}

func spanIDFromString(id string) []byte {
	if id == "" {
		return randomBytes(8, time.Now())
	}
	normalized := normalizeHex(id)
	if len(normalized) == 16 {
		if decoded, err := hex.DecodeString(normalized); err == nil {
			return decoded
		}
	}
	if len(normalized) == 32 {
		if decoded, err := hex.DecodeString(normalized); err == nil {
			return decoded[:8]
		}
	}
	sum := sha256.Sum256([]byte(id))
	return sum[:8]
}

func normalizeHex(id string) string {
	normalized := strings.ToLower(strings.TrimSpace(id))
	normalized = strings.ReplaceAll(normalized, "-", "")
	for _, ch := range normalized {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return ""
		}
	}
	return normalized
}

func uuidFallback(ts time.Time) string {
	seed := fmt.Sprintf("%d", ts.UnixNano())
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:16])
}

func randomBytes(n int, ts time.Time) []byte {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%d", ts.UnixNano(), n)))
	return sum[:n]
}
