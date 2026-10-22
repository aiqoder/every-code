package admin

import (
	"strings"

	"chengzi-agent-code-server/internal/types"
)

// LlmProviderInput 供应商创建/更新入参。
type LlmProviderInput struct {
	Name                    string
	Website                 string
	APIKey                  string
	OpenAICompatibleBaseURL string
	AnthropicBaseURL        string
}

func toLlmProviderInput(req *types.LlmProviderReq) LlmProviderInput {
	in := LlmProviderInput{
		Name:                    req.Name,
		Website:                 req.Website,
		APIKey:                  req.APIKey,
		OpenAICompatibleBaseURL: req.OpenAICompatibleBaseURL,
		AnthropicBaseURL:        req.AnthropicBaseURL,
	}
	return normalizeProviderInput(in)
}

func ToLlmModelInput(req *types.LlmModelReq) LlmModelInput {
	return LlmModelInput{
		ProviderID:         req.ProviderID,
		Name:               req.Name,
		Attachment:         req.Attachment,
		Reasoning:          req.Reasoning,
		ReasoningOptions:   req.ReasoningOptions,
		Temperature:        req.Temperature,
		InputCapabilities:  req.InputCapabilities,
		InputContextLimit:  req.InputContextLimit,
		OutputContextLimit: req.OutputContextLimit,
		InputPrice:         req.InputPrice,
		OutputPrice:        req.OutputPrice,
		CachePrice:         req.CachePrice,
	}
}

func normalizeProviderInput(in LlmProviderInput) LlmProviderInput {
	in.Name = strings.TrimSpace(in.Name)
	in.Website = strings.TrimSpace(in.Website)
	in.APIKey = strings.TrimSpace(in.APIKey)
	in.OpenAICompatibleBaseURL = strings.TrimSpace(in.OpenAICompatibleBaseURL)
	in.AnthropicBaseURL = strings.TrimSpace(in.AnthropicBaseURL)
	return in
}
