package api

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ezardev-team/langfuse-go/model"
)

// --- OpenTelemetryTracesRequest tests ---

func TestOpenTelemetryTracesRequest_Path_Default(t *testing.T) {
	req := &OpenTelemetryTracesRequest{}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/api/public/otel/v1/traces" {
		t.Errorf("expected /api/public/otel/v1/traces, got %s", path)
	}
}

func TestOpenTelemetryTracesRequest_Path_Override(t *testing.T) {
	req := &OpenTelemetryTracesRequest{
		PathOverride: "/custom/path",
	}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/custom/path" {
		t.Errorf("expected /custom/path, got %s", path)
	}
}

func TestOpenTelemetryTracesRequest_ContentType_Default(t *testing.T) {
	req := &OpenTelemetryTracesRequest{}
	if req.ContentType() != ContentTypeProtobuf {
		t.Errorf("expected %s, got %s", ContentTypeProtobuf, req.ContentType())
	}
}

func TestOpenTelemetryTracesRequest_ContentType_Override(t *testing.T) {
	req := &OpenTelemetryTracesRequest{
		ContentTypeOverride: ContentTypeJSON,
	}
	if req.ContentType() != ContentTypeJSON {
		t.Errorf("expected %s, got %s", ContentTypeJSON, req.ContentType())
	}
}

func TestOpenTelemetryTracesRequest_Encode(t *testing.T) {
	payload := []byte(`{"resourceSpans":[]}`)
	req := &OpenTelemetryTracesRequest{Body: payload}

	reader, err := req.Encode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	if string(data) != string(payload) {
		t.Errorf("expected %s, got %s", string(payload), string(data))
	}
}

// --- PromptRequest tests ---

func TestPromptRequest_Path(t *testing.T) {
	req := &PromptRequest{Name: "my-prompt"}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/api/public/v2/prompts/my-prompt" {
		t.Errorf("expected /api/public/v2/prompts/my-prompt, got %s", path)
	}
}

func TestPromptRequest_Path_EmptyName(t *testing.T) {
	req := &PromptRequest{}
	_, err := req.Path()
	if err == nil {
		t.Fatal("expected error for empty prompt name")
	}
}

func TestPromptRequest_Path_WithVersion(t *testing.T) {
	version := 3
	req := &PromptRequest{
		Name:    "my-prompt",
		Version: &version,
	}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse path: %v", err)
	}

	if u.Path != "/api/public/v2/prompts/my-prompt" {
		t.Errorf("unexpected path: %s", u.Path)
	}
	if u.Query().Get("version") != "3" {
		t.Errorf("expected version=3, got %s", u.Query().Get("version"))
	}
}

func TestPromptRequest_Path_WithLabel(t *testing.T) {
	req := &PromptRequest{
		Name:  "my-prompt",
		Label: "production",
	}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse path: %v", err)
	}

	if u.Query().Get("label") != "production" {
		t.Errorf("expected label=production, got %s", u.Query().Get("label"))
	}
}

func TestPromptRequest_Path_VersionOverridesLabel(t *testing.T) {
	version := 2
	req := &PromptRequest{
		Name:    "my-prompt",
		Version: &version,
		Label:   "production",
	}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse path: %v", err)
	}

	if u.Query().Get("version") != "2" {
		t.Errorf("expected version=2, got %s", u.Query().Get("version"))
	}
	if u.Query().Get("label") != "" {
		t.Errorf("expected label to be absent when version is set, got %s", u.Query().Get("label"))
	}
}

func TestPromptRequest_Path_WithEnvironment(t *testing.T) {
	req := &PromptRequest{
		Name:        "my-prompt",
		Environment: "staging",
	}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse path: %v", err)
	}

	if u.Query().Get("environment") != "staging" {
		t.Errorf("expected environment=staging, got %s", u.Query().Get("environment"))
	}
}

func TestPromptRequest_Path_SpecialCharacters(t *testing.T) {
	req := &PromptRequest{Name: "my prompt/test"}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(path, "my+prompt") && !strings.Contains(path, "my%20prompt") {
		t.Errorf("expected URL-encoded space in path, got %s", path)
	}
}

func TestPromptRequest_ContentType(t *testing.T) {
	req := &PromptRequest{Name: "test"}
	if req.ContentType() != "" {
		t.Errorf("expected empty content type, got %s", req.ContentType())
	}
}

