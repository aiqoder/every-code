package llmchat

import "testing"

func TestReasoningTextFromDeltaJSON(t *testing.T) {
	if got := reasoningTextFromDeltaJSON(`{"reasoning_content":"先看需求"}`); got != "先看需求" {
		t.Fatalf("deepseek: %q", got)
	}
	if got := reasoningTextFromDeltaJSON(`{"reasoning":"ok"}`); got != "ok" {
		t.Fatalf("reasoning: %q", got)
	}
	if got := reasoningTextFromDeltaJSON(`{"reasoning":{"text":"nested"}}`); got != "nested" {
		t.Fatalf("nested: %q", got)
	}
	if got := reasoningTextFromDeltaJSON(`{"content":"hi"}`); got != "" {
		t.Fatalf("plain content should not count as reasoning: %q", got)
	}
}

func TestThinkingBudgetTokens(t *testing.T) {
	if got := thinkingBudgetTokens("low", 8192); got != 2048 {
		t.Fatalf("low: %d", got)
	}
	if got := thinkingBudgetTokens("medium", 4096); got != 2048 {
		t.Fatalf("capped to half: %d", got)
	}
	if got := thinkingBudgetTokens("high", 1000); got != 0 {
		t.Fatalf("too small window: %d", got)
	}
}
