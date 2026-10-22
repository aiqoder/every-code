package llmchat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

type anthropicChat struct {
	client anthropic.Client
}

func newAnthropicChat(apiKey, baseURL string) *anthropicChat {
	return &anthropicChat{
		client: anthropic.NewClient(
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseURL),
		),
	}
}

func (c *anthropicChat) Protocol() Protocol { return ProtocolAnthropic }

func (c *anthropicChat) ChatStream(ctx context.Context, req *ChatRequest, onEvent EventHandler) (*AssistantTurn, error) {
	if req == nil {
		return nil, fmt.Errorf("llmchat: request is nil")
	}
	if req.Model == "" {
		return nil, fmt.Errorf("llmchat: model is required")
	}
	if onEvent == nil {
		onEvent = func(StreamEvent) error { return nil }
	}

	system, messages := toAnthropicMessages(req.Messages)
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		// Anthropic 必填 max_tokens；8192 不够写完整单页游戏/长文件。
		maxTokens = 32768
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: maxTokens,
		Messages:  messages,
	}
	if system != "" {
		params.System = []anthropic.TextBlockParam{{Text: system}}
	}
	if req.Temperature != nil {
		params.Temperature = param.NewOpt(*req.Temperature)
	}
	if req.Reasoning {
		if budget := thinkingBudgetTokens(req.ReasoningEffort, maxTokens); budget > 0 {
			params.Thinking = anthropic.ThinkingConfigParamOfEnabled(budget)
			// 打开 thinking 时 Anthropic 要求 temperature=1
			params.Temperature = param.NewOpt(1.0)
		}
	}
	if len(req.Tools) > 0 {
		params.Tools = toAnthropicTools(req.Tools)
	}

	stream := c.client.Messages.NewStreaming(ctx, params)
	var acc anthropic.Message

	for stream.Next() {
		event := stream.Current()
		if err := acc.Accumulate(event); err != nil {
			_ = onEvent(StreamEvent{Type: StreamEventError, Error: err.Error()})
			return nil, err
		}

		switch ev := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch delta := ev.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				if delta.Text != "" {
					if err := onEvent(StreamEvent{Type: StreamEventTextDelta, Text: delta.Text}); err != nil {
						return nil, err
					}
				}
			case anthropic.ThinkingDelta:
				if delta.Thinking != "" {
					if err := onEvent(StreamEvent{Type: StreamEventReasoningDelta, Text: delta.Thinking}); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		_ = onEvent(StreamEvent{Type: StreamEventError, Error: err.Error()})
		return nil, err
	}

	turn := &AssistantTurn{
		FinishReason: string(acc.StopReason),
		Usage: NormalizeAnthropicUsage(
			acc.Usage.InputTokens,
			acc.Usage.OutputTokens,
			acc.Usage.CacheCreationInputTokens,
			acc.Usage.CacheReadInputTokens,
		),
	}
	for _, block := range acc.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			turn.Content += b.Text
		case anthropic.ToolUseBlock:
			args, _ := json.Marshal(b.Input)
			call := ToolCall{
				ID:        b.ID,
				Name:      b.Name,
				Arguments: string(args),
			}
			turn.ToolCalls = append(turn.ToolCalls, call)
			_ = onEvent(StreamEvent{Type: StreamEventToolCallDelta, ToolCall: &call})
		}
	}

	if err := onEvent(StreamEvent{Type: StreamEventMessageEnd, Turn: turn}); err != nil {
		return turn, err
	}
	return turn, nil
}

func toAnthropicMessages(msgs []Message) (system string, out []anthropic.MessageParam) {
	var pendingToolResults []anthropic.ContentBlockParamUnion

	flushToolResults := func() {
		if len(pendingToolResults) == 0 {
			return
		}
		out = append(out, anthropic.NewUserMessage(pendingToolResults...))
		pendingToolResults = nil
	}

	for _, m := range msgs {
		switch m.Role {
		case RoleSystem:
			if system == "" {
				system = m.Content
			} else {
				system += "\n" + m.Content
			}
		case RoleUser:
			flushToolResults()
			out = append(out, anthropicUserMessage(m))
		case RoleAssistant:
			flushToolResults()
			blocks := make([]anthropic.ContentBlockParamUnion, 0, 1+len(m.ToolCalls))
			if m.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(m.Content))
			}
			for _, tc := range m.ToolCalls {
				var input any
				if err := json.Unmarshal([]byte(tc.Arguments), &input); err != nil {
					input = map[string]any{}
				}
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
			}
			if len(blocks) == 0 {
				blocks = append(blocks, anthropic.NewTextBlock(""))
			}
			out = append(out, anthropic.NewAssistantMessage(blocks...))
		case RoleTool:
			pendingToolResults = append(pendingToolResults,
				anthropic.NewToolResultBlock(m.ToolCallID, m.Content, false))
		}
	}
	flushToolResults()
	return system, out
}

func anthropicUserMessage(m Message) anthropic.MessageParam {
	if len(m.Parts) == 0 {
		return anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content))
	}
	text := m.Content
	if strings.TrimSpace(text) == "" {
		text = "请查看附件。"
	}
	blocks := make([]anthropic.ContentBlockParamUnion, 0, 1+len(m.Parts))
	blocks = append(blocks, anthropic.NewTextBlock(text))
	for _, p := range m.Parts {
		switch p.Kind {
		case PartImage:
			blocks = append(blocks, anthropic.NewImageBlockBase64(p.MimeType, p.Data))
		case PartPDF:
			blocks = append(blocks, anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{Data: p.Data}))
		}
	}
	return anthropic.NewUserMessage(blocks...)
}

func toAnthropicTools(tools []ToolDef) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		schema := anthropic.ToolInputSchemaParam{
			Type: constant.Object("object"),
		}
		params := t.Parameters
		if params == nil {
			params = DefaultToolParameters()
		}
		if props, ok := params["properties"]; ok {
			schema.Properties = props
		} else {
			schema.Properties = map[string]any{}
		}
		if req, ok := params["required"]; ok {
			schema.Required = toStringSlice(req)
		}
		tool := anthropic.ToolParam{
			Name:        t.Name,
			Description: param.NewOpt(t.Description),
			InputSchema: schema,
		}
		out = append(out, anthropic.ToolUnionParam{OfTool: &tool})
	}
	return out
}

func toStringSlice(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
