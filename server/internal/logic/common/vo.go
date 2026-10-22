package common

import (
	"encoding/json"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/pkg/strutil"
	"chengzi-agent-code-server/pkg/timex"
)

// LlmModelVO 对外返回的大模型视图。
type LlmModelVO struct {
	ID                 uint64          `json:"id"`
	ProviderID         uint64          `json:"providerId"`
	ProviderName       string          `json:"providerName"`
	Name               string          `json:"name"`
	Attachment         bool            `json:"attachment"`
	Reasoning          bool            `json:"reasoning"`
	ReasoningOptions   []string        `json:"reasoningOptions"`
	Temperature        bool            `json:"temperature"`
	InputCapabilities  []string        `json:"inputCapabilities"`
	InputContextLimit  int64           `json:"inputContextLimit"`
	OutputContextLimit int64           `json:"outputContextLimit"`
	InputPrice         float64         `json:"inputPrice"`
	OutputPrice        float64         `json:"outputPrice"`
	CachePrice         float64         `json:"cachePrice"`
	CreatedAt          timex.LocalTime `json:"createdAt"`
	UpdatedAt          timex.LocalTime `json:"updatedAt"`
	UserID             *uint64         `json:"userId,omitempty"`
	Editable           bool            `json:"editable"`
}

func ToLlmModelVO(m *model.LlmModel) *LlmModelVO {
	return ToLlmModelVOFor(m, 0, false)
}

func ToLlmModelVOFor(m *model.LlmModel, viewerUserID uint64, isAdmin bool) *LlmModelVO {
	providerName := ""
	var userID *uint64
	if m.Provider != nil {
		providerName = m.Provider.Name
		userID = m.Provider.UserID
	}
	return &LlmModelVO{
		ID:                 m.ID,
		ProviderID:         m.ProviderID,
		ProviderName:       providerName,
		Name:               m.Name,
		Attachment:         m.Attachment,
		Reasoning:          m.Reasoning,
		ReasoningOptions:   strutil.SplitCSV(m.ReasoningOptions),
		Temperature:        m.Temperature,
		InputCapabilities:  strutil.SplitCSV(m.InputCapabilities),
		InputContextLimit:  m.InputContextLimit,
		OutputContextLimit: m.OutputContextLimit,
		InputPrice:         m.InputPrice,
		OutputPrice:        m.OutputPrice,
		CachePrice:         m.CachePrice,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
		UserID:             userID,
		Editable:           providerEditable(m.Provider, viewerUserID, isAdmin),
	}
}

// LlmProviderVO 对外返回的供应商视图。
type LlmProviderVO struct {
	ID                      uint64          `json:"id"`
	UserID                  *uint64         `json:"userId"`
	Name                    string          `json:"name"`
	Website                 string          `json:"website"`
	APIKey                  string          `json:"apiKey"`
	OpenAICompatibleBaseURL string          `json:"openaiCompatibleBaseUrl"`
	AnthropicBaseURL        string          `json:"anthropicBaseUrl"`
	Editable                bool            `json:"editable"`
	CreatedAt               timex.LocalTime `json:"createdAt"`
	UpdatedAt               timex.LocalTime `json:"updatedAt"`
}

func ToLlmProviderVO(p *model.LlmProvider, viewerUserID uint64, isAdmin bool) *LlmProviderVO {
	editable := providerEditable(p, viewerUserID, isAdmin)
	apiKey := ""
	if editable {
		apiKey = p.APIKey
	} else if p.APIKey != "" {
		apiKey = maskSecret(p.APIKey)
	}
	return &LlmProviderVO{
		ID:                      p.ID,
		UserID:                  p.UserID,
		Name:                    p.Name,
		Website:                 p.Website,
		APIKey:                  apiKey,
		OpenAICompatibleBaseURL: p.OpenAICompatibleBaseURL,
		AnthropicBaseURL:        p.AnthropicBaseURL,
		Editable:                editable,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
	}
}

