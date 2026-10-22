package model

import "chengzi-agent-code-server/pkg/timex"

const (
	ToolTypeHTTP    = "http"
	ToolTypeClient  = "client" // 历史镜像类型，不再写入；列表/聊天已排除
	ToolTypeBuiltin = "builtin"
)

// Tool 可配置工具表。UserID 为 nil 表示管理员工具，非 nil 表示用户私有工具。
// 客户端工具通过 catalog 上报，不再落库。
type Tool struct {
	ID          uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID      *uint64         `gorm:"index;comment:所属用户，空表示管理员工具" json:"userId"`
	Name        string          `gorm:"size:64;not null;comment:英文名称" json:"name"`
	DisplayName string          `gorm:"size:128;not null;comment:中文名称" json:"displayName"`
	Description string          `gorm:"size:2048;not null;default:'';comment:用途说明" json:"description"`
	Parameters  string          `gorm:"type:text;not null;comment:参数 JSON Schema" json:"-"`
	Type        string          `gorm:"size:32;not null;default:'http';comment:类型。http 或 builtin，client 为历史类型不再写入" json:"type"`
	Extra       string          `gorm:"type:text;not null;comment:附加配置 JSON，HTTP 工具含请求方法、地址和请求头" json:"-"`
	Version     int64           `gorm:"not null;default:0;comment:历史字段，客户端工具已不再落库" json:"version"`
	CreatedAt   timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt   timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (Tool) TableName() string {
	return "tools"
}

func (Tool) TableComment() string {
	return "可配置工具，客户端工具不入库"
}
