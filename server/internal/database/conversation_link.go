package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
)

// conversationLegacyColumns 只用来探测旧库里还留在 conversations 上的项目和通道列。
type conversationLegacyColumns struct {
	ProjectID uint64 `gorm:"column:project_id"`
	Channel   string `gorm:"column:channel"`
}

func (conversationLegacyColumns) TableName() string {
	return "conversations"
}

// migrateConversationLinks 把 conversations 上的 project_id、channel 搬到中间表后删掉这两列。
func migrateConversationLinks(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("conversations") {
		return nil
	}
	legacy := &conversationLegacyColumns{}
	hasProject := db.Migrator().HasColumn(legacy, "ProjectID")
	hasChannel := db.Migrator().HasColumn(legacy, "Channel")
	if !hasProject && !hasChannel {
		return nil
	}
	if err := db.AutoMigrate(&model.ConversationLink{}); err != nil {
		return fmt.Errorf("migrate conversation links: %w", err)
	}
	if err := backfillConversationLinks(db, hasProject, hasChannel); err != nil {
		return err
	}
	if hasProject {
		if err := dropColumn(db, "conversations", "project_id"); err != nil {
			return err
		}
	}
	if hasChannel {
		if err := dropColumn(db, "conversations", "channel"); err != nil {
			return err
		}
	}
	return nil
}

func backfillConversationLinks(db *gorm.DB, hasProject, hasChannel bool) error {
	projectExpr := "0"
	channelExpr := "''"
	conds := make([]string, 0, 2)
	if hasProject {
		projectExpr = "COALESCE(project_id, 0)"
		conds = append(conds, "COALESCE(project_id, 0) > 0")
	}
	if hasChannel {
		channelExpr = "COALESCE(channel, '')"
		conds = append(conds, "COALESCE(channel, '') <> ''")
	}
	stmt := fmt.Sprintf(`
INSERT INTO conversation_links (conversation_id, project_id, channel, task_id, created_at, updated_at)
SELECT id, %s, %s, 0, created_at, updated_at
FROM conversations
WHERE (%s)
AND NOT EXISTS (
  SELECT 1 FROM conversation_links WHERE conversation_links.conversation_id = conversations.id
)`, projectExpr, channelExpr, strings.Join(conds, " OR "))
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("backfill conversation links: %w", err)
	}
	return nil
}

func dropColumn(db *gorm.DB, table, column string) error {
	switch db.Dialector.Name() {
	case "mysql":
		return dropMySQLColumn(db, table, column)
	default:
		return dropSQLiteColumn(db, table, column)
	}
}

func dropSQLiteColumn(db *gorm.DB, table, column string) error {
	type indexRow struct {
		Name string
		SQL  string `gorm:"column:sql"`
	}
	var rows []indexRow
	if err := db.Raw(
		`SELECT name, sql FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND sql IS NOT NULL`,
		table,
	).Scan(&rows).Error; err != nil {
		return fmt.Errorf("list indexes on %s: %w", table, err)
	}
	needle := strings.ToLower(column)
	for _, row := range rows {
		if !strings.Contains(strings.ToLower(row.SQL), needle) {
			continue
		}
		if err := db.Exec("DROP INDEX " + quoteIdent(db, row.Name)).Error; err != nil {
			return fmt.Errorf("drop index %s: %w", row.Name, err)
		}
	}
	stmt := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", quoteIdent(db, table), quoteIdent(db, column))
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("drop %s.%s: %w", table, column, err)
	}
	return nil
}

func dropMySQLColumn(db *gorm.DB, table, column string) error {
	var names []string
	err := db.Raw(`
SELECT DISTINCT INDEX_NAME FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? AND INDEX_NAME <> 'PRIMARY'`,
		table, column,
	).Scan(&names).Error
	if err != nil {
		return fmt.Errorf("list indexes on %s.%s: %w", table, column, err)
	}
	for _, name := range names {
		stmt := fmt.Sprintf("ALTER TABLE %s DROP INDEX %s", quoteIdent(db, table), quoteIdent(db, name))
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("drop index %s: %w", name, err)
		}
	}
	stmt := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", quoteIdent(db, table), quoteIdent(db, column))
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("drop %s.%s: %w", table, column, err)
	}
	return nil
}

func quoteIdent(db *gorm.DB, name string) string {
	if db.Dialector.Name() == "mysql" {
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
