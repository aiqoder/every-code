package model

import "chengzi-agent-code-server/pkg/timex"

// ApiKey 用户远程调用凭证（明文可按需查看）。
type ApiKey struct {
	ID         uint64           `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID     uint64           `gorm:"not null;index;comment:所属用户" json:"userId"`
	Name       string           `gorm:"size:64;not null;comment:凭证名称" json:"name"`
	KeyPrefix  string           `gorm:"size:16;not null;comment:密钥前缀，用于展示" json:"keyPrefix"`
	KeyHash    string           `gorm:"size:128;not null;uniqueIndex;comment:密钥哈希" json:"-"`
	KeyPlain   string           `gorm:"column:key_plain;size:128;not null;default:'';comment:密钥明文" json:"-"`
	LastUsedAt *timex.LocalTime `gorm:"comment:最近使用时间" json:"lastUsedAt,omitempty"`
	CreatedAt  timex.LocalTime  `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt  timex.LocalTime  `gorm:"comment:更新时间" json:"updatedAt"`
}

func (ApiKey) TableName() string {
	return "api_keys"
}

func (ApiKey) TableComment() string {
	return "用户远程调用凭证"
}
