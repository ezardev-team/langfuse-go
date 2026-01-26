package api

import (
	"context"
	"os"
	"testing"

	"github.com/ezardev-team/langfuse-go/model"
)

// skipIfNoEnv skips the test if the required Langfuse environment variables are not set.
func skipIfNoEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("LANGFUSE_PUBLIC_KEY") == "" || os.Getenv("LANGFUSE_SECRET_KEY") == "" {
		t.Skip("LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY must be set")
	}
}

func TestNew(t *testing.T) {
	skipIfNoEnv(t)

	client := New()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.restClient == nil {
		t.Fatal("expected non-nil restClient")
	}
}

func TestNew_DefaultHost(t *testing.T) {
	publicKey := os.Getenv("LANGFUSE_PUBLIC_KEY")
	secretKey := os.Getenv("LANGFUSE_SECRET_KEY")
	origHost := os.Getenv("LANGFUSE_HOST")

	// Temporarily unset LANGFUSE_HOST to test default
	os.Unsetenv("LANGFUSE_HOST")
	defer func() {
		if origHost != "" {
			os.Setenv("LANGFUSE_HOST", origHost)
		}
		os.Setenv("LANGFUSE_PUBLIC_KEY", publicKey)
		os.Setenv("LANGFUSE_SECRET_KEY", secretKey)
	}()

	client := New()
	if client == nil {
		t.Fatal("expected non-nil client when LANGFUSE_HOST is unset")
	}
}

func TestOpenTelemetryTraces(t *testing.T) {
	skipIfNoEnv(t)

	client := New()
	ctx := context.Background()

	// Send a minimal valid OTLP JSON body
	body := []byte(`{"resourceSpans":[]}`)
	req := &OpenTelemetryTracesRequest{
		Body:                body,
		ContentTypeOverride: ContentTypeJSON,
	}

	res := &OpenTelemetryResponse{}
	err := client.OpenTelemetryTraces(ctx, req, res)
	if err != nil {
		t.Fatalf("OpenTelemetryTraces failed: %v", err)
	}

	t.Logf("OpenTelemetryTraces response code=%d", res.Code)
}

func TestPrompt(t *testing.T) {
	skipIfNoEnv(t)

	// This test requires a prompt named "test-prompt" to exist in the Langfuse project.
	// If the prompt does not exist, the test will report the error but not fail fatally.
	promptName := os.Getenv("LANGFUSE_TEST_PROMPT_NAME")
	if promptName == "" {
		promptName = "test-prompt"
	}

	client := New()
	ctx := context.Background()

	req := &PromptRequest{
		Name: promptName,
	}

	res := &PromptResponse{}
	err := client.Prompt(ctx, req, res)
	if err != nil {
		t.Logf("Prompt request failed (prompt '%s' may not exist): %v", promptName, err)
		t.SkipNow()
	}

	if !res.IsSuccess() {
		t.Logf("Prompt response was not successful: code=%d, rawBody=%v", res.Code, res.RawBody)
		t.SkipNow()
	}

	t.Logf("Prompt response: code=%d, name=%s, version=%d", res.Code, res.Prompt.Name, res.Prompt.Version)
}

func TestPrompt_WithVersion(t *testing.T) {
	skipIfNoEnv(t)

	promptName := os.Getenv("LANGFUSE_TEST_PROMPT_NAME")
	if promptName == "" {
		promptName = "test-prompt"
	}

	client := New()
	ctx := context.Background()

	version := 1
	req := &PromptRequest{
		Name:    promptName,
		Version: &version,
	}

	res := &PromptResponse{}
	err := client.Prompt(ctx, req, res)
	if err != nil {
		t.Logf("Prompt with version request failed (prompt '%s' v%d may not exist): %v", promptName, version, err)
		t.SkipNow()
	}

	if res.IsSuccess() {
		t.Logf("Prompt with version: code=%d, name=%s, version=%d", res.Code, res.Prompt.Name, res.Prompt.Version)
	}
}

