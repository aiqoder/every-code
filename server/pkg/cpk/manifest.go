// Package cpk 实现 chengzi 智能体安装包（.cpk）的解析、校验、解包与打包。
//
// .cpk 本质为 zip 压缩包，目录约定：
//
//	agent.cpk
//	├── manifest.yaml          # 必需，包清单
//	├── prompt.md              # 系统提示词（manifest.systemPrompt 指向）
//	├── icon.png               # 可选头像（manifest.avatar 指向）
//	├── server/                # 可选；服务端代码（打包时剔除，仅开发期存在）
//	└── skills/<skill-name>/
//	    ├── SKILL.md           # 必需，YAML frontmatter
//	    ├── scripts/           # 桌面端脚本（入包）
//	    ├── references/  assets/
package cpk

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v2"
)

// ManifestFile 包内清单文件名。
const ManifestFile = "manifest.yaml"

// ChecksumFile 打包时生成的 sha256 清单文件名。
const ChecksumFile = "manifest.sha256"

// maxManifestSteps 与服务端 maxAgentSteps 对齐。
const maxManifestSteps = 999

// manifestIDPattern 包 ID 规范：小写字母/数字开头，可含 - _ .。
var manifestIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// envNamePattern 环境变量名：字母或下划线开头，仅含字母、数字、下划线，最长 64 位。
var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

// Manifest 智能体包清单。
type Manifest struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Version     string `yaml:"version" json:"version"`
	Author      string `yaml:"author" json:"author,omitempty"`
	// MinClientVersion 可选，要求桌面端最低版本。
	MinClientVersion string `yaml:"minClientVersion" json:"minClientVersion,omitempty"`
	// SystemPrompt 系统提示词文件路径（相对包根目录），如 prompt.md。
	SystemPrompt string `yaml:"systemPrompt" json:"systemPrompt"`
	// Avatar 头像文件路径（相对包根目录），可选。
	Avatar string `yaml:"avatar" json:"avatar,omitempty"`
	// MaxSteps 单轮最大模型调用步数；0 表示使用服务端默认值。
	MaxSteps int `yaml:"maxSteps" json:"maxSteps,omitempty"`
	// Skills 技能目录名列表（相对 skills/ 下）；为空表示打包 skills/ 下全部技能。
	Skills []string   `yaml:"skills" json:"skills,omitempty"`
	Server ServerConf `yaml:"server" json:"server"`
	// Env 用户可配置的环境变量。变量名和含义由开发者声明，取值由用户在桌面端填写。
	Env []EnvVar `yaml:"env" json:"env,omitempty"`
}

// ServerConf 开发者服务器配置（server 端脚本部署位置）。
type ServerConf struct {
	// BaseURL 开发者服务器地址，如 https://api.example.com。
	BaseURL string `yaml:"baseUrl" json:"baseUrl"`
}

// EnvTypeString / EnvTypeList 是 env.type 的合法取值；缺省视为 string。
const (
	EnvTypeString = "string"
	EnvTypeList   = "list"
)

const maxEnvListItems = 16

// EnvVar 开发者声明、由用户填写的环境变量。
type EnvVar struct {
	// Name 环境变量名，脚本按此名称读取。
	Name string `yaml:"name" json:"name"`
	// Label 展示给用户的说明；为空时桌面端显示 Name。
	Label string `yaml:"label" json:"label,omitempty"`
	// Secret 为 true 时按密文输入（仅 type=string 生效；list 的密文由列级 secret 控制）。
	Secret bool `yaml:"secret" json:"secret,omitempty"`
	// Required 为 true 时用户必须填写后脚本才会拿到该变量。
	// type=list 时表示至少需要一行有效数据。
	Required bool `yaml:"required" json:"required,omitempty"`
	// Type 输入形态：string（默认单行）或 list（结构化清单）。
	Type string `yaml:"type" json:"type,omitempty"`
	// Items type=list 时的列定义；type=string 时必须为空。
	Items []EnvItem `yaml:"items" json:"items,omitempty"`
}

// EnvItem 结构化清单的一列。
type EnvItem struct {
	// Name 列键名（序列化顺序用，不是环境变量名）。
	Name string `yaml:"name" json:"name"`
	// Label 展示给用户的列标题；为空时桌面端显示 Name。
	Label string `yaml:"label" json:"label,omitempty"`
	// Secret 为 true 时该列按密文输入。
	Secret bool `yaml:"secret" json:"secret,omitempty"`
	// Required 为 true 时该列在有效行中必须非空。
	Required bool `yaml:"required" json:"required,omitempty"`
}

// NormalizedType 返回规范化后的 type（空视为 string）。
func (ev EnvVar) NormalizedType() string {
	t := strings.TrimSpace(strings.ToLower(ev.Type))
	if t == "" {
		return EnvTypeString
	}
	return t
}

