package llmprovider

import (
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/types"
)

func CanAccess(p *model.LlmProvider, userID uint64) bool {
	if p == nil {
		return false
	}
	if p.UserID == nil {
		return true
	}
	return *p.UserID == userID
}

func CanEdit(p *model.LlmProvider, userID uint64) bool {
	return p != nil && p.UserID != nil && *p.UserID == userID
}

func CanAccessModel(m *model.LlmModel, userID uint64) bool {
	if m == nil {
		return false
	}
	return CanAccess(m.Provider, userID)
}

func CanEditModel(m *model.LlmModel, userID uint64) bool {
	if m == nil {
		return false
	}
	return CanEdit(m.Provider, userID)
}

func buildProvider(req *types.LlmProviderReq, userID *uint64) (*model.LlmProvider, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("名称不能为空")
	}
	return &model.LlmProvider{
		UserID:                  userID,
		Name:                    name,
		Website:                 strings.TrimSpace(req.Website),
		APIKey:                  strings.TrimSpace(req.APIKey),
		OpenAICompatibleBaseURL: strings.TrimSpace(req.OpenAICompatibleBaseURL),
		AnthropicBaseURL:        strings.TrimSpace(req.AnthropicBaseURL),
	}, nil
}

func applyProvider(p *model.LlmProvider, req *types.LlmProviderReq) error {
	built, err := buildProvider(req, p.UserID)
	if err != nil {
		return err
	}
	p.Name = built.Name
	p.Website = built.Website
	p.APIKey = built.APIKey
	p.OpenAICompatibleBaseURL = built.OpenAICompatibleBaseURL
	p.AnthropicBaseURL = built.AnthropicBaseURL
	return nil
}
