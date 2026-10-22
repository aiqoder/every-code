package agentloop

import (
	"chengzi-agent-code-server/pkg/llmchat"
	"chengzi-agent-code-server/pkg/tokencount"
)

// EstimateTokens 使用 cl100k_base 精确计数，与前端 desktop/frontend/src/lib/tokens.ts 对齐。
func EstimateTokens(text string) int {
	return tokencount.Count(text)
}

// EstimateMessage 估算单条消息（含角色开销与 tool_calls）。
func EstimateMessage(m llmchat.Message) int {
	return tokencount.CountMessage(m)
}

// EstimateMessages 估算消息列表（含回复预填充）。
func EstimateMessages(msgs []llmchat.Message) int {
	return tokencount.CountMessages(msgs)
}

// EstimateTools 估算工具定义占用。
func EstimateTools(tools []llmchat.ToolDef) int {
	return tokencount.CountTools(tools)
}

// EstimatePromptTokens 估算即将送给模型的 prompt（消息 + 工具定义）。
func EstimatePromptTokens(msgs []llmchat.Message, toolsTok int) int {
	return tokencount.CountPrompt(msgs, toolsTok)
}

func takePrefixByTokens(s string, max int) string {
	return tokencount.Prefix(s, max)
}

func takeSuffixByTokens(s string, max int) string {
	return tokencount.Suffix(s, max)
}
