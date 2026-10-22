package llmchat

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
)

type openAIChat struct {
	client openai.Client
}

func newOpenAIChat(apiKey, baseURL string) *openAIChat {
	return &openAIChat{
		client: openai.NewClient(
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseURL),
		),
	}
}

func (c *openAIChat) Protocol() Protocol { return ProtocolOpenAICompatible }

func (c *openAIChat) ChatStream(ctx context.Context, req *ChatRequest, onEvent EventHandler) (*AssistantTurn, error) {
	if req == nil {
		return nil, fmt.Errorf("llmchat: request is nil")
	}
	if req.Model == "" {
		return nil, fmt.Errorf("llmchat: model is required")
	}
	if onEvent == nil {
		onEvent = func(StreamEvent) error { return nil }
	}

	params := openai.ChatCompletionNewParams{
		Model:    req.Model,
		Messages: toOpenAIMessages(req.Messages),
	}
	if req.Temperature != nil {
		params.Temperature = openai.Float(*req.Temperature)
	}
	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(req.MaxTokens)
	}
	if req.Reasoning {
		effort := normalizeReasoningEffort(req.ReasoningEffort)
		params.ReasoningEffort = shared.ReasoningEffort(effort)
	}
	if len(req.Tools) > 0 {
		params.Tools = toOpenAITools(req.Tools)
	}
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, params)
	acc := openai.ChatCompletionAccumulator{}

	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if text := reasoningTextFromDeltaJSON(delta.RawJSON()); text != "" {
				if err := onEvent(StreamEvent{Type: StreamEventReasoningDelta, Text: text}); err != nil {
					return nil, err
				}
			}
			if delta.Content != "" {
				if err := onEvent(StreamEvent{Type: StreamEventTextDelta, Text: delta.Content}); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		_ = onEvent(StreamEvent{Type: StreamEventError, Error: err.Error()})
		return nil, err
	}

	turn := &AssistantTurn{
		Usage: NormalizeOpenAIUsage(
			acc.Usage.PromptTokens,
			acc.Usage.CompletionTokens,
			acc.Usage.PromptTokensDetails.CachedTokens,
			acc.Usage.PromptTokensDetails.CacheWriteTokens,
		),
	}
	if len(acc.Choices) > 0 {
		msg := acc.Choices[0].Message
		turn.Content = msg.Content
		turn.FinishReason = string(acc.Choices[0].FinishReason)
		for _, tc := range msg.ToolCalls {
			// 注意：不可用 tc.AsFunction()，SDK 累积器不保留 JSON.raw，该方法会返回全空结构体
			// 直接读取累积后的字段（流式增量已正确拼接）
			call := ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
			// 跳过全空调用：部分网关会发送空 tool_calls 增量，直接透传会导致 unknown tool 错误
			if call.ID == "" && call.Name == "" && call.Arguments == "" {
				continue
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

func toOpenAIMessages(msgs []Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case RoleSystem:
			out = append(out, openai.SystemMessage(m.Content))
		case RoleUser:
			out = append(out, openAIUserMessage(m))
		case RoleAssistant:
			if len(m.ToolCalls) == 0 {
				out = append(out, openai.AssistantMessage(m.Content))
				continue
			}
			asst := openai.ChatCompletionAssistantMessageParam{}
			if m.Content != "" {
				asst.Content.OfString = param.NewOpt(m.Content)
			}
			for _, tc := range m.ToolCalls {
				asst.ToolCalls = append(asst.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
					OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
						ID: tc.ID,
						Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
							Name:      tc.Name,
							Arguments: tc.Arguments,
						},
					},
				})
			}
			out = append(out, openai.ChatCompletionMessageParamUnion{OfAssistant: &asst})
		case RoleTool:
			out = append(out, openai.ToolMessage(m.Content, m.ToolCallID))
		}
	}
	return out
}

func openAIUserMessage(m Message) openai.ChatCompletionMessageParamUnion {
	if len(m.Parts) == 0 {
		return openai.UserMessage(m.Content)
	}
	parts := make([]openai.ChatCompletionContentPartUnionParam, 0, 1+len(m.Parts))
	text := m.Content
	if strings.TrimSpace(text) == "" {
		text = "请查看附件。"
	}
	parts = append(parts, openai.TextContentPart(text))
	for _, p := range m.Parts {
		switch p.Kind {
		case PartImage:
			parts = append(parts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
				URL: "data:" + p.MimeType + ";base64," + p.Data,
			}))
		case PartPDF:
			name := p.FileName
			if name == "" {
				name = "file.pdf"
			}
			parts = append(parts, openai.FileContentPart(openai.ChatCompletionContentPartFileFileParam{
				FileData: param.NewOpt("data:application/pdf;base64," + p.Data),
				Filename: param.NewOpt(name),
			}))
		}
	}
	return openai.UserMessage(parts)
}

func toOpenAITools(tools []ToolDef) []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		params := t.Parameters
		if params == nil {
			params = DefaultToolParameters()
		}
		out = append(out, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  shared.FunctionParameters(params),
		}))
	}
	return out
}
