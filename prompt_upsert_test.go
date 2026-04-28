package langfuse

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ezardev-team/langfuse-go/internal/pkg/api"
	"github.com/ezardev-team/langfuse-go/model"
)

// newTestLangfuseFromServer wires a Langfuse client to the given httptest.Server by
// pointing LANGFUSE_HOST at the mock URL. Public/secret keys are set to dummy values
// so basicAuth still works without surprising the test.
//
// The returned cleanup must be deferred to restore env state.
func newTestLangfuseFromServer(t *testing.T, srv *httptest.Server) (*Langfuse, func()) {
	t.Helper()
	origHost := os.Getenv("LANGFUSE_HOST")
	origPublic := os.Getenv("LANGFUSE_PUBLIC_KEY")
	origSecret := os.Getenv("LANGFUSE_SECRET_KEY")

	os.Setenv("LANGFUSE_HOST", srv.URL)
	os.Setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
	os.Setenv("LANGFUSE_SECRET_KEY", "sk-test")

	client := api.New()
	l := &Langfuse{client: client}

	cleanup := func() {
		if origHost == "" {
			os.Unsetenv("LANGFUSE_HOST")
		} else {
			os.Setenv("LANGFUSE_HOST", origHost)
		}
		if origPublic == "" {
			os.Unsetenv("LANGFUSE_PUBLIC_KEY")
		} else {
			os.Setenv("LANGFUSE_PUBLIC_KEY", origPublic)
		}
		if origSecret == "" {
			os.Unsetenv("LANGFUSE_SECRET_KEY")
		} else {
			os.Setenv("LANGFUSE_SECRET_KEY", origSecret)
		}
	}

	return l, cleanup
}

// captureRequest records the inbound HTTP request's method/path/body so tests can
// assert on what UpsertPrompt actually sent.
type captureRequest struct {
	Method      string
	Path        string
	ContentType string
	Body        []byte
}

// TestUpsertPrompt_Success_Chat verifies a successful chat-prompt upsert returns the
// server's prompt (id, version, etc.) and that the request reached the expected path.
func TestUpsertPrompt_Success_Chat(t *testing.T) {
	var captured captureRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Method = r.Method
		captured.Path = r.URL.Path
		captured.ContentType = r.Header.Get("Content-Type")
		captured.Body, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "prompt-id-001",
			"name":    "test-chat-prompt",
			"version": 3,
			"label":   "production",
			"prompt": []map[string]any{
				{"role": "system", "content": "You are helpful"},
			},
		})
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	chatBody := []map[string]any{
		{"role": "system", "content": "You are helpful"},
	}
	res, err := l.UpsertPrompt(context.Background(), UpsertPromptRequest{
		Name:          "test-chat-prompt",
		Type:          "chat",
		Prompt:        chatBody,
		Labels:        []string{"production"},
		Tags:          []string{"qa"},
		CommitMessage: "initial",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil prompt")
	}
	if res.Version != 3 {
		t.Errorf("Version=%d, want 3", res.Version)
	}
	if res.Name != "test-chat-prompt" {
		t.Errorf("Name=%q, want %q", res.Name, "test-chat-prompt")
	}
	if res.ID != "prompt-id-001" {
		t.Errorf("ID=%q, want %q", res.ID, "prompt-id-001")
	}

	if captured.Method != http.MethodPost {
		t.Errorf("Method=%q, want POST", captured.Method)
	}
	if captured.Path != "/api/public/v2/prompts" {
		t.Errorf("Path=%q, want %q", captured.Path, "/api/public/v2/prompts")
	}
	if !strings.HasPrefix(captured.ContentType, "application/json") {
		t.Errorf("Content-Type=%q, want application/json prefix", captured.ContentType)
	}

	// Verify body was JSON-encoded with the expected fields.
	var sent map[string]any
	if err := json.Unmarshal(captured.Body, &sent); err != nil {
		t.Fatalf("failed to parse captured body: %v", err)
	}
	if sent["name"] != "test-chat-prompt" {
		t.Errorf("sent.name=%v, want test-chat-prompt", sent["name"])
	}
	if sent["type"] != "chat" {
		t.Errorf("sent.type=%v, want chat", sent["type"])
	}
}

// TestUpsertPrompt_Success_Text verifies a text-prompt upsert returns its prompt.
func TestUpsertPrompt_Success_Text(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "prompt-text-001",
			"name":    "text-prompt",
			"version": 1,
			"prompt":  "Hello {{name}}",
		})
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	res, err := l.UpsertPrompt(context.Background(), UpsertPromptRequest{
		Name:   "text-prompt",
		Type:   "text",
		Prompt: "Hello {{name}}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Version != 1 {
		t.Errorf("Version=%d, want 1", res.Version)
	}
	if body, ok := res.Prompt.(string); !ok || body != "Hello {{name}}" {
		t.Errorf("Prompt body=%v (%T), want %q (string)", res.Prompt, res.Prompt, "Hello {{name}}")
	}
}

