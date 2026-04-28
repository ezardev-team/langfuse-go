package model

import (
	"errors"
	"strings"
	"testing"
)

// TestPromptCompile_StringBody verifies a text-prompt body wraps into a single user message.
func TestPromptCompile_StringBody(t *testing.T) {
	p := &Prompt{Prompt: "Hello {{name}}"}
	msgs, err := p.Compile(map[string]string{"name": "World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected role=user, got %q", msgs[0].Role)
	}
	if msgs[0].Content != "Hello World" {
		t.Errorf("expected content=%q, got %q", "Hello World", msgs[0].Content)
	}
}

// TestPromptCompile_ChatBody_AnySlice verifies []any input (Langfuse default JSON unmarshal shape).
func TestPromptCompile_ChatBody_AnySlice(t *testing.T) {
	p := &Prompt{
		Prompt: []any{
			map[string]any{"role": "system", "content": "You are {{role_desc}}"},
			map[string]any{"role": "user", "content": "Question: {{q}}"},
		},
	}
	msgs, err := p.Compile(map[string]string{"role_desc": "an analyst", "q": "Buy or sell?"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[0].Content != "You are an analyst" {
		t.Errorf("msg[0]: got role=%q content=%q", msgs[0].Role, msgs[0].Content)
	}
	if msgs[1].Role != "user" || msgs[1].Content != "Question: Buy or sell?" {
		t.Errorf("msg[1]: got role=%q content=%q", msgs[1].Role, msgs[1].Content)
	}
}

// TestPromptCompile_ChatBody_MapSlice verifies []map[string]any input (typed shape).
func TestPromptCompile_ChatBody_MapSlice(t *testing.T) {
	p := &Prompt{
		Prompt: []map[string]any{
			{"role": "system", "content": "Role: {{r}}"},
			{"role": "user", "content": "Hi {{name}}!"},
		},
	}
	msgs, err := p.Compile(map[string]string{"r": "bot", "name": "Alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "Role: bot" {
		t.Errorf("msg[0].content=%q, want %q", msgs[0].Content, "Role: bot")
	}
	if msgs[1].Content != "Hi Alice!" {
		t.Errorf("msg[1].content=%q, want %q", msgs[1].Content, "Hi Alice!")
	}
}

// TestPromptCompile_UndefinedVariable verifies fail-fast on missing var with typed error.
func TestPromptCompile_UndefinedVariable(t *testing.T) {
	p := &Prompt{Prompt: "Hello {{missing}}"}
	_, err := p.Compile(map[string]string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *ErrUndefinedVariable
	if !errors.As(err, &target) {
		t.Fatalf("expected *ErrUndefinedVariable, got %T: %v", err, err)
	}
	if target.Variable != "missing" {
		t.Errorf("Variable=%q, want %q", target.Variable, "missing")
	}
	if !strings.Contains(target.Error(), "missing") {
		t.Errorf("Error() = %q, expected to contain %q", target.Error(), "missing")
	}
}

// TestPromptCompile_ExtraVariablesIgnored verifies unused vars are silently ignored.
func TestPromptCompile_ExtraVariablesIgnored(t *testing.T) {
	p := &Prompt{Prompt: "Hello {{name}}"}
	msgs, err := p.Compile(map[string]string{
		"name":   "World",
		"unused": "should-be-ignored",
		"extra":  "also-ignored",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgs[0].Content != "Hello World" {
		t.Errorf("got %q, want %q", msgs[0].Content, "Hello World")
	}
}

// TestPromptCompile_NilPrompt verifies error on nil receiver.
func TestPromptCompile_NilPrompt(t *testing.T) {
	var p *Prompt
	_, err := p.Compile(nil)
	if err == nil {
		t.Fatal("expected error on nil Prompt")
	}
	if !strings.Contains(err.Error(), "nil Prompt") {
		t.Errorf("Error()=%q, expected to contain %q", err.Error(), "nil Prompt")
	}
}

// TestPromptCompile_NilBody verifies error when Prompt.Prompt body is nil.
func TestPromptCompile_NilBody(t *testing.T) {
	p := &Prompt{Prompt: nil}
	_, err := p.Compile(nil)
	if err == nil {
		t.Fatal("expected error on nil body")
	}
	if !strings.Contains(err.Error(), "is nil") {
		t.Errorf("Error()=%q, expected to contain %q", err.Error(), "is nil")
	}
}

// TestPromptCompile_UnsupportedBodyType verifies error on unexpected body type.
func TestPromptCompile_UnsupportedBodyType(t *testing.T) {
	p := &Prompt{Prompt: 42}
	_, err := p.Compile(nil)
	if err == nil {
		t.Fatal("expected error for unsupported body type")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("Error()=%q, expected to contain %q", err.Error(), "unsupported")
	}
}

// TestPromptCompile_WhitespaceInVarPattern verifies both `{{ name }}` and `{{name}}` match.
func TestPromptCompile_WhitespaceInVarPattern(t *testing.T) {
	p := &Prompt{Prompt: "{{ greeting }} {{name}}!"}
	msgs, err := p.Compile(map[string]string{"greeting": "Hi", "name": "X"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgs[0].Content != "Hi X!" {
		t.Errorf("got %q, want %q", msgs[0].Content, "Hi X!")
	}
}

// TestPromptCompile_ChatRoleMissing verifies error when a chat message lacks role.
func TestPromptCompile_ChatRoleMissing(t *testing.T) {
	p := &Prompt{
		Prompt: []any{
			map[string]any{"role": "", "content": "hi"},
		},
	}
	_, err := p.Compile(nil)
	if err == nil {
		t.Fatal("expected error for empty role")
	}
	if !strings.Contains(err.Error(), "role") {
		t.Errorf("Error()=%q, expected to contain %q", err.Error(), "role")
	}
}

// TestPromptCompile_FirstUndefinedReportedOnly verifies only the first undefined var is reported.
func TestPromptCompile_FirstUndefinedReportedOnly(t *testing.T) {
	p := &Prompt{Prompt: "{{a}} and {{b}}"}
	_, err := p.Compile(map[string]string{})
	if err == nil {
		t.Fatal("expected error")
	}
	var target *ErrUndefinedVariable
	if !errors.As(err, &target) {
		t.Fatalf("expected *ErrUndefinedVariable, got %T: %v", err, err)
	}
	if target.Variable != "a" {
		t.Errorf("expected first undefined=%q, got %q", "a", target.Variable)
	}
}

// TestPromptCompile_ChatNonMapItem verifies error when message item is not a map.
func TestPromptCompile_ChatNonMapItem(t *testing.T) {
	p := &Prompt{
		Prompt: []any{"plain string is invalid"},
	}
	_, err := p.Compile(nil)
	if err == nil {
		t.Fatal("expected error for non-map item")
	}
	if !strings.Contains(err.Error(), "not a map") {
		t.Errorf("Error()=%q, expected to contain %q", err.Error(), "not a map")
	}
}

// TestPromptCompile_EmptyChat verifies an empty []any returns an empty result with no error.
func TestPromptCompile_EmptyChat(t *testing.T) {
	p := &Prompt{Prompt: []any{}}
	msgs, err := p.Compile(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

// TestPromptCompile_NoVariables verifies a body without {{}} markers passes through unchanged.
func TestPromptCompile_NoVariables(t *testing.T) {
	p := &Prompt{Prompt: "static text without markers"}
	msgs, err := p.Compile(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgs[0].Content != "static text without markers" {
		t.Errorf("got %q, want %q", msgs[0].Content, "static text without markers")
	}
}

// TestPromptCompile_ChatErrorIncludesIndex verifies chat errors include the message index.
func TestPromptCompile_ChatErrorIncludesIndex(t *testing.T) {
	p := &Prompt{
		Prompt: []any{
			map[string]any{"role": "system", "content": "ok"},
			map[string]any{"role": "user", "content": "{{missing}}"},
		},
	}
	_, err := p.Compile(map[string]string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "messages[1]") {
		t.Errorf("Error()=%q, expected to contain index %q", err.Error(), "messages[1]")
	}
	// Wrapped *ErrUndefinedVariable should still be reachable via errors.As.
	var target *ErrUndefinedVariable
	if !errors.As(err, &target) {
		t.Fatalf("expected wrapped *ErrUndefinedVariable, got %T: %v", err, err)
	}
	if target.Variable != "missing" {
		t.Errorf("Variable=%q, want %q", target.Variable, "missing")
	}
}