func providerEditable(p *model.LlmProvider, viewerUserID uint64, isAdmin bool) bool {
	if p == nil {
		return false
	}
	if p.UserID == nil {
		return isAdmin
	}
	return *p.UserID == viewerUserID
}

func maskSecret(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// ApiKeyCreatedVO 创建 API Key 时返回。
type ApiKeyCreatedVO struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	KeyPrefix string `json:"keyPrefix"`
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
}

// ConversationVO 会话列表项。会话不绑定智能体，智能体在发送消息时按请求参数指定。
type ConversationVO struct {
	ID           uint64          `json:"id"`
	Title        string          `json:"title"`
	ModelID      uint64          `json:"modelId"`
	ModelName    string          `json:"modelName"`
	ProviderName string          `json:"providerName"`
	ClientID     string          `json:"clientId"`
	ProjectID    uint64          `json:"projectId"`
	Channel      string          `json:"channel"`
	TaskID       uint64          `json:"taskId"`
	Workspace    string          `json:"workspace"`
	ContextUsage map[string]any  `json:"contextUsage,omitempty"`
	CreatedAt    timex.LocalTime `json:"createdAt"`
	UpdatedAt    timex.LocalTime `json:"updatedAt"`
}

// MessageVO 消息。
type MessageVO struct {
	ID         uint64          `json:"id"`
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Metadata   string          `json:"metadata"`
	CreatedAt  timex.LocalTime `json:"createdAt"`
}

// ConversationDetailVO 会话详情。
type ConversationDetailVO struct {
	ConversationVO
	Messages []MessageVO `json:"messages"`
}

// ToolVO 对外返回的工具视图。
type ToolVO struct {
	ID          uint64          `json:"id"`
	UserID      *uint64         `json:"userId"`
	ClientID    string          `json:"clientId,omitempty"`
	ClientName  string          `json:"clientName,omitempty"`
	Name        string          `json:"name"`
	DisplayName string          `json:"displayName"`
	Description string          `json:"description"`
	Parameters  map[string]any  `json:"parameters"`
	Type        string          `json:"type"`
	Extra       map[string]any  `json:"extra"`
	Version     int64           `json:"version"`
	Builtin     bool            `json:"builtin"`
	Editable    bool            `json:"editable"`
	CreatedAt   timex.LocalTime `json:"createdAt,omitempty"`
	UpdatedAt   timex.LocalTime `json:"updatedAt,omitempty"`
}

func ToToolVO(t *model.Tool, viewerUserID uint64, isAdmin bool) *ToolVO {
	isBuiltin := t.Type == model.ToolTypeBuiltin
	editable := false
	switch {
	case t.UserID == nil:
		editable = isAdmin
	default:
		editable = *t.UserID == viewerUserID
	}
	return &ToolVO{
		ID:          t.ID,
		UserID:      t.UserID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		Description: t.Description,
		Parameters:  parseJSONMap(t.Parameters),
		Type:        t.Type,
		Extra:       parseJSONMap(t.Extra),
		Version:     t.Version,
		Builtin:     isBuiltin,
		Editable:    editable,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func parseJSONMap(raw string) map[string]any {
	out := map[string]any{}
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func ToConversationVO(c *model.Conversation, m *model.LlmModel) *ConversationVO {
	vo := &ConversationVO{
		ID:           c.ID,
		Title:        c.Title,
		ModelID:      c.ModelID,
		ClientID:     c.ClientID,
		Workspace:    c.Workspace,
		ContextUsage: parseJSONMap(c.ContextUsage),
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
	if c.Link != nil {
		vo.ProjectID = c.Link.ProjectID
		vo.Channel = c.Link.Channel
		vo.TaskID = c.Link.TaskID
	}
	if len(vo.ContextUsage) == 0 {
		vo.ContextUsage = nil
	}
	if m != nil {
		vo.ModelName = m.Name
		if m.Provider != nil {
			vo.ProviderName = m.Provider.Name
		}
	}
	return vo
}
