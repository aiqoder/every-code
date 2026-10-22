package llmchat

import "testing"

func TestNormalizeOpenAIUsage_SubtractsCached(t *testing.T) {
	u := NormalizeOpenAIUsage(100, 20, 30, 5)
	if u.PromptTokens != 70 {
		t.Fatalf("prompt=%d", u.PromptTokens)
	}
	if u.CacheReadTokens != 30 || u.CacheCreationTokens != 5 || u.CompletionTokens != 20 {
		t.Fatalf("got %#v", u)
	}
	if u.RealTokens() != 70+20+5+30 {
		t.Fatalf("real=%d", u.RealTokens())
	}
}

func TestNormalizeOpenAIUsage_NoCache(t *testing.T) {
	u := NormalizeOpenAIUsage(40, 8, 0, 0)
	if u.PromptTokens != 40 || u.CompletionTokens != 8 {
		t.Fatalf("got %#v", u)
	}
}

func TestNormalizeAnthropicUsage(t *testing.T) {
	u := NormalizeAnthropicUsage(12, 99, 4, 5)
	if u.PromptTokens != 12 || u.CompletionTokens != 99 || u.CacheCreationTokens != 4 || u.CacheReadTokens != 5 {
		t.Fatalf("got %#v", u)
	}
}

func TestCacheHitRate(t *testing.T) {
	u := TokenUsage{PromptTokens: 5310, CacheReadTokens: 19845}
	got := u.CacheHitRate()
	if got < 0.788 || got > 0.790 {
		t.Fatalf("rate=%v", got)
	}
	if (TokenUsage{}).CacheHitRate() != 0 {
		t.Fatal("empty should be 0")
	}
}

func TestCost(t *testing.T) {
	u := TokenUsage{PromptTokens: 1_000_000, CompletionTokens: 500_000, CacheReadTokens: 2_000_000}
	cost := u.Cost(1, 2, 0.1)
	// 1*1 + 0.5*2 + 2*0.1 = 2.2
	if cost < 2.199 || cost > 2.201 {
		t.Fatalf("cost=%v", cost)
	}
}

func TestAdd(t *testing.T) {
	sum := TokenUsage{PromptTokens: 1}.Add(TokenUsage{CompletionTokens: 2, CacheReadTokens: 3})
	if sum.PromptTokens != 1 || sum.CompletionTokens != 2 || sum.CacheReadTokens != 3 {
		t.Fatalf("got %#v", sum)
	}
}
