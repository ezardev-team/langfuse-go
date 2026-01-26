package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// --- Response tests ---

func TestResponse_IsSuccess(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{http.StatusOK, true},
		{http.StatusCreated, true},
		{http.StatusNoContent, true},
		{http.StatusMultipleChoices, true},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			r := &Response{Code: tt.code}
			if r.IsSuccess() != tt.expected {
				t.Errorf("code %d: expected IsSuccess=%v, got %v", tt.code, tt.expected, r.IsSuccess())
			}
		})
	}
}

func TestResponse_SetStatusCode(t *testing.T) {
	r := &Response{}
	err := r.SetStatusCode(http.StatusOK)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Code != http.StatusOK {
		t.Errorf("expected code %d, got %d", http.StatusOK, r.Code)
	}
}

func TestResponse_SetBody(t *testing.T) {
	r := &Response{}
	body := strings.NewReader(`{"successes":[],"errors":[]}`)
	err := r.SetBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.RawBody == nil {
		t.Fatal("expected non-nil RawBody")
	}
	if !strings.Contains(*r.RawBody, "successes") {
		t.Errorf("RawBody does not contain expected content: %s", *r.RawBody)
	}
}

func TestResponse_AcceptContentType(t *testing.T) {
	r := &Response{}
	if r.AcceptContentType() != ContentTypeJSON {
		t.Errorf("expected %s, got %s", ContentTypeJSON, r.AcceptContentType())
	}
}

func TestResponse_Decode(t *testing.T) {
	jsonBody := `{"successes":[{"id":"abc","status":200}],"errors":[]}`
	r := &Response{}
	err := r.Decode(strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Successes) != 1 {
		t.Fatalf("expected 1 success, got %d", len(r.Successes))
	}
	if r.Successes[0].ID != "abc" {
		t.Errorf("expected success ID abc, got %s", r.Successes[0].ID)
	}
	if r.Successes[0].Status != 200 {
		t.Errorf("expected success status 200, got %d", r.Successes[0].Status)
	}
}

func TestResponse_Decode_WithErrors(t *testing.T) {
	jsonBody := `{"successes":[],"errors":[{"id":"err1","status":400,"message":"bad input","error":"validation_error"}]}`
	r := &Response{}
	err := r.Decode(strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(r.Errors))
	}
	if r.Errors[0].ID != "err1" {
		t.Errorf("expected error ID err1, got %s", r.Errors[0].ID)
	}
	if r.Errors[0].Message != "bad input" {
		t.Errorf("expected error message 'bad input', got %s", r.Errors[0].Message)
	}
}

