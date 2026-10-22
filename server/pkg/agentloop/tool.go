package agentloop

import (
	"context"

	"chengzi-agent-code-server/pkg/llmchat"
)

// Tool 可注入 agent loop 的工具。
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, argsJSON string, rt *Runtime) (string, error)
}

// SharedWorkspace 是 Runtime.Shared 里当前工作目录的键。
// 智能体用 shell 建好 worktree 后写入，后续下发到客户端的工具会改到这个目录。
const SharedWorkspace = "workspace"

// Runtime 工具执行时的运行时上下文。
type Runtime struct {
	Emit               func(Event)
	Depth              int
	MaxDepth           int
	Chat               llmchat.Chat
	Model              string
	Tools              []Tool
	Shared             map[string]any
	InputContextLimit  int64
	OutputContextLimit int64
	Reasoning          bool
	ReasoningEffort    string
	// ToolCallID 当前这次工具调用的 id。子代理转发事件时写入 ParentToolCallID。
	ToolCallID string
	// Ask 登记一次向用户的提问并返回 questionId。toolCallID 由工具侧传入，
	// 上层把它与 questionId 一起登记。nil 表示当前运行不能提问。
	Ask func(ctx context.Context, toolCallID, specJSON string) (questionID string, err error)
	// AskWait 阻塞等待用户提交该 questionId 的答案，返回格式化后的答案 JSON。
	// 与 Ask 成对注入；超时/取消由调用方（AskQuestion）根据 ctx 处理。
	AskWait func(ctx context.Context, questionID string) (answerJSON string, err error)
}

// ToolDefs 将 Tool 列表转为 llmchat 工具定义。
func ToolDefs(tools []Tool) []llmchat.ToolDef {
	out := make([]llmchat.ToolDef, 0, len(tools))
	for _, t := range tools {
		params := t.Parameters()
		if params == nil {
			params = llmchat.DefaultToolParameters()
		}
		out = append(out, llmchat.ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  params,
		})
	}
	return out
}

// FindTool 按名称查找工具。
func FindTool(tools []Tool, name string) Tool {
	for _, t := range tools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}
