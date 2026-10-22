package agentloop

import (
	"context"
	"strings"

	"chengzi-agent-code-server/pkg/llmchat"
)

const compactSystemPrompt = `你是对话压缩器。将用户提供的历史对话浓缩成结构化摘要，供后续模型继续完成任务。

必须保留：
- 用户目标和约束
- 已做出的决策与结论
- 改动过的文件、路径、命令与关键输出
- 未完成事项与当前进度
- 错误原因与已尝试的修复

不要编造，不要寒暄，不要重复原始工具输出。使用简洁中文。`

type llmSummarizer struct {
	chat  llmchat.Chat
	model string
}

func newLLMSummarizer(chat llmchat.Chat, model string) summarizer {
	if chat == nil || strings.TrimSpace(model) == "" {
		return nil
	}
	return &llmSummarizer{chat: chat, model: model}
}

func (s *llmSummarizer) Summarize(ctx context.Context, transcript string) (string, error) {
	transcript = strings.TrimSpace(transcript)
	if transcript == "" {
		return "", nil
	}
	turn, err := s.chat.ChatStream(ctx, &llmchat.ChatRequest{
		Model:     s.model,
		MaxTokens: summarizerMaxTokens,
		Messages: []llmchat.Message{
			{Role: llmchat.RoleSystem, Content: compactSystemPrompt},
			{Role: llmchat.RoleUser, Content: "请压缩以下历史对话：\n\n" + transcript},
		},
	}, func(llmchat.StreamEvent) error { return nil })
	if err != nil {
		return "", err
	}
	if turn == nil {
		return "", nil
	}
	return strings.TrimSpace(turn.Content), nil
}

func isCompactRequest(req *llmchat.ChatRequest) bool {
	if req == nil || len(req.Tools) > 0 || len(req.Messages) == 0 {
		return false
	}
	return strings.Contains(req.Messages[0].Content, "你是对话压缩器")
}