func TestPromptRequest_Encode(t *testing.T) {
	req := &PromptRequest{Name: "test"}
	reader, err := req.Encode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader != nil {
		t.Error("expected nil reader for GET request")
	}
}

// --- ObservationsRequest tests ---

func TestObservationsRequest_Path_NoParams(t *testing.T) {
	req := &ObservationsRequest{}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/api/public/observations" {
		t.Errorf("expected /api/public/observations, got %s", path)
	}
}

func TestObservationsRequest_Path_WithPagination(t *testing.T) {
	page := 2
	limit := 10
	req := &ObservationsRequest{
		Page:  &page,
		Limit: &limit,
	}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse path: %v", err)
	}

	if u.Query().Get("page") != "2" {
		t.Errorf("expected page=2, got %s", u.Query().Get("page"))
	}
	if u.Query().Get("limit") != "10" {
		t.Errorf("expected limit=10, got %s", u.Query().Get("limit"))
	}
}

func TestObservationsRequest_Path_AllParams(t *testing.T) {
	page := 1
	limit := 5
	orderBy := "startTime"
	fromTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	req := &ObservationsRequest{
		Page:                &page,
		Limit:               &limit,
		Name:                "test-obs",
		UserID:              "user-123",
		Type:                model.ObservationTypeGeneration,
		TraceID:             "trace-456",
		Level:               model.ObservationLevelWarning,
		ParentObservationID: "parent-789",
		Environment:         []string{"production", "staging"},
		FromStartTime:       &fromTime,
		ToStartTime:         &toTime,
		Version:             "v1",
		Filter:              "some-filter",
		OrderBy:             &orderBy,
	}

	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse path: %v", err)
	}

	q := u.Query()

	checks := map[string]string{
		"page":                "1",
		"limit":               "5",
		"name":                "test-obs",
		"userId":              "user-123",
		"type":                "GENERATION",
		"traceId":             "trace-456",
		"level":               "WARNING",
		"parentObservationId": "parent-789",
		"version":             "v1",
		"filter":              "some-filter",
		"orderBy":             "startTime",
	}

	for key, expected := range checks {
		if got := q.Get(key); got != expected {
			t.Errorf("param %s: expected %s, got %s", key, expected, got)
		}
	}

	envValues := q["environment"]
	if len(envValues) != 2 {
		t.Errorf("expected 2 environment values, got %d", len(envValues))
	}

	if q.Get("fromStartTime") == "" {
		t.Error("expected fromStartTime to be set")
	}
	if q.Get("toStartTime") == "" {
		t.Error("expected toStartTime to be set")
	}
}

func TestObservationsRequest_Path_EmptyEnvironment(t *testing.T) {
	req := &ObservationsRequest{
		Environment: []string{"", "production", ""},
	}
	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse path: %v", err)
	}

	envValues := u.Query()["environment"]
	if len(envValues) != 1 {
		t.Errorf("expected 1 non-empty environment value, got %d: %v", len(envValues), envValues)
	}
}

func TestObservationsRequest_Path_TimeFormat(t *testing.T) {
	fromTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	req := &ObservationsRequest{
		FromStartTime: &fromTime,
	}

	path, err := req.Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse path: %v", err)
	}

	fromStr := u.Query().Get("fromStartTime")
	expected := fromTime.Format(time.RFC3339)
	if fromStr != expected {
		t.Errorf("expected fromStartTime=%s, got %s", expected, fromStr)
	}
}

func TestObservationsRequest_ContentType(t *testing.T) {
	req := &ObservationsRequest{}
	if req.ContentType() != "" {
		t.Errorf("expected empty content type, got %s", req.ContentType())
	}
}

func TestObservationsRequest_Encode(t *testing.T) {
	req := &ObservationsRequest{}
	reader, err := req.Encode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader != nil {
		t.Error("expected nil reader for GET request")
	}
}

func TestObservationsRequest_Path_PageValues(t *testing.T) {
	tests := []struct {
		page     int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{100, "100"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("page_%d", tt.page), func(t *testing.T) {
			req := &ObservationsRequest{Page: &tt.page}
			path, err := req.Path()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			u, err := url.Parse(path)
			if err != nil {
				t.Fatalf("failed to parse path: %v", err)
			}

			if got := u.Query().Get("page"); got != tt.expected {
				t.Errorf("expected page=%s, got %s", tt.expected, got)
			}
		})
	}
}