func TestResponse_SetHeaders(t *testing.T) {
	r := &Response{}
	err := r.SetHeaders(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- OpenTelemetryResponse tests ---

func TestOpenTelemetryResponse_IsSuccess(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{http.StatusOK, true},
		{http.StatusBadRequest, false},
	}

	for _, tt := range tests {
		r := &OpenTelemetryResponse{Code: tt.code}
		if r.IsSuccess() != tt.expected {
			t.Errorf("code %d: expected %v, got %v", tt.code, tt.expected, r.IsSuccess())
		}
	}
}

func TestOpenTelemetryResponse_SetStatusCode(t *testing.T) {
	r := &OpenTelemetryResponse{}
	err := r.SetStatusCode(http.StatusAccepted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Code != http.StatusAccepted {
		t.Errorf("expected %d, got %d", http.StatusAccepted, r.Code)
	}
}

func TestOpenTelemetryResponse_SetBody(t *testing.T) {
	r := &OpenTelemetryResponse{}
	body := strings.NewReader(`ok`)
	err := r.SetBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.RawBody == nil || *r.RawBody != "ok" {
		t.Errorf("expected RawBody=ok, got %v", r.RawBody)
	}
}

func TestOpenTelemetryResponse_AcceptContentType(t *testing.T) {
	r := &OpenTelemetryResponse{}
	if r.AcceptContentType() != "" {
		t.Errorf("expected empty accept content type, got %s", r.AcceptContentType())
	}
}

func TestOpenTelemetryResponse_Decode(t *testing.T) {
	r := &OpenTelemetryResponse{}
	body := strings.NewReader(`some response body`)
	err := r.Decode(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.RawBody == nil {
		t.Fatal("expected RawBody to be set")
	}
	if *r.RawBody != "some response body" {
		t.Errorf("expected 'some response body', got %s", *r.RawBody)
	}
}

func TestOpenTelemetryResponse_Decode_WithExistingRawBody(t *testing.T) {
	existing := "already set"
	r := &OpenTelemetryResponse{RawBody: &existing}
	body := strings.NewReader(`new body`)
	err := r.Decode(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// RawBody should NOT be overwritten if already set
	if *r.RawBody != "already set" {
		t.Errorf("expected RawBody to remain 'already set', got %s", *r.RawBody)
	}
}

func TestOpenTelemetryResponse_SetHeaders(t *testing.T) {
	r := &OpenTelemetryResponse{}
	err := r.SetHeaders(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- PromptResponse tests ---

func TestPromptResponse_IsSuccess(t *testing.T) {
	r := &PromptResponse{Code: http.StatusOK}
	if !r.IsSuccess() {
		t.Error("expected IsSuccess for 200")
	}

	r2 := &PromptResponse{Code: http.StatusNotFound}
	if r2.IsSuccess() {
		t.Error("expected not success for 404")
	}
}

func TestPromptResponse_SetStatusCode(t *testing.T) {
	r := &PromptResponse{}
	err := r.SetStatusCode(http.StatusOK)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, r.Code)
	}
}

func TestPromptResponse_SetBody(t *testing.T) {
	r := &PromptResponse{}
	body := strings.NewReader(`{"name":"test"}`)
	err := r.SetBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.RawBody == nil {
		t.Fatal("expected non-nil RawBody")
	}
}

func TestPromptResponse_AcceptContentType(t *testing.T) {
	r := &PromptResponse{}
	if r.AcceptContentType() != ContentTypeJSON {
		t.Errorf("expected %s, got %s", ContentTypeJSON, r.AcceptContentType())
	}
}

func TestPromptResponse_Decode_DirectPromptObject(t *testing.T) {
	jsonBody := `{"id":"p1","name":"my-prompt","version":2,"prompt":"Hello {{name}}"}`
	r := &PromptResponse{}
	err := r.Decode(strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Prompt.Name != "my-prompt" {
		t.Errorf("expected name my-prompt, got %s", r.Prompt.Name)
	}
	if r.Prompt.Version != 2 {
		t.Errorf("expected version 2, got %d", r.Prompt.Version)
	}
}

func TestPromptResponse_Decode_NestedPromptObject(t *testing.T) {
	jsonBody := `{"prompt":{"name":"nested-prompt","version":1,"prompt":"content here"}}`
	r := &PromptResponse{}
	err := r.Decode(strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Prompt.Name != "nested-prompt" {
		t.Errorf("expected name nested-prompt, got %s", r.Prompt.Name)
	}
}

func TestPromptResponse_Decode_WithExistingRawBody(t *testing.T) {
	existing := "already set"
	r := &PromptResponse{RawBody: &existing}
	jsonBody := `{"id":"p1","name":"test-prompt","version":1}`
	err := r.Decode(strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// RawBody should NOT be overwritten
	if *r.RawBody != "already set" {
		t.Errorf("expected RawBody to remain 'already set', got %s", *r.RawBody)
	}
}

func TestPromptResponse_SetHeaders(t *testing.T) {
	r := &PromptResponse{}
	err := r.SetHeaders(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ObservationsResponse tests ---

func TestObservationsResponse_Decode(t *testing.T) {
	jsonBody := `{
		"data": [
			{"id": "obs-1", "type": "GENERATION", "name": "gen-1"},
			{"id": "obs-2", "type": "SPAN", "name": "span-1"}
		],
		"meta": {
			"page": 1,
			"limit": 10,
			"totalItems": 2,
			"totalPages": 1
		}
	}`

	r := &ObservationsResponse{}
	err := r.Decode(strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Data) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(r.Data))
	}
	if r.Data[0].ID != "obs-1" {
		t.Errorf("expected first observation ID obs-1, got %s", r.Data[0].ID)
	}
	if r.Meta.TotalItems != 2 {
		t.Errorf("expected totalItems=2, got %d", r.Meta.TotalItems)
	}
	if r.Meta.TotalPages != 1 {
		t.Errorf("expected totalPages=1, got %d", r.Meta.TotalPages)
	}
}

func TestObservationsResponse_Decode_SetsRawBody(t *testing.T) {
	jsonBody := `{"data":[],"meta":{"page":1,"limit":10,"totalItems":0,"totalPages":0}}`
	r := &ObservationsResponse{}
	err := r.Decode(strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.RawBody == nil {
		t.Fatal("expected RawBody to be set")
	}
}

func TestObservationsResponse_Decode_WithExistingRawBody(t *testing.T) {
	existing := "existing"
	r := &ObservationsResponse{}
	r.RawBody = &existing

	jsonBody := `{"data":[],"meta":{"page":1,"limit":10,"totalItems":0,"totalPages":0}}`
	err := r.Decode(strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *r.RawBody != "existing" {
		t.Errorf("expected RawBody to remain 'existing', got %s", *r.RawBody)
	}
}

// --- hasPromptMetadata tests ---

func TestHasPromptMetadata_WithID(t *testing.T) {
	envelope := map[string]json.RawMessage{
		"id": json.RawMessage(`"abc"`),
	}
	if !hasPromptMetadata(envelope) {
		t.Error("expected true for envelope with 'id' key")
	}
}

func TestHasPromptMetadata_WithName(t *testing.T) {
	envelope := map[string]json.RawMessage{
		"name": json.RawMessage(`"test"`),
	}
	if !hasPromptMetadata(envelope) {
		t.Error("expected true for envelope with 'name' key")
	}
}

func TestHasPromptMetadata_WithoutKnownKeys(t *testing.T) {
	envelope := map[string]json.RawMessage{
		"prompt":  json.RawMessage(`"hello"`),
		"unknown": json.RawMessage(`"value"`),
	}
	if hasPromptMetadata(envelope) {
		t.Error("expected false for envelope without known metadata keys")
	}
}

func TestHasPromptMetadata_Empty(t *testing.T) {
	envelope := map[string]json.RawMessage{}
	if hasPromptMetadata(envelope) {
		t.Error("expected false for empty envelope")
	}
}

func TestHasPromptMetadata_AllKeys(t *testing.T) {
	keys := []string{"id", "name", "version", "label", "environment", "config", "metadata", "createdAt", "updatedAt"}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			envelope := map[string]json.RawMessage{
				key: json.RawMessage(`"test"`),
			}
			if !hasPromptMetadata(envelope) {
				t.Errorf("expected true for envelope with '%s' key", key)
			}
		})
	}
}

// --- Decode edge cases ---

func TestResponse_Decode_InvalidJSON(t *testing.T) {
	r := &Response{}
	err := r.Decode(strings.NewReader(`{invalid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestPromptResponse_Decode_InvalidJSON(t *testing.T) {
	r := &PromptResponse{}
	err := r.Decode(strings.NewReader(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestObservationsResponse_Decode_InvalidJSON(t *testing.T) {
	r := &ObservationsResponse{}
	err := r.Decode(bytes.NewReader([]byte(`not json`)))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
