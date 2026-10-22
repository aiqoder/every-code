package model

import "chengzi-agent-code-server/pkg/timex"

// TokenUsage 一次 LLM 调用的用量流水。
type TokenUsage struct {
	ID                  uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID              uint64          `gorm:"not null;index;default:0;comment:所属用户，0 表示没有" json:"userId"`
	ConversationID      uint64          `gorm:"not null;index;default:0;comment:所属会话，0 表示没有" json:"conversationId"`
	ModelID             uint64          `gorm:"not null;index;default:0;comment:模型，0 表示没有" json:"modelId"`
	ModelName           string          `gorm:"size:128;not null;default:'';comment:模型名称快照" json:"modelName"`
	ProviderID          uint64          `gorm:"not null;default:0;comment:供应商，0 表示没有" json:"providerId"`
	PromptTokens        int64           `gorm:"not null;default:0;comment:新增输入 token" json:"promptTokens"`
	CompletionTokens    int64           `gorm:"not null;default:0;comment:输出 token" json:"completionTokens"`
	CacheCreationTokens int64           `gorm:"not null;default:0;comment:缓存写入 token" json:"cacheCreationTokens"`
	CacheReadTokens     int64           `gorm:"not null;default:0;comment:缓存读取 token" json:"cacheReadTokens"`
	Cost                float64         `gorm:"type:decimal(18,8);not null;default:0;comment:本次费用，美元" json:"cost"`
	CreatedAt           timex.LocalTime `gorm:"index;comment:创建时间" json:"createdAt"`
}

func (TokenUsage) TableName() string {
	return "token_usages"
}

func (TokenUsage) TableComment() string {
	return "一次大模型调用的用量流水"
}
