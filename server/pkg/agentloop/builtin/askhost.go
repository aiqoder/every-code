package builtin

import (
	"context"

	"chengzi-agent-code-server/pkg/agentloop/askgate"
)

// AskHost 宿主会话向用户提问的能力：登记题目并阻塞等待作答。
// 由 chat_logic 注入，AskQuestion 工具通过 Runtime 回调使用。
type AskHost interface {
	AskOpen(ctx context.Context, toolCallID string, spec askgate.Spec) (questionID string, err error)
	AskWait(ctx context.Context, questionID string) (answerJSON string, err error)
}
