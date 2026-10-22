package database

import (
	"testing"

	"chengzi-agent-code-server/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigrateDropObsoleteTaskColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:ade_task_columns?mode=memory&cache=private"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE ade_tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL DEFAULT 0,
  project_id INTEGER NOT NULL DEFAULT 0,
  name TEXT NOT NULL DEFAULT '',
  overview TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  raw_request TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending',
  mode TEXT NOT NULL DEFAULT 'plan',
  branch TEXT NOT NULL DEFAULT '',
  worktree TEXT NOT NULL DEFAULT '',
  merged_dev INTEGER NOT NULL DEFAULT 0,
  merged_main INTEGER NOT NULL DEFAULT 0,
  retries INTEGER NOT NULL DEFAULT 0,
  coder_agent_id TEXT NOT NULL DEFAULT '',
  verifier_agent_id TEXT NOT NULL DEFAULT '',
  reviewer_agent_id TEXT NOT NULL DEFAULT '',
  planned INTEGER NOT NULL DEFAULT 0,
  error_kind TEXT NOT NULL DEFAULT '',
  error_agent_id TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  error_at DATETIME,
  client_id TEXT NOT NULL DEFAULT '',
  created_at DATETIME,
  updated_at DATETIME
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
INSERT INTO ade_tasks (
  user_id, project_id, name, coder_agent_id, verifier_agent_id, reviewer_agent_id,
  planned, error_kind, error_agent_id, error_message, client_id, created_at, updated_at
) VALUES (
  1, 1, 'demo', 'ade-coder', 'ade-verifier', 'ade-reviewer',
  1, 'coder', 'ade-coder', 'boom', 'desk', datetime('now'), datetime('now')
)`).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(&model.Task{}); err != nil {
		t.Fatal(err)
	}
	if err := migrateDropObsoleteTaskColumns(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateDropObsoleteTaskColumns(db); err != nil {
		t.Fatal(err)
	}

	present, err := tableColumnSet(db, "ade_tasks")
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range obsoleteAdeTaskColumns {
		if present[column] {
			t.Fatalf("column still present: %s", column)
		}
	}

	var name string
	if err := db.Raw(`SELECT name FROM ade_tasks WHERE id = 1`).Scan(&name).Error; err != nil {
		t.Fatal(err)
	}
	if name != "demo" {
		t.Fatalf("row lost: %q", name)
	}
}

func TestMigrateDropObsoleteTodoColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:ade_todo_columns?mode=memory&cache=private"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE ade_todos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id INTEGER NOT NULL DEFAULT 0,
  seq INTEGER NOT NULL DEFAULT 0,
  title TEXT NOT NULL DEFAULT '',
  detail TEXT NOT NULL DEFAULT '',
  acceptance TEXT NOT NULL DEFAULT '',
  depends_on TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'todo',
  source TEXT NOT NULL DEFAULT 'agent',
  parent_id INTEGER NOT NULL DEFAULT 0,
  retry INTEGER NOT NULL DEFAULT 0,
  conversation_id INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME,
  updated_at DATETIME
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_ade_todos_parent_id ON ade_todos(parent_id)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
INSERT INTO ade_todos (task_id, seq, title, acceptance, depends_on, parent_id, retry, conversation_id, created_at, updated_at)
VALUES (1, 1, '加接口', '接口返回列表', '第 1 条', 9, 2, 42, datetime('now'), datetime('now'))
`).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(&model.Todo{}); err != nil {
		t.Fatal(err)
	}
	if err := migrateDropObsoleteTodoColumns(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateDropObsoleteTodoColumns(db); err != nil {
		t.Fatal(err)
	}

	present, err := tableColumnSet(db, "ade_todos")
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range obsoleteAdeTodoColumns {
		if present[column] {
			t.Fatalf("column still present: %s", column)
		}
	}

	var title string
	if err := db.Raw(`SELECT title FROM ade_todos WHERE id = 1`).Scan(&title).Error; err != nil {
		t.Fatal(err)
	}
	if title != "加接口" {
		t.Fatalf("row lost: %q", title)
	}
}
