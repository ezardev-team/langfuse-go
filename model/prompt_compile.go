package model

import (
	"fmt"
	"regexp"
)

// PromptMessage 는 Compile 결과를 표현한다.
// langfuse 의 chat prompt 가 요구하는 role/content 구조.
type PromptMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ErrUndefinedVariable 은 prompt 가 참조하는 변수가 vars 맵에 없을 때 반환된다.
type ErrUndefinedVariable struct {
	Variable string
}

func (e *ErrUndefinedVariable) Error() string {
	return fmt.Sprintf("undefined prompt variable: %q", e.Variable)
}

// promptVarRE 매칭: {{name}} 또는 {{ name }} (앞뒤 공백 허용)
// 변수 이름은 [A-Za-z_][A-Za-z0-9_]* 만 허용.
var promptVarRE = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// Compile 은 prompt body 의 {{var}} 를 vars[var] 로 치환한다.
// vars 에 없는 변수는 ErrUndefinedVariable 반환 (fail-fast).
// vars 에는 있지만 body 가 참조 안 하는 키는 무시 (no-op).
//
// Type 분기:
//   - p.Prompt 가 string: 단일 메시지로 wrap (role="user") 반환.
//   - p.Prompt 가 []any 또는 []map[string]any: 각 메시지 content 치환 후 []PromptMessage 반환.
//   - 그 외: error.
//
// 주의: p.Prompt 의 실제 타입은 Langfuse API 응답에 따라 달라짐.
//   - chat: []map[string]any (또는 []any) — 각 항목 {"role": ..., "content": ...}
//   - text: string
func (p *Prompt) Compile(vars map[string]string) ([]PromptMessage, error) {
	if p == nil {
		return nil, fmt.Errorf("Compile on nil Prompt")
	}
	switch body := p.Prompt.(type) {
	case string:
		out, err := substituteVars(body, vars)
		if err != nil {
			return nil, err
		}
		return []PromptMessage{{Role: "user", Content: out}}, nil
	case []any:
		return compileChatMessages(body, vars)
	case []map[string]any:
		// 변환 후 위와 동일 처리
		anys := make([]any, len(body))
		for i, m := range body {
			anys[i] = m
		}
		return compileChatMessages(anys, vars)
	case nil:
		return nil, fmt.Errorf("Compile: Prompt.Prompt is nil")
	default:
		return nil, fmt.Errorf("Compile: unsupported Prompt body type %T", body)
	}
}

func compileChatMessages(items []any, vars map[string]string) ([]PromptMessage, error) {
	out := make([]PromptMessage, 0, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Compile: messages[%d] is not a map (got %T)", i, item)
		}
		role, _ := m["role"].(string)
		content, _ := m["content"].(string)
		if role == "" {
			return nil, fmt.Errorf("Compile: messages[%d].role is empty", i)
		}
		substituted, err := substituteVars(content, vars)
		if err != nil {
			return nil, fmt.Errorf("messages[%d]: %w", i, err)
		}
		out = append(out, PromptMessage{Role: role, Content: substituted})
	}
	return out, nil
}

// substituteVars 는 {{var}} 패턴을 vars[var] 로 치환한다. 미정의 변수 발견 시 fail-fast.
func substituteVars(body string, vars map[string]string) (string, error) {
	var firstUndef string
	result := promptVarRE.ReplaceAllStringFunc(body, func(match string) string {
		m := promptVarRE.FindStringSubmatch(match)
		if len(m) < 2 {
			return match
		}
		name := m[1]
		v, ok := vars[name]
		if !ok {
			if firstUndef == "" {
				firstUndef = name
			}
			return match
		}
		return v
	})
	if firstUndef != "" {
		return "", &ErrUndefinedVariable{Variable: firstUndef}
	}
	return result, nil
}
