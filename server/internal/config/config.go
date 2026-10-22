package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/rest"
)

// Config 应用配置（嵌入 go-zero RestConf）。
type Config struct {
	rest.RestConf
	Database     DatabaseConfig     `json:",optional"`
	UserFeedback UserFeedbackConfig `json:",optional"`
	ClientAPI    ClientAPIConfig    `json:",optional"`
	Store        StoreConfig        `json:",optional"`
	Redis        RedisConfig        `json:",optional"`
	Cors         CorsConfig         `json:",optional"`
	// Skills 系统技能目录（默认 server/skills）。
	Skills SkillsConfig `json:",optional"`
	// Brand 各端顶栏展示的名称与 logo，由 GET /api/brand 下发。
	Brand BrandConfig `json:",optional"`
	// DevAuth 本机联调用的假登录。默认关闭，只在 etc/ade-dev.yaml 里打开。
	DevAuth DevAuthConfig `json:",optional"`
}

// DevAuthConfig 本机联调身份。打开后 /api/auth/login 不再访问 user-feedback，
// 直接发放固定令牌；服务端只用于本地开发，部署时务必保持 Enabled=false。
type DevAuthConfig struct {
	Enabled  bool   `json:",optional"`
	UserID   uint64 `json:",optional"`
	Username string `json:",optional"`
	Token    string `json:",optional"`
}

// 本机联调令牌缺省值。
const (
	DefaultDevToken    = "ade-dev-token"
	DefaultDevUsername = "本地开发者"
)

// 品牌缺省值。配置留空时回填，保证各端始终有可展示的名称和 logo。
const (
	DefaultBrandName = "ADE"
	DefaultBrandLogo = "⬢"
)

// BrandConfig 系统品牌。Logo 为 emoji/文本，或 http(s) 图片地址、以 / 开头的图片路径。
type BrandConfig struct {
	Name string `json:",optional"`
	Logo string `json:",optional"`
}

// Normalize 去掉空白，空字段回填默认品牌。
func (b *BrandConfig) Normalize() {
	if b == nil {
		return
	}
	b.Name = strings.TrimSpace(b.Name)
	if b.Name == "" {
		b.Name = DefaultBrandName
	}
	b.Logo = strings.TrimSpace(b.Logo)
	if b.Logo == "" {
		b.Logo = DefaultBrandLogo
	}
}

// SkillsConfig 系统技能仓库配置。
type SkillsConfig struct {
	Dir string `json:",optional"`
}

// CorsConfig 跨域配置，供安卓壳等非同源客户端调用接口。
type CorsConfig struct {
	// Enabled 是否放行跨域；默认关闭，同源网页与桌面端无需开启。
	Enabled bool `json:",optional"`
	// AllowedOrigins 允许的来源，默认 ["*"]（不限来源）。
	AllowedOrigins []string `json:",optional"`
}

// StoreConfig 会话附件等本地文件存储。
type StoreConfig struct {
	StorageDir      string `json:",optional"`
	MaxPackageBytes int64  `json:",optional"`
}

// ClientAPIConfig 客户端 API 配置。
type ClientAPIConfig struct {
	InvokeTimeout     time.Duration `json:",optional"`
	HeartbeatInterval time.Duration `json:",optional"`
}

// RedisConfig 客户端 catalog 等缓存。
type RedisConfig struct {
	Addr       string        `json:",optional"`
	Password   string        `json:",optional"`
	DB         int           `json:",optional"`
	CatalogTTL time.Duration `json:",optional"`
}

// UserFeedbackConfig user-feedback 身份服务与 OAuth2 客户端配置。
// 本项目不再自建账号，统一以 user-feedback 为身份源。
type UserFeedbackConfig struct {
	// BaseURL user-feedback 服务根地址，如 http://127.0.0.1:8001。
	BaseURL string `json:",optional"`
	// ClientID / ClientSecret 本项目在 user-feedback 注册的 OAuth2 客户端凭证。
	ClientID     string `json:",optional"`
	ClientSecret string `json:",optional"`
	// RedirectURI OAuth2 授权码回调地址（指向本服务前端页面）。
	RedirectURI string `json:",optional"`
	// AdminRoleCode 管理员角色编码；用户角色包含该编码即视为管理员。
	// 默认 agent_admin（user-feedback 论坛超级管理员）。
	AdminRoleCode string `json:",optional"`
	// Timeout 调用 user-feedback 的 HTTP 超时。
	Timeout time.Duration `json:",optional"`
}

// DatabaseConfig 数据库配置。
type DatabaseConfig struct {
	Driver          string        `json:",optional"`
	Host            string        `json:",optional"`
	Port            int           `json:",optional"`
	User            string        `json:",optional"`
	Password        string        `json:",optional"`
	Name            string        `json:",optional"`
	Charset         string        `json:",optional"`
	SqlitePath      string        `json:",optional"`
	MaxIdleConns    int           `json:",optional"`
	MaxOpenConns    int           `json:",optional"`
	ConnMaxLifetime time.Duration `json:",optional"`
	LogLevel        string        `json:",optional"`
	AutoMigrate     bool          `json:",optional"`
}

// DSN 返回 MySQL DSN。
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Name, c.Charset)
}

// Validate 校验关键配置项。
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	switch strings.ToLower(c.Database.Driver) {
	case "mysql", "sqlite", "":
		if c.Database.Driver == "" {
			c.Database.Driver = "sqlite"
		}
	default:
		return fmt.Errorf("unsupported database driver: %s (mysql|sqlite)", c.Database.Driver)
	}
	if strings.EqualFold(c.Database.Driver, "mysql") && c.Database.Name == "" {
		return fmt.Errorf("database.name is required for mysql")
	}
	if c.UserFeedback.Timeout <= 0 {
		c.UserFeedback.Timeout = 10 * time.Second
	}
	if strings.TrimSpace(c.UserFeedback.AdminRoleCode) == "" {
		c.UserFeedback.AdminRoleCode = "agent_admin"
	}
	if c.ClientAPI.InvokeTimeout <= 0 {
		c.ClientAPI.InvokeTimeout = 60 * time.Second
	}
	if c.ClientAPI.HeartbeatInterval <= 0 {
		c.ClientAPI.HeartbeatInterval = 30 * time.Second
	}
	if c.Database.SqlitePath == "" {
		c.Database.SqlitePath = "data/app.db"
	}
	if strings.TrimSpace(c.Skills.Dir) == "" {
		c.Skills.Dir = "skills"
	}
	if strings.TrimSpace(c.Store.StorageDir) == "" {
		c.Store.StorageDir = "data/store"
	}
	if c.Store.MaxPackageBytes <= 0 {
		c.Store.MaxPackageBytes = 20 << 20
	}
	if c.DevAuth.Enabled {
		if c.DevAuth.UserID == 0 {
			c.DevAuth.UserID = 1
		}
		if strings.TrimSpace(c.DevAuth.Username) == "" {
			c.DevAuth.Username = DefaultDevUsername
		}
		if strings.TrimSpace(c.DevAuth.Token) == "" {
			c.DevAuth.Token = DefaultDevToken
		}
	}
	c.Brand.Normalize()
	return nil
}
