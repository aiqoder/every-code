package conversation

import (
	"context"

	"chengzi-agent-code-server/pkg/agentloop/askgate"
	"chengzi-agent-code-server/pkg/agentloop/builtin"
)

// chatHost 宿主会话环境：把会话级服务（提问登记/等待）注入 AskQuestion 工具，
// 供其登记并等待用户作答。新增会话级能力时在此挂载，不需要修改 agentloop 引擎。
type chatHost struct {
	gate *askgate.Gate
	conv uint64
	user uint64
}

// newChatHost 创建宿主环境；提问通道未就绪时返回 nil（工具侧据此报错）。
func newChatHost(gate *askgate.Gate, conversationID, userID uint64) builtin.AskHost {
	if gate == nil {
		return nil
	}
	return &chatHost{gate: gate, conv: conversationID, user: userID}
}

// 编译期断言：chatHost 实现 AskQuestion 所需的宿主能力接口。
var _ builtin.AskHost = (*chatHost)(nil)

// AskOpen 登记一次提问并返回 questionId。
func (h *chatHost) AskOpen(_ context.Context, toolCallID string, spec askgate.Spec) (string, error) {
	return h.gate.Open(h.conv, h.user, toolCallID, spec)
}

// AskWait 阻塞等待用户提交答案，返回格式化后的答案 JSON。
func (h *chatHost) AskWait(ctx context.Context, questionID string) (string, error) {
	return h.gate.Wait(ctx, questionID)
}
