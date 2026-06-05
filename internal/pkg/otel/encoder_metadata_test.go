package otel

import (
	"testing"

	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
)

// attrValue looks up the string value of an OTEL attribute by key.
func attrValue(attrs []*commonv1.KeyValue, key string) (string, bool) {
	for _, kv := range attrs {
		if kv != nil && kv.Key == key {
			return kv.Value.GetStringValue(), true
		}
	}
	return "", false
}

// TestAppendMetadataAttrs_StringNotQuoted verifies that string metadata values
// are stored verbatim (not JSON-quoted). Wrapping them in literal double quotes
// is what previously broke exact-match metadata filters and cache_key lookups
// on the Langfuse side.
func TestAppendMetadataAttrs_StringNotQuoted(t *testing.T) {
	meta := map[string]any{
		"cache_key": "abc123",
		"count":     5,
		"obj":       map[string]any{"a": 1},
	}

	attrs := appendMetadataAttrs("p.", meta, nil)

	if got, ok := attrValue(attrs, "p.cache_key"); !ok || got != "abc123" {
		t.Errorf("cache_key=%q (ok=%v), want \"abc123\" with no surrounding quotes", got, ok)
	}
	// Non-string values keep their JSON encoding.
	if got, ok := attrValue(attrs, "p.count"); !ok || got != "5" {
		t.Errorf("count=%q (ok=%v), want \"5\"", got, ok)
	}
	if got, ok := attrValue(attrs, "p.obj"); !ok || got != `{"a":1}` {
		t.Errorf("obj=%q (ok=%v), want {\"a\":1}", got, ok)
	}
}
