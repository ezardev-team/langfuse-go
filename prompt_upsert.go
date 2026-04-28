package langfuse

import (
	"context"
	"fmt"
	"log"

	"github.com/ezardev-team/langfuse-go/internal/pkg/api"
	"github.com/ezardev-team/langfuse-go/model"
)

// UpsertPromptRequest 는 Langfuse `POST /api/public/v2/prompts` body.
// Langfuse 가 같은 name 으로 받으면 새 version 을 자동 생성한다 (idempotent semantics).
//
// Type 분기:
//   - "chat": Prompt 는 []model.PromptMessage (또는 동일 구조의 []map[string]any).
//   - "text": Prompt 는 string.
type UpsertPromptRequest struct {
	Name          string
	Type          string // "chat" | "text"
	Prompt        any
	Config        any
	Labels        []string
	Tags          []string
	CommitMessage string
}

// UpsertPrompt 는 Langfuse 에 prompt 를 등록 또는 업데이트한다.
// 같은 name 이 이미 있으면 Langfuse 가 새 version 을 생성한다.
// 응답으로 새로 생성된 prompt (id + version) 를 반환.
func (l *Langfuse) UpsertPrompt(ctx context.Context, req UpsertPromptRequest) (*model.Prompt, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("UpsertPrompt: Name 필수")
	}
	switch req.Type {
	case "chat", "text":
		// ok
	default:
		return nil, fmt.Errorf("UpsertPrompt: Type 가 잘못됨: %q (chat|text)", req.Type)
	}
	if req.Prompt == nil {
		return nil, fmt.Errorf("UpsertPrompt: Prompt body 필수")
	}

	apiReq := &api.PromptUpsertRequest{
		Name:          req.Name,
		Type:          req.Type,
		Prompt:        req.Prompt,
		Config:        req.Config,
		Labels:        req.Labels,
		Tags:          req.Tags,
		CommitMessage: req.CommitMessage,
	}

	path, err := apiReq.Path()
	if err != nil {
		return nil, fmt.Errorf("UpsertPrompt %q: %w", req.Name, err)
	}

	res := api.PromptUpsertResponse{}
	if err := l.client.UpsertPrompt(ctx, apiReq, &res); err != nil {
		log.Printf("UpsertPrompt request failed: %v (path=%s name=%s)", err, path, req.Name)
		return nil, fmt.Errorf("UpsertPrompt %q: %w", req.Name, err)
	}

	if !res.IsSuccess() {
		if res.RawBody != nil {
			log.Printf("UpsertPrompt failed with status code: %d (path=%s name=%s) body=%s", res.Code, path, req.Name, *res.RawBody)
			return nil, fmt.Errorf("UpsertPrompt %q failed: status=%d body=%s", req.Name, res.Code, *res.RawBody)
		}
		log.Printf("UpsertPrompt failed with status code: %d (path=%s name=%s)", res.Code, path, req.Name)
		return nil, fmt.Errorf("UpsertPrompt %q failed: status=%d", req.Name, res.Code)
	}

	return &res.Prompt, nil
}
