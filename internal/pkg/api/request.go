package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/ezardev-team/langfuse-go/model"
)

const (
	ContentTypeJSON     = "application/json"
	ContentTypeProtobuf = "application/x-protobuf"
)

type Request struct{}

type Ingestion struct {
	Batch []model.IngestionEvent `json:"batch"`
}

func (t *Ingestion) Path() (string, error) {
	return "/api/public/ingestion", nil
}

func (t *Ingestion) Encode() (io.Reader, error) {
	jsonBytes, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(jsonBytes), nil
}

func (t *Ingestion) ContentType() string {
	return ContentTypeJSON
}

type OpenTelemetryTracesRequest struct {
	Body                []byte
	PathOverride        string
	ContentTypeOverride string
}

func (t *OpenTelemetryTracesRequest) Path() (string, error) {
	if t.PathOverride != "" {
		return t.PathOverride, nil
	}

	return "/api/public/otel/v1/traces", nil
}

func (t *OpenTelemetryTracesRequest) Encode() (io.Reader, error) {
	return bytes.NewReader(t.Body), nil
}

func (t *OpenTelemetryTracesRequest) ContentType() string {
	if t.ContentTypeOverride != "" {
		return t.ContentTypeOverride
	}

	return ContentTypeProtobuf
}

type ObservationsRequest struct {
	Page                *int
	Limit               *int
	Name                string
	UserID              string
	Type                model.ObservationType
	TraceID             string
	Level               model.ObservationLevel
	ParentObservationID string
	Environment         []string
	FromStartTime       *time.Time
	ToStartTime         *time.Time
	Version             string
	Filter              string
	OrderBy             *string
}

func (o *ObservationsRequest) Path() (string, error) {
	queryParams := url.Values{}

	if o.Page != nil {
		queryParams.Set("page", fmt.Sprintf("%d", *o.Page))
	}

	if o.Limit != nil {
		queryParams.Set("limit", fmt.Sprintf("%d", *o.Limit))
	}

	if o.Name != "" {
		queryParams.Set("name", o.Name)
	}

	if o.UserID != "" {
		queryParams.Set("userId", o.UserID)
	}

	if o.Type != "" {
		queryParams.Set("type", string(o.Type))
	}

	if o.TraceID != "" {
		queryParams.Set("traceId", o.TraceID)
	}

	if o.Level != "" {
		queryParams.Set("level", string(o.Level))
	}

	if o.ParentObservationID != "" {
		queryParams.Set("parentObservationId", o.ParentObservationID)
	}

	for _, environment := range o.Environment {
		if environment != "" {
			queryParams.Add("environment", environment)
		}
	}

	if o.FromStartTime != nil {
		queryParams.Set("fromStartTime", o.FromStartTime.Format(time.RFC3339))
	}

	if o.ToStartTime != nil {
		queryParams.Set("toStartTime", o.ToStartTime.Format(time.RFC3339))
	}

	if o.Version != "" {
		queryParams.Set("version", o.Version)
	}

	if o.Filter != "" {
		queryParams.Set("filter", o.Filter)
	}

	if o.OrderBy != nil {
		queryParams.Set("orderBy", *o.OrderBy)
	}

	path := "/api/public/observations"
	if encodedQuery := queryParams.Encode(); encodedQuery != "" {
		path += "?" + encodedQuery
	}

	return path, nil
}

func (o *ObservationsRequest) Encode() (io.Reader, error) {
	return nil, nil
}

func (o *ObservationsRequest) ContentType() string {
	return ""
}

type PromptRequest struct {
	Name        string
	Version     *int
	Label       string
	Environment string
}

func (p *PromptRequest) Path() (string, error) {
	if p.Name == "" {
		return "", fmt.Errorf("prompt name is required")
	}

	queryParams := url.Values{}

	if p.Environment != "" {
		queryParams.Set("environment", p.Environment)
	}

	if p.Version != nil {
		queryParams.Set("version", fmt.Sprintf("%d", *p.Version))
	} else if p.Label != "" {
		queryParams.Set("label", p.Label)
	}

	path := "/api/public/v2/prompts/" + url.PathEscape(p.Name)

	if encodedQuery := queryParams.Encode(); encodedQuery != "" {
		path += "?" + encodedQuery
	}

	return path, nil
}

func (p *PromptRequest) Encode() (io.Reader, error) {
	return nil, nil
}

func (p *PromptRequest) ContentType() string {
	return ""
}
