package agentloop

import "encoding/json"

// EventType 流程事件类型（对外 SSE 同构）。
type EventType string

const (
	EventRunStarted       EventType = "run_started"
	EventRunFinished      EventType = "run_finished"
	EventError            EventType = "error"
	EventTextDelta        EventType = "text_delta"
	EventReasoningDelta   EventType = "reasoning_delta"
	EventToolCallStart    EventType = "tool_call_start"
	EventToolCallEnd      EventType = "tool_call_end"
	EventPlanUpdated      EventType = "plan_updated"
	EventSubagentStarted  EventType = "subagent_started"
	EventSubagentFinished EventType = "subagent_finished"
	// EventAskQuestion 向用户发起选择题；Data 含 questionId、title、questions。
	EventAskQuestion      EventType = "ask_question"
	EventContextUsage     EventType = "context_usage"
	EventContextCompacted EventType = "context_compacted"
	// EventLLMRetrying LLM 调用瞬时失败、即将退避重试；Data 含 attempt / max_attempts / delay_ms。
	EventLLMRetrying EventType = "llm_retrying"
	// EventRunWarning 非致命运行警告（输出截断、内容过滤、步数用尽等）；Text 为面向用户的描述。
	EventRunWarning EventType = "run_warning"
)

// Event 流程事件。
type Event struct {
	Type             EventType      `json:"type"`
	Text             string         `json:"text,omitempty"`
	ToolCallID       string         `json:"toolCallId,omitempty"`
	ToolName         string         `json:"toolName,omitempty"`
	Arguments        string         `json:"arguments,omitempty"`
	Result           string         `json:"result,omitempty"`
	Plan             any            `json:"plan,omitempty"`
	Goal             string         `json:"goal,omitempty"`
	ParentToolCallID string         `json:"parentToolCallId,omitempty"`
	Error            string         `json:"error,omitempty"`
	Data             map[string]any `json:"data,omitempty"`
}

// EmitJSON 将事件编码为 JSON（失败时返回 "{}"）。
func (e Event) EmitJSON() string {
	b, err := json.Marshal(e)
	if err != nil {
		return "{}"
	}
	return string(b)
}
