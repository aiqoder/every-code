package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
)

// obsoleteAdeTaskColumns 已从 Task 模型删除、启动时要从 ade_tasks 清掉的列。
var obsoleteAdeTaskColumns = []string{
	"coder_agent_id",
	"verifier_agent_id",
	"reviewer_agent_id",
	"planned",
	"error_kind",
	"error_agent_id",
	"error_message",
	"error_at",
	"client_id",
	"status",
	"mode",
	"branch",
	"worktree",
	"merged_dev",
	"merged_main",
	"retries",
}

// obsoleteAdeTodoColumns 已从 Todo 模型删除、启动时要从 ade_todos 清掉的列。
var obsoleteAdeTodoColumns = []string{
	"acceptance",
	"depends_on",
	"parent_id",
	"retry",
	"conversation_id",
}

// migrateDropObsoleteTaskColumns 按真实表结构删掉 ade_tasks 上已废弃的列，避免 AutoMigrate 残留。
func migrateDropObsoleteTaskColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.Task{}) {
		return nil
	}
	present, err := tableColumnSet(db, "ade_tasks")
	if err != nil {
		return err
	}
	for _, column := range obsoleteAdeTaskColumns {
		if !present[column] {
			continue
		}
		if err := dropColumn(db, "ade_tasks", column); err != nil {
			return fmt.Errorf("drop ade_tasks.%s: %w", column, err)
		}
	}
	return nil
}

// migrateDropObsoleteTodoColumns 按真实表结构删掉 ade_todos 上已废弃的列，避免 AutoMigrate 残留。
func migrateDropObsoleteTodoColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.Todo{}) {
		return nil
	}
	present, err := tableColumnSet(db, "ade_todos")
	if err != nil {
		return err
	}
	for _, column := range obsoleteAdeTodoColumns {
		if !present[column] {
			continue
		}
		if err := dropColumn(db, "ade_todos", column); err != nil {
			return fmt.Errorf("drop ade_todos.%s: %w", column, err)
		}
	}
	return nil
}

// tableColumnSet 读表上现有列名（小写），不依赖当前模型字段。
func tableColumnSet(db *gorm.DB, table string) (map[string]bool, error) {
	cols, err := db.Migrator().ColumnTypes(table)
	if err != nil {
		return nil, fmt.Errorf("list columns on %s: %w", table, err)
	}
	out := make(map[string]bool, len(cols))
	for _, col := range cols {
		out[strings.ToLower(col.Name())] = true
	}
	return out, nil
}
