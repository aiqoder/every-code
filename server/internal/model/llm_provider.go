package model

import "chengzi-agent-code-server/pkg/timex"

// LlmProvider 大模型供应商。UserID 为 nil 表示平台供应商，非 nil 表示用户私有。
type LlmProvider struct {
	ID                      uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID                  *uint64         `gorm:"index;comment:所属用户，空表示平台供应商" json:"userId"`
	Name                    string          `gorm:"size:128;not null;comment:供应商名称" json:"name"`
	Website                 string          `gorm:"size:512;not null;default:'';comment:官网" json:"website"`
	APIKey                  string          `gorm:"column:api_key;size:512;not null;default:'';comment:接口密钥" json:"apiKey"`
	OpenAICompatibleBaseURL string          `gorm:"column:openai_compatible_base_url;size:512;not null;default:'';comment:OpenAI 兼容接口地址" json:"openaiCompatibleBaseUrl"`
	AnthropicBaseURL        string          `gorm:"column:anthropic_base_url;size:512;not null;default:'';comment:Anthropic 接口地址" json:"anthropicBaseUrl"`
	CreatedAt               timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt               timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (LlmProvider) TableName() string {
	return "llm_providers"
}

func (LlmProvider) TableComment() string {
	return "大模型供应商"
}
