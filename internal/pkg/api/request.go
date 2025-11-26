package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/ezardev-team/langfuse-go/model"
)

const (
	ContentTypeJSON = "application/json"
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
