package llmchat

import (
	"encoding/json"
	"strings"
)

func normalizeReasoningEffort(effort string) string {
	s := strings.ToLower(strings.TrimSpace(effort))
	switch s {
	case "", "medium":
		return "medium"
	case "none", "minimal", "low", "high", "xhigh", "max":
		return s
	default:
		return s
	}
}

func thinkingBudgetTokens(effort string, maxTokens int64) int64 {
	var budget int64
	switch normalizeReasoningEffort(effort) {
	case "none", "minimal":
		budget = 1024
	case "low":
		budget = 2048
	case "high", "xhigh", "max":
		budget = 16384
	default:
		budget = 8192
	}
	if maxTokens > 0 {
		capAt := maxTokens / 2
		if capAt < 1024 {
			return 0
		}
		if budget >= maxTokens {
			budget = capAt
		}
	}
	if budget < 1024 {
		return 0
	}
	return budget
}

func reasoningTextFromDeltaJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return ""
	}
	for _, key := range []string{"reasoning_content", "reasoning", "thinking", "reasoning_text"} {
		v, ok := obj[key]
		if !ok {
			continue
		}
		if s := stringFromJSON(v); s != "" {
			return s
		}
	}
	return ""
}

func stringFromJSON(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj map[string]any
	if json.Unmarshal(raw, &obj) != nil {
		return ""
	}
	for _, key := range []string{"text", "content"} {
		if t, ok := obj[key].(string); ok && t != "" {
			return t
		}
	}
	return ""
}