// ParseManifest 解析并校验 manifest.yaml 内容。
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest.yaml 解析失败: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate 校验清单字段合法性，并填充默认值。
func (m *Manifest) Validate() error {
	m.ID = strings.TrimSpace(m.ID)
	m.Name = strings.TrimSpace(m.Name)
	m.Description = strings.TrimSpace(m.Description)
	m.Version = strings.TrimSpace(m.Version)
	m.SystemPrompt = strings.TrimSpace(m.SystemPrompt)
	m.Server.BaseURL = strings.TrimRight(strings.TrimSpace(m.Server.BaseURL), "/")
	for i := range m.Env {
		m.Env[i].Name = strings.TrimSpace(m.Env[i].Name)
		m.Env[i].Label = strings.TrimSpace(m.Env[i].Label)
		m.Env[i].Type = strings.TrimSpace(strings.ToLower(m.Env[i].Type))
		for j := range m.Env[i].Items {
			m.Env[i].Items[j].Name = strings.TrimSpace(m.Env[i].Items[j].Name)
			m.Env[i].Items[j].Label = strings.TrimSpace(m.Env[i].Items[j].Label)
		}
	}

	if !manifestIDPattern.MatchString(m.ID) {
		return fmt.Errorf("manifest: id 无效（需小写字母/数字开头，可含 - _ .，最长 64 位）: %q", m.ID)
	}
	if m.Name == "" {
		return fmt.Errorf("manifest: name 不能为空")
	}
	if len(m.Description) > 512 {
		return fmt.Errorf("manifest: description 不能超过 512 字")
	}
	if _, err := ParseVersion(m.Version); err != nil {
		return fmt.Errorf("manifest: version 无效: %w", err)
	}
	if m.MinClientVersion != "" {
		if _, err := ParseVersion(m.MinClientVersion); err != nil {
			return fmt.Errorf("manifest: minClientVersion 无效: %w", err)
		}
	}
	if m.SystemPrompt == "" {
		return fmt.Errorf("manifest: systemPrompt 不能为空（指向提示词文件，如 prompt.md）")
	}
	if err := validateRelPath(m.SystemPrompt); err != nil {
		return fmt.Errorf("manifest: systemPrompt %w", err)
	}
	if m.Avatar != "" {
		if err := validateRelPath(m.Avatar); err != nil {
			return fmt.Errorf("manifest: avatar %w", err)
		}
	}
	if m.MaxSteps < 0 || m.MaxSteps > maxManifestSteps {
		return fmt.Errorf("manifest: maxSteps 需在 0-%d 之间", maxManifestSteps)
	}
	if err := m.validateEnv(); err != nil {
		return err
	}
	if m.Server.BaseURL != "" {
		if !strings.HasPrefix(m.Server.BaseURL, "https://") && !strings.HasPrefix(m.Server.BaseURL, "http://") {
			return fmt.Errorf("manifest: server.baseUrl 需以 http:// 或 https:// 开头")
		}
	}
	for _, s := range m.Skills {
		if err := validateSkillDirName(s); err != nil {
			return err
		}
	}
	return nil
}

// validateEnv 校验开发者声明的环境变量名与 list schema。
func (m *Manifest) validateEnv() error {
	seen := make(map[string]string, len(m.Env))
	for i := range m.Env {
		ev := &m.Env[i]
		if !envNamePattern.MatchString(ev.Name) {
			return fmt.Errorf("manifest: env.name 无效（字母或下划线开头，仅含字母、数字、下划线，最长 64 位）: %q", ev.Name)
		}
		if strings.HasPrefix(strings.ToUpper(ev.Name), "CHENGZI_") {
			return fmt.Errorf("manifest: env.name 不能使用 CHENGZI_ 前缀: %q", ev.Name)
		}
		key := strings.ToUpper(ev.Name)
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("manifest: env.name 重复: %q 与 %q", prev, ev.Name)
		}
		seen[key] = ev.Name
		if utf8.RuneCountInString(ev.Label) > 64 {
			return fmt.Errorf("manifest: env.label 不能超过 64 字")
		}
		typ := ev.NormalizedType()
		switch typ {
		case EnvTypeString:
			if len(ev.Items) > 0 {
				return fmt.Errorf("manifest: env %s 为 string 时不能声明 items", ev.Name)
			}
			ev.Type = "" // 缺省不写进对外表示
			ev.Items = nil
		case EnvTypeList:
			if err := validateEnvItems(ev); err != nil {
				return err
			}
			ev.Type = EnvTypeList
			ev.Secret = false // list 密文由列级 secret 控制
		default:
			return fmt.Errorf("manifest: env.type 无效（仅支持 string 或 list）: %q", ev.Type)
		}
	}
	return nil
}

func validateEnvItems(ev *EnvVar) error {
	if len(ev.Items) == 0 {
		return fmt.Errorf("manifest: env %s 为 list 时必须声明 items", ev.Name)
	}
	if len(ev.Items) > maxEnvListItems {
		return fmt.Errorf("manifest: env %s 的 items 不能超过 %d 列", ev.Name, maxEnvListItems)
	}
	seen := make(map[string]struct{}, len(ev.Items))
	for i, item := range ev.Items {
		if !envNamePattern.MatchString(item.Name) {
			return fmt.Errorf("manifest: env %s 的 items[%d].name 无效: %q", ev.Name, i, item.Name)
		}
		key := strings.ToUpper(item.Name)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("manifest: env %s 的 items.name 重复: %q", ev.Name, item.Name)
		}
		seen[key] = struct{}{}
		if utf8.RuneCountInString(item.Label) > 64 {
			return fmt.Errorf("manifest: env %s 的 items.label 不能超过 64 字", ev.Name)
		}
	}
	return nil
}

// validateRelPath 校验包内相对路径安全（不允许绝对路径与 .. 穿越）。
func validateRelPath(p string) error {
	if p == "" {
		return fmt.Errorf("路径不能为空")
	}
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) {
		return fmt.Errorf("不允许绝对路径: %q", p)
	}
	for _, seg := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." {
			return fmt.Errorf("不允许包含 .. 路径穿越: %q", p)
		}
	}
	return nil
}

// validateSkillDirName 校验技能目录名。
func validateSkillDirName(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("manifest: skills 中存在空目录名")
	}
	if strings.ContainsAny(s, `/\`) || s == "." || s == ".." {
		return fmt.Errorf("manifest: 技能目录名无效: %q", s)
	}
	return nil
}
