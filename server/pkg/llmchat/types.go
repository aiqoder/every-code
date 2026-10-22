package llmchat

import "encoding/json"

// Protocol 聊天协议。
type Protocol string

const (
	ProtocolOpenAICompatible Protocol = "openai_compatible"
	ProtocolAnthropic        Protocol = "anthropic"
)

// Role 消息角色。
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall 模型发起的工具调用。
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串
}

// PartKind 用户消息里的非文本附件。
type PartKind string

const (
	PartImage PartKind = "image"
	PartPDF   PartKind = "pdf"
)

// Part 图片或 PDF。Data 为不含 data: 前缀的 base64。
type Part struct {
	Kind     PartKind
	FileName string
	MimeType string
	Data     string
}

// Message 协议无关消息。
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"toolCalls,omitempty"`
	ToolCallID string     `json:"toolCallId,omitempty"`
	Name       string     `json:"name,omitempty"`
	// Parts 只出现在用户消息上，由各协议转成图片或文件块。
	Parts []Part `json:"-"`
}

// ToolDef 工具定义（JSON Schema parameters）。
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // 完整 JSON Schema object
}

// ChatRequest 一轮流式对话请求。
type ChatRequest struct {
	Model       string
	Messages    []Message
	Tools       []ToolDef
	Temperature *float64
	MaxTokens   int64
	// Reasoning 为 true 时向支持的协议打开思考/推理。
	Reasoning bool
	// ReasoningEffort 如 low / medium / high；空则在 Reasoning 时用 medium。
	ReasoningEffort string
}

// StreamEventType 流式事件类型。
type StreamEventType string

const (
	StreamEventTextDelta      StreamEventType = "text_delta"
	StreamEventReasoningDelta StreamEventType = "reasoning_delta"
	StreamEventToolCallDelta  StreamEventType = "tool_call_delta"
	StreamEventMessageEnd     StreamEventType = "message_end"
	StreamEventError          StreamEventType = "error"
)

// StreamEvent 统一流式事件。
type StreamEvent struct {
	Type     StreamEventType `json:"type"`
	Text     string          `json:"text,omitempty"`
	ToolCall *ToolCall       `json:"toolCall,omitempty"`
	Turn     *AssistantTurn  `json:"turn,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// TokenUsage 一次模型调用的用量（缓存命中已从新增输入中剥离）。
type TokenUsage struct {
	PromptTokens        int64 `json:"promptTokens"`
	CompletionTokens    int64 `json:"completionTokens"`
	CacheCreationTokens int64 `json:"cacheCreationTokens"`
	CacheReadTokens     int64 `json:"cacheReadTokens"`
}

// AssistantTurn 一轮模型输出结果。
type AssistantTurn struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"toolCalls,omitempty"`
	FinishReason string     `json:"finishReason,omitempty"`
	Usage        TokenUsage `json:"usage,omitempty"`
}

// Config 客户端配置。
type Config struct {
	Protocol Protocol
	APIKey   string
	BaseURL  string
}

// DefaultToolParameters 返回空 object schema。
func DefaultToolParameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// ObjectParams 构造 object JSON Schema，properties 为字段定义。
func ObjectParams(properties map[string]any, required ...string) map[string]any {
	if properties == nil {
		return DefaultToolParameters()
	}
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// MustJSON 将 v 编码为 JSON 字符串；失败时返回 "{}"。
func MustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
