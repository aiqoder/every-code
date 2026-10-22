package model

import "chengzi-agent-code-server/pkg/timex"

// Conversation 聊天会话。不绑定智能体，也不直接保存项目、工位和任务。
// 归属写在 ConversationLink。智能体在发送消息时按请求参数解析。
type Conversation struct {
	ID           uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID       uint64          `gorm:"not null;index;default:0;comment:所属用户，0 表示没有" json:"userId"`
	Title        string          `gorm:"size:256;not null;default:'';comment:会话标题" json:"title"`
	ModelID      uint64          `gorm:"not null;index;comment:使用的模型" json:"modelId"`
	ClientID     string          `gorm:"size:128;not null;default:'';comment:桌面端标识" json:"clientId"`
	Workspace    string          `gorm:"size:1024;not null;default:'';comment:工作目录" json:"workspace"`
	ContextUsage string          `gorm:"type:text;not null;comment:上下文用量 JSON" json:"contextUsage"`
	CreatedAt    timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt    timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`

	Model    *LlmModel         `gorm:"foreignKey:ModelID" json:"model,omitempty"`
	Messages []Message         `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
	Link     *ConversationLink `gorm:"foreignKey:ConversationID" json:"-"`
}

// ConversationLink 会话归属中间表。一个会话最多一行：属于哪个项目、哪个工位，以及哪条任务。
// TaskID 为 0 表示这条会话不属于某条任务，例如环境检测和执行、测试、部署工位。
type ConversationLink struct {
	ID             uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	ConversationID uint64          `gorm:"not null;uniqueIndex;comment:会话，一个会话最多一行" json:"conversationId"`
	ProjectID      uint64          `gorm:"not null;index:idx_conversation_link_scope,priority:1;comment:所属项目" json:"projectId"`
	Channel        string          `gorm:"size:32;not null;default:'';index:idx_conversation_link_scope,priority:2;comment:工位。create 创建，execute 执行，deploy 部署" json:"channel"`
	TaskID         uint64          `gorm:"not null;default:0;index;comment:所属任务，0 表示不属于某条任务" json:"taskId"`
	CreatedAt      timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt      timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (ConversationLink) TableName() string {
	return "conversation_links"
}

func (ConversationLink) TableComment() string {
	return "会话归属，一个会话对应一个项目、工位和可选任务"
}

func (Conversation) TableName() string {
	return "conversations"
}

func (Conversation) TableComment() string {
	return "聊天会话，不直接保存项目、工位和任务"
}

// Message 会话消息。
type Message struct {
	ID             uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	ConversationID uint64          `gorm:"not null;index;comment:所属会话" json:"conversationId"`
	Role           string          `gorm:"size:32;not null;comment:角色。user、assistant、system 或 tool" json:"role"`
	Content        string          `gorm:"type:text;not null;comment:消息正文" json:"content"`
	ToolCallID     string          `gorm:"size:128;not null;default:'';comment:工具调用编号" json:"toolCallId"`
	ToolName       string          `gorm:"size:128;not null;default:'';comment:工具名称" json:"toolName"`
	Metadata       string          `gorm:"type:text;not null;comment:附加信息 JSON" json:"metadata"`
	CreatedAt      timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt      timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (Message) TableName() string {
	return "messages"
}

func (Message) TableComment() string {
	return "会话消息"
}
