package conversation

import (
	"context"
	"testing"
	"time"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/llmchat"
)

type stubChat struct {
	turn *llmchat.AssistantTurn
	err  error
}

func (s stubChat) Protocol() llmchat.Protocol { return llmchat.ProtocolOpenAICompatible }

func (s stubChat) ChatStream(context.Context, *llmchat.ChatRequest, llmchat.EventHandler) (*llmchat.AssistantTurn, error) {
	return s.turn, s.err
}

type memoryUsageRepo struct {
	rows []*model.TokenUsage
}

func (m *memoryUsageRepo) Create(_ context.Context, row *model.TokenUsage) error {
	cp := *row
	m.rows = append(m.rows, &cp)
	return nil
}

func (m *memoryUsageRepo) Summarize(context.Context, uint64, time.Time, time.Time) (repository.TokenUsageTotals, error) {
	return repository.TokenUsageTotals{}, nil
}

func (m *memoryUsageRepo) Trend(context.Context, uint64, time.Time, time.Time, string) ([]repository.TokenUsageBucket, error) {
	return nil, nil
}

func TestRecordingChatPersistsUsage(t *testing.T) {
	repo := &memoryUsageRepo{}
	inner := stubChat{turn: &llmchat.AssistantTurn{
		Content: "ok",
		Usage:   llmchat.TokenUsage{PromptTokens: 10, CompletionTokens: 4},
	}}
	chat := wrapTokenUsageRecorder(inner, repo, tokenUsageMeta{
		UserID: 7, ConversationID: 3, ModelName: "demo",
		InputPrice: 1, OutputPrice: 2,
	})
	turn, err := chat.ChatStream(context.Background(), &llmchat.ChatRequest{Model: "demo"}, nil)
	if err != nil || turn.Content != "ok" {
		t.Fatalf("stream: %v %#v", err, turn)
	}
	if len(repo.rows) != 1 {
		t.Fatalf("rows=%d", len(repo.rows))
	}
	got := repo.rows[0]
	if got.PromptTokens != 10 || got.CompletionTokens != 4 || got.UserID != 7 {
		t.Fatalf("row %#v", got)
	}
	if got.Cost <= 0 {
		t.Fatalf("cost=%v", got.Cost)
	}
}

func TestRecordingChatSkipsOnError(t *testing.T) {
	repo := &memoryUsageRepo{}
	inner := stubChat{err: context.Canceled}
	chat := wrapTokenUsageRecorder(inner, repo, tokenUsageMeta{})
	_, err := chat.ChatStream(context.Background(), &llmchat.ChatRequest{Model: "demo"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(repo.rows) != 0 {
		t.Fatalf("should not persist on error")
	}
}
