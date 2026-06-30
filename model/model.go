package model

import "time"

type IngestionEventType string

const (
	IngestionEventTypeTraceCreate      = "trace-create"
	IngestionEventTypeGenerationCreate = "generation-create"
	IngestionEventTypeGenerationUpdate = "generation-update"
	IngestionEventTypeScoreCreate      = "score-create"
	IngestionEventTypeSpanCreate       = "span-create"
	IngestionEventTypeSpanUpdate       = "span-update"
	IngestionEventTypeEventCreate      = "event-create"
)

type IngestionEvent struct {
	Type      IngestionEventType `json:"type"`
	ID        string             `json:"id"`
	Timestamp time.Time          `json:"timestamp"`
	Metadata  any
	Body      any `json:"body"`
}

type Trace struct {
	ID        string     `json:"id,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
	Name      string     `json:"name,omitempty"`
	UserID    string     `json:"userId,omitempty"`
	Input     any        `json:"input,omitempty"`
	Output    any        `json:"output,omitempty"`
	SessionID string     `json:"sessionId,omitempty"`
	Release   string     `json:"release,omitempty"`
	Version   string     `json:"version,omitempty"`
	Metadata  any        `json:"metadata,omitempty"`
	Tags      []string   `json:"tags,omitempty"`
	Public    bool       `json:"public,omitempty"`
}

type ObservationLevel string

const (
	ObservationLevelDebug   ObservationLevel = "DEBUG"
	ObservationLevelDefault ObservationLevel = "DEFAULT"
	ObservationLevelWarning ObservationLevel = "WARNING"
	ObservationLevelError   ObservationLevel = "ERROR"
)

type ObservationType string

const (
	ObservationTypeSpan       ObservationType = "SPAN"
	ObservationTypeGeneration ObservationType = "GENERATION"
	ObservationTypeEvent      ObservationType = "EVENT"
	ObservationTypeAgent      ObservationType = "AGENT"
	ObservationTypeTool       ObservationType = "TOOL"
	ObservationTypeChain      ObservationType = "CHAIN"
	ObservationTypeRetriever  ObservationType = "RETRIEVER"
	ObservationTypeEvaluator  ObservationType = "EVALUATOR"
	ObservationTypeEmbedding  ObservationType = "EMBEDDING"
	ObservationTypeGuardrail  ObservationType = "GUARDRAIL"
)

type Generation struct {
	TraceID             string           `json:"traceId,omitempty"`
	Name                string           `json:"name,omitempty"`
	StartTime           *time.Time       `json:"startTime,omitempty"`
	Metadata            any              `json:"metadata,omitempty"`
	Input               any              `json:"input,omitempty"`
	Output              any              `json:"output,omitempty"`
	Level               ObservationLevel `json:"level,omitempty"`
	StatusMessage       string           `json:"statusMessage,omitempty"`
	ParentObservationID string           `json:"parentObservationId,omitempty"`
	Version             string           `json:"version,omitempty"`
	ID                  string           `json:"id,omitempty"`
	EndTime             *time.Time       `json:"endTime,omitempty"`
	CompletionStartTime *time.Time       `json:"completionStartTime,omitempty"`
	Model               string           `json:"model,omitempty"`
	ModelParameters     any              `json:"modelParameters,omitempty"`
	Usage               Usage            `json:"usage,omitempty"`
	PromptName          string           `json:"promptName,omitempty"`
	PromptVersion       int              `json:"promptVersion,omitempty"`
	// Prompt object reference for Linked Generations auto-matching.
	Prompt *Prompt `json:"prompt,omitempty"`
}

type Usage struct {
	Input      int       `json:"input,omitempty"`
	Output     int       `json:"output,omitempty"`
	Total      int       `json:"total,omitempty"`
	Unit       UsageUnit `json:"unit,omitempty"`
	InputCost  float64   `json:"inputCost,omitempty"`
	OutputCost float64   `json:"outputCost,omitempty"`
	TotalCost  float64   `json:"totalCost,omitempty"`

	PromptTokens     int `json:"promptTokens,omitempty"`
	CompletionTokens int `json:"completionTokens,omitempty"`
	TotalTokens      int `json:"totalTokens,omitempty"`

	// UsageDetails carries arbitrary usage-type token counts beyond the fixed
	// fields above (e.g. "output_reasoning", "cache_read_input_tokens"). It is
	// merged into the emitted langfuse.observation.usage_details attribute and
	// its keys override the fixed-field defaults on collision. Ingestion is via
	// the OTLP encoder, so these are not part of the JSON ingestion payload.
	UsageDetails map[string]int `json:"-"`
	// CostDetails carries arbitrary precomputed per-usage-type costs in USD
	// (e.g. "input", "output", "output_reasoning", "cache_read_input_tokens").
	// It is merged into the emitted langfuse.observation.cost_details attribute.
	// When present, Langfuse uses these ingested costs directly instead of
	// inferring cost from the model definition.
	CostDetails map[string]float64 `json:"-"`
}

type UsageUnit string

const (
	ModelUsageUnitCharacters   UsageUnit = "CHARACTERS"
	ModelUsageUnitTokens       UsageUnit = "TOKENS"
	ModelUsageUnitMilliseconds UsageUnit = "MILLISECONDS"
	ModelUsageUnitSeconds      UsageUnit = "SECONDS"
	ModelUsageUnitImages       UsageUnit = "IMAGES"
)

type Score struct {
	ID            string  `json:"id,omitempty"`
	TraceID       string  `json:"traceId,omitempty"`
	Name          string  `json:"name,omitempty"`
	Value         float64 `json:"value,omitempty"`
	ObservationID string  `json:"observationId,omitempty"`
	Comment       string  `json:"comment,omitempty"`
}

type Span struct {
	TraceID             string           `json:"traceId,omitempty"`
	Name                string           `json:"name,omitempty"`
	StartTime           *time.Time       `json:"startTime,omitempty"`
	Metadata            any              `json:"metadata,omitempty"`
	Input               any              `json:"input,omitempty"`
	Output              any              `json:"output,omitempty"`
	Level               ObservationLevel `json:"level,omitempty"`
	StatusMessage       string           `json:"statusMessage,omitempty"`
	ParentObservationID string           `json:"parentObservationId,omitempty"`
	Version             string           `json:"version,omitempty"`
	ID                  string           `json:"id,omitempty"`
	EndTime             *time.Time       `json:"endTime,omitempty"`
}

type Event struct {
	TraceID             string           `json:"traceId,omitempty"`
	Name                string           `json:"name,omitempty"`
	StartTime           *time.Time       `json:"startTime,omitempty"`
	Metadata            any              `json:"metadata,omitempty"`
	Input               any              `json:"input,omitempty"`
	Output              any              `json:"output,omitempty"`
	Level               ObservationLevel `json:"level,omitempty"`
	StatusMessage       string           `json:"statusMessage,omitempty"`
	ParentObservationID string           `json:"parentObservationId,omitempty"`
	Version             string           `json:"version,omitempty"`
	ID                  string           `json:"id,omitempty"`
}

// ObservationView is decoded from `GET /api/public/v2/observations`.
//
// Fields available depend on the request's `fields` parameter; missing groups
// stay at zero values. See:
// https://api.reference.langfuse.com/#tag/observationsv2/GET/api/public/v2/observations
type ObservationView struct {
	ID                  string             `json:"id"`
	ProjectID           string             `json:"projectId,omitempty"`
	TraceID             string             `json:"traceId,omitempty"`
	Type                ObservationType    `json:"type,omitempty"`
	Name                string             `json:"name,omitempty"`
	StartTime           *time.Time         `json:"startTime,omitempty"`
	EndTime             *time.Time         `json:"endTime,omitempty"`
	CompletionStartTime *time.Time         `json:"completionStartTime,omitempty"`
	CreatedAt           *time.Time         `json:"createdAt,omitempty"`
	UpdatedAt           *time.Time         `json:"updatedAt,omitempty"`
	Model               string             `json:"providedModelName,omitempty"`
	ModelID             string             `json:"internalModelId,omitempty"`
	ModelParameters     any                `json:"modelParameters,omitempty"`
	Input               any                `json:"input,omitempty"`
	Version             string             `json:"version,omitempty"`
	Metadata            any                `json:"metadata,omitempty"`
	Output              any                `json:"output,omitempty"`
	UsageDetails        map[string]int     `json:"usageDetails,omitempty"`
	CostDetails         map[string]float64 `json:"costDetails,omitempty"`
	TotalCost           float64            `json:"totalCost,omitempty"`
	Latency             float64            `json:"latency,omitempty"`
	TimeToFirstToken    float64            `json:"timeToFirstToken,omitempty"`
	Level               ObservationLevel   `json:"level,omitempty"`
	StatusMessage       string             `json:"statusMessage,omitempty"`
	ParentObservationID string             `json:"parentObservationId,omitempty"`
	PromptID            string             `json:"promptId,omitempty"`
	PromptName          string             `json:"promptName,omitempty"`
	PromptVersion       int                `json:"promptVersion,omitempty"`
	UserID              string             `json:"userId,omitempty"`
	SessionID           string             `json:"sessionId,omitempty"`
	Environment         string             `json:"environment,omitempty"`
	Bookmarked          bool               `json:"bookmarked,omitempty"`
	Public              bool               `json:"public,omitempty"`
}

// ObservationsCursorMeta is the pagination meta returned by
// `GET /api/public/v2/observations`. Cursor is nil when no more pages remain.
type ObservationsCursorMeta struct {
	Cursor *string `json:"cursor"`
}

type Prompt struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Version     int        `json:"version,omitempty"`
	Label       string     `json:"label,omitempty"`
	Environment string     `json:"environment,omitempty"`
	Prompt      any        `json:"prompt,omitempty"`
	Config      any        `json:"config,omitempty"`
	Metadata    any        `json:"metadata,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
}

type PromptRequestOptions struct {
	Version     *int
	Label       string
	Environment string
}

type M map[string]interface{}
