package conversation

import (
	"context"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/llmchat"
)

type tokenUsageMeta struct {
	UserID         uint64
	ConversationID uint64
	ModelID        uint64
	ModelName      string
	ProviderID     uint64
	InputPrice     float64
	OutputPrice    float64
	CachePrice     float64
}

type recordingChat struct {
	inner llmchat.Chat
	repo  repository.TokenUsageRepository
	meta  tokenUsageMeta
}

func wrapTokenUsageRecorder(inner llmchat.Chat, repo repository.TokenUsageRepository, meta tokenUsageMeta) llmchat.Chat {
	if inner == nil || repo == nil {
		return inner
	}
	return &recordingChat{inner: inner, repo: repo, meta: meta}
}

func (c *recordingChat) Protocol() llmchat.Protocol {
	return c.inner.Protocol()
}

func (c *recordingChat) ChatStream(ctx context.Context, req *llmchat.ChatRequest, onEvent llmchat.EventHandler) (*llmchat.AssistantTurn, error) {
	turn, err := c.inner.ChatStream(ctx, req, onEvent)
	if err == nil && turn != nil {
		c.persist(context.WithoutCancel(ctx), turn.Usage)
	}
	return turn, err
}

func (c *recordingChat) persist(ctx context.Context, usage llmchat.TokenUsage) {
	row := &model.TokenUsage{
		UserID:              c.meta.UserID,
		ConversationID:      c.meta.ConversationID,
		ModelID:             c.meta.ModelID,
		ModelName:           c.meta.ModelName,
		ProviderID:          c.meta.ProviderID,
		PromptTokens:        usage.PromptTokens,
		CompletionTokens:    usage.CompletionTokens,
		CacheCreationTokens: usage.CacheCreationTokens,
		CacheReadTokens:     usage.CacheReadTokens,
		Cost:                usage.Cost(c.meta.InputPrice, c.meta.OutputPrice, c.meta.CachePrice),
	}
	_ = c.repo.Create(ctx, row)
}
