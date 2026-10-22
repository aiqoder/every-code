package model

import "chengzi-agent-code-server/pkg/timex"

// SkillBinding 系统技能与智能体的绑定。未配置或 AgentID 为空表示该技能不参与任何智能体。
type SkillBinding struct {
	ID        uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	SkillID   string          `gorm:"size:128;not null;uniqueIndex;comment:系统技能 id" json:"skillId"`
	AgentID   string          `gorm:"size:128;not null;default:'';comment:绑定的智能体，空表示不参与任何智能体" json:"agentId"`
	CreatedAt timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (SkillBinding) TableName() string {
	return "skill_bindings"
}

func (SkillBinding) TableComment() string {
	return "系统技能与智能体的绑定"
}