func TestPrompt_WithLabel(t *testing.T) {
	skipIfNoEnv(t)

	promptName := os.Getenv("LANGFUSE_TEST_PROMPT_NAME")
	if promptName == "" {
		promptName = "test-prompt"
	}

	client := New()
	ctx := context.Background()

	req := &PromptRequest{
		Name:  promptName,
		Label: "latest",
	}

	res := &PromptResponse{}
	err := client.Prompt(ctx, req, res)
	if err != nil {
		t.Logf("Prompt with label request failed: %v", err)
		t.SkipNow()
	}

	if res.IsSuccess() {
		t.Logf("Prompt with label: code=%d, name=%s, label=%s", res.Code, res.Prompt.Name, res.Prompt.Label)
	}
}

func TestObservations(t *testing.T) {
	skipIfNoEnv(t)

	client := New()
	ctx := context.Background()

	limit := 5
	page := 1
	req := &ObservationsRequest{
		Page:  &page,
		Limit: &limit,
	}

	res := &ObservationsResponse{}
	err := client.Observations(ctx, req, res)
	if err != nil {
		t.Fatalf("Observations failed: %v", err)
	}

	if !res.IsSuccess() {
		t.Fatalf("expected success response, got code=%d", res.Code)
	}

	t.Logf("Observations: code=%d, data_count=%d, total_items=%d, total_pages=%d",
		res.Code, len(res.Data), res.Meta.TotalItems, res.Meta.TotalPages)
}

func TestObservations_WithTypeFilter(t *testing.T) {
	skipIfNoEnv(t)

	client := New()
	ctx := context.Background()

	limit := 5
	page := 1
	req := &ObservationsRequest{
		Page:  &page,
		Limit: &limit,
		Type:  model.ObservationTypeGeneration,
	}

	res := &ObservationsResponse{}
	err := client.Observations(ctx, req, res)
	if err != nil {
		t.Fatalf("Observations with type filter failed: %v", err)
	}

	if !res.IsSuccess() {
		t.Fatalf("expected success response, got code=%d", res.Code)
	}

	for _, obs := range res.Data {
		if obs.Type != model.ObservationTypeGeneration {
			t.Errorf("expected type GENERATION, got %s", obs.Type)
		}
	}

	t.Logf("Observations (GENERATION): code=%d, data_count=%d", res.Code, len(res.Data))
}

func TestObservations_WithNameFilter(t *testing.T) {
	skipIfNoEnv(t)

	client := New()
	ctx := context.Background()

	limit := 5
	page := 1
	req := &ObservationsRequest{
		Page:  &page,
		Limit: &limit,
		Name:  "api-test-multi-generation",
	}

	res := &ObservationsResponse{}
	err := client.Observations(ctx, req, res)
	if err != nil {
		t.Fatalf("Observations with name filter failed: %v", err)
	}

	if !res.IsSuccess() {
		t.Fatalf("expected success response, got code=%d", res.Code)
	}

	t.Logf("Observations (by name): code=%d, data_count=%d", res.Code, len(res.Data))
}

func TestBasicAuth(t *testing.T) {
	got := basicAuth("pk-lf-1234", "sk-lf-5678")
	// "pk-lf-1234:sk-lf-5678" -> base64 = "cGstbGYtMTIzNDpzay1sZi01Njc4"
	expected := "Basic cGstbGYtMTIzNDpzay1sZi01Njc4"
	if got != expected {
		t.Errorf("basicAuth mismatch:\n  got:  %s\n  want: %s", got, expected)
	}
}

func TestBasicAuth_Empty(t *testing.T) {
	got := basicAuth("", "")
	// ":" -> base64 = "Og=="
	expected := "Basic Og=="
	if got != expected {
		t.Errorf("basicAuth empty mismatch:\n  got:  %s\n  want: %s", got, expected)
	}
}