// TestUpsertPrompt_EmptyName verifies the call short-circuits before reaching HTTP.
func TestUpsertPrompt_EmptyName(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	_, err := l.UpsertPrompt(context.Background(), UpsertPromptRequest{
		Name:   "",
		Type:   "text",
		Prompt: "x",
	})
	if err == nil {
		t.Fatal("expected error for empty Name")
	}
	if !strings.Contains(err.Error(), "Name") {
		t.Errorf("Error()=%q, expected to mention Name", err.Error())
	}
	if hit {
		t.Error("HTTP server was hit despite empty Name")
	}
}

// TestUpsertPrompt_InvalidType verifies the type guard rejects unknown values.
func TestUpsertPrompt_InvalidType(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	_, err := l.UpsertPrompt(context.Background(), UpsertPromptRequest{
		Name:   "x",
		Type:   "invalid",
		Prompt: "x",
	})
	if err == nil {
		t.Fatal("expected error for invalid Type")
	}
	if !strings.Contains(err.Error(), "Type") {
		t.Errorf("Error()=%q, expected to mention Type", err.Error())
	}
	if hit {
		t.Error("HTTP server was hit despite invalid Type")
	}
}

// TestUpsertPrompt_NilPrompt verifies the nil-body guard rejects empty prompt content.
func TestUpsertPrompt_NilPrompt(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	_, err := l.UpsertPrompt(context.Background(), UpsertPromptRequest{
		Name:   "x",
		Type:   "text",
		Prompt: nil,
	})
	if err == nil {
		t.Fatal("expected error for nil Prompt body")
	}
	if !strings.Contains(err.Error(), "Prompt") {
		t.Errorf("Error()=%q, expected to mention Prompt", err.Error())
	}
	if hit {
		t.Error("HTTP server was hit despite nil Prompt body")
	}
}

// TestUpsertPrompt_4xx verifies the error includes the response body for 400-class.
func TestUpsertPrompt_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid prompt structure"}`))
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	_, err := l.UpsertPrompt(context.Background(), UpsertPromptRequest{
		Name:   "bad",
		Type:   "text",
		Prompt: "x",
	})
	if err == nil {
		t.Fatal("expected error on 4xx")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Error()=%q, expected to contain status %q", err.Error(), "400")
	}
	if !strings.Contains(err.Error(), "invalid prompt structure") {
		t.Errorf("Error()=%q, expected to contain server body %q", err.Error(), "invalid prompt structure")
	}
}

// TestUpsertPrompt_5xx verifies non-success status codes >=500 also surface as errors.
func TestUpsertPrompt_5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream error"))
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	_, err := l.UpsertPrompt(context.Background(), UpsertPromptRequest{
		Name:   "x",
		Type:   "text",
		Prompt: "x",
	})
	if err == nil {
		t.Fatal("expected error on 5xx")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Error()=%q, expected to contain status %q", err.Error(), "500")
	}
}

// TestUpsertPrompt_NestedPromptEnvelope verifies the response decoder also accepts
// the {"prompt": {...}} envelope form (vs. the top-level metadata form).
func TestUpsertPrompt_NestedPromptEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"prompt": {"id":"nested-1","name":"nested","version":7,"prompt":"hi"}}`))
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	res, err := l.UpsertPrompt(context.Background(), UpsertPromptRequest{
		Name:   "nested",
		Type:   "text",
		Prompt: "hi",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Version != 7 {
		t.Errorf("Version=%d, want 7", res.Version)
	}
	if res.ID != "nested-1" {
		t.Errorf("ID=%q, want %q", res.ID, "nested-1")
	}
}

// TestUpsertPrompt_ResponseShapeIsPrompt verifies the SDK preserves the model.Prompt
// type so callers can pass it straight back into Compile (Compile would itself error
// on the empty body, but the round-trip type contract should hold).
func TestUpsertPrompt_ResponseShapeIsPrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"p1","name":"p","version":1,"prompt":"only-text"}`))
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	res, err := l.UpsertPrompt(context.Background(), UpsertPromptRequest{
		Name:   "p",
		Type:   "text",
		Prompt: "only-text",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var _ *model.Prompt = res // compile-time check on type
	if res.ID != "p1" {
		t.Errorf("ID=%q, want %q", res.ID, "p1")
	}
}
