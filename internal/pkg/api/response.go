package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/ezardev-team/langfuse-go/model"
	"github.com/henomis/restclientgo"
)

type Response struct {
	Code      int       `json:"-"`
	RawBody   *string   `json:"-"`
	Successes []Success `json:"successes"`
	Errors    []Error   `json:"errors"`
}

type Success struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

type Error struct {
	ID      string `json:"id"`
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

type PromptResponse struct {
	Code    int     `json:"-"`
	RawBody *string `json:"-"`
	Prompt  model.Prompt
}

func (r *Response) IsSuccess() bool {
	return r.Code < http.StatusBadRequest
}

func (r *Response) SetStatusCode(code int) error {
	r.Code = code
	return nil
}

func (r *Response) SetBody(body io.Reader) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	s := string(b)
	r.RawBody = &s

	return nil
}

func (r *Response) AcceptContentType() string {
	return ContentTypeJSON
}

func (r *Response) Decode(body io.Reader) error {
	return json.NewDecoder(body).Decode(r)
}

func (r *Response) SetHeaders(_ restclientgo.Headers) error {
	return nil
}

type IngestionResponse struct {
	Response
}

func (r *PromptResponse) IsSuccess() bool {
	return r.Code < http.StatusBadRequest
}

func (r *PromptResponse) SetStatusCode(code int) error {
	r.Code = code
	return nil
}

func (r *PromptResponse) SetBody(body io.Reader) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	s := string(b)
	r.RawBody = &s

	return nil
}

func (r *PromptResponse) AcceptContentType() string {
	return ContentTypeJSON
}

func (r *PromptResponse) Decode(body io.Reader) error {
	rawBody, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	if r.RawBody == nil {
		bodyString := string(rawBody)
		r.RawBody = &bodyString
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		return err
	}

	// If the API returns the prompt object directly (id/name/label at the top
	// level), decode straight into the prompt model to support prompt content
	// being a plain string or structured payload.
	if hasPromptMetadata(envelope) {
		return json.Unmarshal(rawBody, &r.Prompt)
	}

	// Otherwise look for a nested prompt object.
	if promptRaw, ok := envelope["prompt"]; ok {
		return json.Unmarshal(promptRaw, &r.Prompt)
	}

	// Fallback: try to decode the entire body into the prompt.
	return json.Unmarshal(rawBody, &r.Prompt)
}

func (r *PromptResponse) SetHeaders(_ restclientgo.Headers) error {
	return nil
}

func hasPromptMetadata(envelope map[string]json.RawMessage) bool {
	keys := []string{
		"id",
		"name",
		"version",
		"label",
		"environment",
		"config",
		"metadata",
		"createdAt",
		"updatedAt",
	}

	for _, key := range keys {
		if _, ok := envelope[key]; ok {
			return true
		}
	}

	return false
}
