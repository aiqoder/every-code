package database

import (
	"testing"

	"chengzi-agent-code-server/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigrateConversationLinks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:conversation_links?mode=memory&cache=private"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE conversations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL DEFAULT 0,
  title TEXT NOT NULL DEFAULT '',
  model_id INTEGER NOT NULL DEFAULT 0,
  client_id TEXT NOT NULL DEFAULT '',
  project_id INTEGER NOT NULL DEFAULT 0,
  channel TEXT NOT NULL DEFAULT '',
  workspace TEXT NOT NULL DEFAULT '',
  context_usage TEXT NOT NULL DEFAULT '',
  created_at DATETIME,
  updated_at DATETIME
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_conversations_project_id ON conversations(project_id)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_conversations_channel ON conversations(channel)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
INSERT INTO conversations (user_id, title, model_id, client_id, project_id, channel, workspace, context_usage, created_at, updated_at)
VALUES
  (1, '创建', 1, 'desk', 7, 'create', '/work', '', datetime('now'), datetime('now')),
  (1, '流水线', 1, 'desk', 0, '', '', '', datetime('now'), datetime('now'))
`).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(&model.Conversation{}, &model.ConversationLink{}); err != nil {
		t.Fatal(err)
	}
	if err := migrateConversationLinks(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateConversationLinks(db); err != nil {
		t.Fatal(err)
	}

	legacy := &conversationLegacyColumns{}
	if db.Migrator().HasColumn(legacy, "ProjectID") || db.Migrator().HasColumn(legacy, "Channel") {
		t.Fatal("project_id or channel still on conversations")
	}

	var links []model.ConversationLink
	if err := db.Order("conversation_id asc").Find(&links).Error; err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].ConversationID != 1 || links[0].ProjectID != 7 || links[0].Channel != "create" {
		t.Fatalf("links=%+v", links)
	}

	var titles []string
	if err := db.Model(&model.Conversation{}).Order("id asc").Pluck("title", &titles).Error; err != nil {
		t.Fatal(err)
	}
	if len(titles) != 2 || titles[0] != "创建" || titles[1] != "流水线" {
		t.Fatalf("titles=%v", titles)
	}
}
