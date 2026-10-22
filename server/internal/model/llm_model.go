package model

import "chengzi-agent-code-server/pkg/timex"

// 模型输入能力可选项（后端硬编码，不落库维护）。
const (
	InputCapabilityText  = "text"
	InputCapabilityImage = "image"
	InputCapabilityPDF   = "pdf"
)

// InputCapabilityOptions 返回硬编码的输入能力选项。
func InputCapabilityOptions() []string {
	return []string{InputCapabilityText, InputCapabilityImage, InputCapabilityPDF}
}

// LlmModel 大模型维护表。
// ReasoningOptions / InputCapabilities 在库中以逗号分隔字符串存储。
// OutputContextLimit 为 0 表示不限制。
type LlmModel struct {
	ID                 uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	ProviderID         uint64          `gorm:"not null;index;comment:所属供应商" json:"providerId"`
	Name               string          `gorm:"size:128;not null;comment:模型名称" json:"name"`
	Attachment         bool            `gorm:"not null;default:false;comment:是否支持附件" json:"attachment"`
	Reasoning          bool            `gorm:"not null;default:false;comment:是否支持推理" json:"reasoning"`
	ReasoningOptions   string          `gorm:"size:1024;not null;default:'';comment:推理档位，逗号分隔" json:"-"`
	Temperature        bool            `gorm:"not null;default:false;comment:是否支持温度" json:"temperature"`
	InputCapabilities  string          `gorm:"size:128;not null;default:'';comment:输入能力，逗号分隔，如 text、image、pdf" json:"-"`
	InputContextLimit  int64           `gorm:"not null;default:200000;comment:输入上下文上限" json:"inputContextLimit"`
	OutputContextLimit int64           `gorm:"not null;default:0;comment:输出上下文上限，0 表示不限制" json:"outputContextLimit"`
	InputPrice         float64         `gorm:"type:decimal(16,8);not null;default:0;comment:输入单价，美元每百万 token" json:"inputPrice"`
	OutputPrice        float64         `gorm:"type:decimal(16,8);not null;default:0;comment:输出单价，美元每百万 token" json:"outputPrice"`
	CachePrice         float64         `gorm:"type:decimal(16,8);not null;default:0;comment:缓存读取单价，美元每百万 token" json:"cachePrice"`
	CreatedAt          timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt          timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`

	Provider *LlmProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

func (LlmModel) TableName() string {
	return "llm_models"
}

func (LlmModel) TableComment() string {
	return "大模型"
}
