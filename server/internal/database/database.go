package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"chengzi-agent-code-server/internal/config"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/pkg/timex"
)

// New 根据配置创建 GORM 连接；sqlite 使用纯 Go 驱动（无需 CGO）。
func New(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(parseLogLevel(cfg.LogLevel)),
	}

	var (
		db  *gorm.DB
		err error
	)

	switch strings.ToLower(cfg.Driver) {
	case "mysql":
		db, err = gorm.Open(mysql.Open(cfg.DSN()), gormCfg)
	case "sqlite":
		if err = ensureDir(cfg.SqlitePath); err != nil {
			return nil, err
		}
		db, err = gorm.Open(sqlite.Open(cfg.SqlitePath), gormCfg)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if cfg.AutoMigrate {
		if err = db.AutoMigrate(
			&model.ApiKey{},
			&model.LlmProvider{},
			&model.LlmModel{},
			&model.Tool{},
			&model.Conversation{},
			&model.ConversationLink{},
			&model.Message{},
			&model.TokenUsage{},
			&model.Project{},
			&model.Task{},
			&model.Todo{},
			&model.DeployHost{},
			&model.DeployTask{},
			&model.Release{},
			&model.CommandAudit{},
			&model.SkillBinding{},
			&model.Scene{},
		); err != nil {
			return nil, fmt.Errorf("auto migrate: %w", err)
		}
		for _, name := range []string{"ade_task_timelines", "ade_run_events", "ade_runs"} {
			if !db.Migrator().HasTable(name) {
				continue
			}
			if err = db.Migrator().DropTable(name); err != nil {
				return nil, fmt.Errorf("drop %s: %w", name, err)
			}
		}
		if err = migrateConversationLinks(db); err != nil {
			return nil, err
		}
		if err = ensureBuiltinTools(db); err != nil {
			return nil, err
		}
		if err = applyTableComments(db,
			model.ApiKey{},
			model.LlmProvider{},
			model.LlmModel{},
			model.Tool{},
			model.Conversation{},
			model.ConversationLink{},
			model.Message{},
			model.TokenUsage{},
			model.Project{},
			model.Task{},
			model.Todo{},
			model.DeployHost{},
			model.DeployTask{},
			model.Release{},
			model.CommandAudit{},
			model.SkillBinding{},
			model.Scene{},
		); err != nil {
			return nil, err
		}
	}
	return db, nil
}

// tableCommenter 表名和表注释都由模型自己提供。
type tableCommenter interface {
	TableName() string
	TableComment() string
}

// applyTableComments 把表注释写进 MySQL。SQLite 没有表注释，直接跳过。
func applyTableComments(db *gorm.DB, tables ...tableCommenter) error {
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	for _, table := range tables {
		name := table.TableName()
		comment := strings.TrimSpace(table.TableComment())
		if name == "" || comment == "" {
			continue
		}
		if err := db.Exec("ALTER TABLE `"+name+"` COMMENT = ?", comment).Error; err != nil {
			return fmt.Errorf("table comment %s: %w", name, err)
		}
	}
	return nil
}

// ensureBuiltinTools 补齐管理员内置工具。已存在同名记录时不覆盖。
func ensureBuiltinTools(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.Tool{}) {
		return nil
	}
	seeds := []model.Tool{
		{
			Name:        "todowriter",
			DisplayName: "待办列表",
			Description: "为当前会话创建并更新结构化任务列表，随进度勾选步骤。",
			Type:        model.ToolTypeBuiltin,
		},
		{
			Name:        "Task",
			DisplayName: "子代理",
			Description: "把独立子任务交给 generalPurpose 或 explore 子代理，同一轮多个 Task 按调用顺序依次执行。",
			Type:        model.ToolTypeBuiltin,
		},
		{
			Name:        "AskQuestion",
			DisplayName: "向用户提问",
			Description: "向用户收集结构化选择题答案，等待提交后再继续。",
			Type:        model.ToolTypeBuiltin,
		},
	}
	now := timex.LocalTime(time.Now())
	for _, seed := range seeds {
		var count int64
		if err := db.Model(&model.Tool{}).
			Where("name = ? AND user_id IS NULL", seed.Name).
			Count(&count).Error; err != nil {
			return fmt.Errorf("count builtin %s: %w", seed.Name, err)
		}
		if count > 0 {
			continue
		}
		seed.Parameters = "{}"
		seed.Extra = "{}"
		seed.CreatedAt = now
		seed.UpdatedAt = now
		if err := db.Create(&seed).Error; err != nil {
			return fmt.Errorf("create builtin %s: %w", seed.Name, err)
		}
	}
	return nil
}

func ensureDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func parseLogLevel(level string) logger.LogLevel {
	switch strings.ToLower(level) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "info":
		return logger.Info
	case "warn", "warning":
		return logger.Warn
	default:
		return logger.Warn
	}
}

// Ping 探测数据库连通性。
func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Close 关闭底层连接池。
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
