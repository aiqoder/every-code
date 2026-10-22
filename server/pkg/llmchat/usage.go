package llmchat

// PricePerMillion 模型单价按「美元 / 百万 tokens」计。
const PricePerMillion = 1_000_000.0

// RealTokens 真实消耗：新增输入 + 输出 + 缓存写入 + 缓存命中。
func (u TokenUsage) RealTokens() int64 {
	return u.PromptTokens + u.CompletionTokens + u.CacheCreationTokens + u.CacheReadTokens
}

// CacheHitRate 缓存命中率 = 命中 / (新增输入 + 命中 + 创建)。
func (u TokenUsage) CacheHitRate() float64 {
	denom := u.PromptTokens + u.CacheReadTokens + u.CacheCreationTokens
	if denom <= 0 {
		return 0
	}
	return float64(u.CacheReadTokens) / float64(denom)
}

// Cost 按百万 token 单价估算美元成本。缓存写入按输入价计。
func (u TokenUsage) Cost(inputPrice, outputPrice, cachePrice float64) float64 {
	return (float64(u.PromptTokens)*inputPrice +
		float64(u.CompletionTokens)*outputPrice +
		float64(u.CacheReadTokens)*cachePrice +
		float64(u.CacheCreationTokens)*inputPrice) / PricePerMillion
}

// Add 累加另一笔用量。
func (u TokenUsage) Add(other TokenUsage) TokenUsage {
	return TokenUsage{
		PromptTokens:        u.PromptTokens + other.PromptTokens,
		CompletionTokens:    u.CompletionTokens + other.CompletionTokens,
		CacheCreationTokens: u.CacheCreationTokens + other.CacheCreationTokens,
		CacheReadTokens:     u.CacheReadTokens + other.CacheReadTokens,
	}
}

// NormalizeOpenAIUsage 将 OpenAI 兼容协议的 usage 归一化。
// prompt 通常已含 cached；若 prompt >= cached，新增输入 = prompt - cached。
func NormalizeOpenAIUsage(prompt, completion, cached, cacheWrite int64) TokenUsage {
	newInput := prompt
	if cached > 0 && prompt >= cached {
		newInput = prompt - cached
	}
	if newInput < 0 {
		newInput = 0
	}
	return TokenUsage{
		PromptTokens:        newInput,
		CompletionTokens:    completion,
		CacheCreationTokens: cacheWrite,
		CacheReadTokens:     cached,
	}
}

// NormalizeAnthropicUsage Anthropic 的 input 一般不含缓存读写，直接分列。
func NormalizeAnthropicUsage(input, output, cacheCreate, cacheRead int64) TokenUsage {
	return TokenUsage{
		PromptTokens:        input,
		CompletionTokens:    output,
		CacheCreationTokens: cacheCreate,
		CacheReadTokens:     cacheRead,
	}
}
