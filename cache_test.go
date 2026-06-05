package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestFindCachedGeneration_HitsV2Endpoint verifies the cache lookup uses the v2
// observations endpoint with cursor-based pagination, requests the metadata
// field group with expandMetadata, and round-trips the v2 response shape into
// ObservationView correctly.
func TestFindCachedGeneration_HitsV2Endpoint(t *testing.T) {
	var capturedPath string
	var capturedQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.Query()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":                "obs-cached-1",
					"projectId":         "proj-1",
					"type":              "GENERATION",
					"name":              "llm-call",
					"providedModelName": "gpt-4o",
					"output":            map[string]any{"answer": "42"},
					// Stored double-quoted, as legacy traces persisted scalar
					// metadata; the lookup must still treat this as a hit.
					"metadata":     map[string]any{cacheMetadataKey: `"key-abc"`},
					"usageDetails": map[string]int{"input": 10, "output": 20, "total": 30},
				},
			},
			"meta": map[string]any{"cursor": nil},
		})
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	obs, err := l.FindCachedGeneration(context.Background(), "key-abc", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs == nil {
		t.Fatal("expected cache hit, got nil")
	}
	if obs.ID != "obs-cached-1" {
		t.Errorf("ID=%q, want obs-cached-1", obs.ID)
	}
	if obs.Model != "gpt-4o" {
		t.Errorf("Model=%q, want gpt-4o (from providedModelName)", obs.Model)
	}
	if obs.UsageDetails["total"] != 30 {
		t.Errorf("UsageDetails[total]=%d, want 30", obs.UsageDetails["total"])
	}

	if capturedPath != "/api/public/v2/observations" {
		t.Errorf("Path=%q, want /api/public/v2/observations", capturedPath)
	}
	if got := capturedQuery.Get("limit"); got != "1" {
		t.Errorf("limit=%q, want 1", got)
	}
	if got := capturedQuery.Get("cursor"); got != "" {
		t.Errorf("cursor should be empty on first call, got %q", got)
	}
	if got := capturedQuery.Get("page"); got != "" {
		t.Errorf("page param must not be sent on v2, got %q", got)
	}
	if got := capturedQuery.Get("expandMetadata"); got != cacheMetadataKey {
		t.Errorf("expandMetadata=%q, want %q", got, cacheMetadataKey)
	}
	fields := capturedQuery.Get("fields")
	for _, group := range []string{"metadata", "io", "model"} {
		if !strings.Contains(fields, group) {
			t.Errorf("fields=%q must contain %q", fields, group)
		}
	}

	filter := capturedQuery.Get("filter")
	if filter == "" {
		t.Fatal("expected filter query param to be set")
	}
	var conds []map[string]any
	if err := json.Unmarshal([]byte(filter), &conds); err != nil {
		t.Fatalf("filter not valid JSON: %v (raw=%s)", err, filter)
	}
	if len(conds) < 2 {
		t.Fatalf("expected at least 2 filter conditions, got %d", len(conds))
	}
	// First filter: type=GENERATION
	if conds[0]["column"] != "type" || conds[0]["value"] != "GENERATION" {
		t.Errorf("first filter not type=GENERATION: %+v", conds[0])
	}
	// Second filter: metadata.cache_key matches key-abc (quote-tolerant)
	if conds[1]["column"] != "metadata" || conds[1]["key"] != cacheMetadataKey ||
		conds[1]["operator"] != "matches" || conds[1]["value"] != "key-abc" {
		t.Errorf("metadata filter wrong: %+v", conds[1])
	}
}

// TestFindCachedGeneration_HitUnquotedMetadata verifies the lookup also treats a
// clean (unquoted) stored cache_key as a hit, i.e. traces written after the
// encoder stopped JSON-quoting scalar metadata values.
func TestFindCachedGeneration_HitUnquotedMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":       "obs-cached-2",
					"type":     "GENERATION",
					"metadata": map[string]any{cacheMetadataKey: "key-abc"},
				},
			},
			"meta": map[string]any{"cursor": nil},
		})
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	obs, err := l.FindCachedGeneration(context.Background(), "key-abc", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs == nil {
		t.Fatal("expected cache hit for unquoted metadata, got nil")
	}
	if obs.ID != "obs-cached-2" {
		t.Errorf("ID=%q, want obs-cached-2", obs.ID)
	}
}

// TestFindCachedGeneration_Miss verifies a miss (cache_key in response does not
// match the requested key) returns nil rather than the wrong observation.
func TestFindCachedGeneration_Miss(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":       "obs-other",
					"type":     "GENERATION",
					"metadata": map[string]any{cacheMetadataKey: "different-key"},
				},
			},
			"meta": map[string]any{"cursor": nil},
		})
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	obs, err := l.FindCachedGeneration(context.Background(), "key-abc", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs != nil {
		t.Errorf("expected miss (nil), got %+v", obs)
	}
}

// TestFindCachedGeneration_EmptyData verifies an empty data array returns nil.
func TestFindCachedGeneration_EmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{},
			"meta": map[string]any{"cursor": nil},
		})
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	obs, err := l.FindCachedGeneration(context.Background(), "key-abc", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs != nil {
		t.Errorf("expected nil, got %+v", obs)
	}
}

// TestFindCachedGeneration_NameOption verifies the optional Name filter is added
// to the filter JSON.
func TestFindCachedGeneration_NameOption(t *testing.T) {
	var capturedFilter string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedFilter = r.URL.Query().Get("filter")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{},
			"meta": map[string]any{"cursor": nil},
		})
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	_, err := l.FindCachedGeneration(context.Background(), "key-abc", &GenerationCacheOptions{Name: "my-fn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var conds []map[string]any
	if err := json.Unmarshal([]byte(capturedFilter), &conds); err != nil {
		t.Fatalf("filter not JSON: %v (raw=%s)", err, capturedFilter)
	}
	var nameCond map[string]any
	for _, c := range conds {
		if c["column"] == "name" {
			nameCond = c
			break
		}
	}
	if nameCond == nil {
		t.Fatalf("expected a name= filter, got: %+v", conds)
	}
	if nameCond["value"] != "my-fn" {
		t.Errorf("name filter value=%v, want my-fn", nameCond["value"])
	}
}

// TestFindCachedGeneration_NonSuccessStatus verifies a non-2xx response surfaces
// as an error, not a silent miss.
func TestFindCachedGeneration_NonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}))
	defer srv.Close()

	l, cleanup := newTestLangfuseFromServer(t, srv)
	defer cleanup()

	_, err := l.FindCachedGeneration(context.Background(), "key-abc", nil)
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
}

func TestFindCachedGeneration_EmptyKey(t *testing.T) {
	l := &Langfuse{}
	_, err := l.FindCachedGeneration(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty cache key, got nil")
	}
}
