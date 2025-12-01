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

type ObservationView struct {
	ID                  string             `json:"id"`
	TraceID             string             `json:"traceId,omitempty"`
	Type                ObservationType    `json:"type,omitempty"`
	Name                string             `json:"name,omitempty"`
	StartTime           *time.Time         `json:"startTime,omitempty"`
	EndTime             *time.Time         `json:"endTime,omitempty"`
	CompletionStartTime *time.Time         `json:"completionStartTime,omitempty"`
	Model               string             `json:"model,omitempty"`
	ModelParameters     any                `json:"modelParameters,omitempty"`
	Input               any                `json:"input,omitempty"`
	Version             string             `json:"version,omitempty"`
	Metadata            any                `json:"metadata,omitempty"`
	Output              any                `json:"output,omitempty"`
	Usage               Usage              `json:"usage,omitempty"`
	UsageDetails        map[string]int     `json:"usageDetails,omitempty"`
	CostDetails         map[string]float64 `json:"costDetails,omitempty"`
	Level               ObservationLevel   `json:"level,omitempty"`
	StatusMessage       string             `json:"statusMessage,omitempty"`
	ParentObservationID string             `json:"parentObservationId,omitempty"`
	PromptName          string             `json:"promptName,omitempty"`
	PromptVersion       int                `json:"promptVersion,omitempty"`
	ModelID             string             `json:"modelId,omitempty"`
	InputPrice          float64            `json:"inputPrice,omitempty"`
	OutputPrice         float64            `json:"outputPrice,omitempty"`
	TotalPrice          float64            `json:"totalPrice,omitempty"`
	Environment         string             `json:"environment,omitempty"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
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
