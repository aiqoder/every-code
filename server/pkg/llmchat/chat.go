package llmchat

import (
	"context"
	"fmt"
	"strings"
)

// EventHandler 流式事件回调；返回 error 可中断流。
type EventHandler func(StreamEvent) error

// Chat 协议无关的流式对话客户端。
type Chat interface {
	Protocol() Protocol
	ChatStream(ctx context.Context, req *ChatRequest, onEvent EventHandler) (*AssistantTurn, error)
}

// New 按协议构造 Chat 客户端。
func New(cfg Config) (Chat, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("llmchat: baseURL is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("llmchat: apiKey is required")
	}
	switch cfg.Protocol {
	case ProtocolOpenAICompatible:
		return newOpenAIChat(cfg.APIKey, base), nil
	case ProtocolAnthropic:
		return newAnthropicChat(cfg.APIKey, base), nil
	default:
		return nil, fmt.Errorf("llmchat: unsupported protocol %q", cfg.Protocol)
	}
}

// ResolveProtocol 根据供应商 BaseURL 选择协议（优先 OpenAI Compatible）。
func ResolveProtocol(openaiBaseURL, anthropicBaseURL string) (Protocol, string, error) {
	if u := strings.TrimSpace(openaiBaseURL); u != "" {
		return ProtocolOpenAICompatible, u, nil
	}
	if u := strings.TrimSpace(anthropicBaseURL); u != "" {
		return ProtocolAnthropic, u, nil
	}
	return "", "", fmt.Errorf("供应商未配置 OpenAI Compatible 或 Anthropic BaseUrl")
}
