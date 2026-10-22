package database

import (
	"testing"

	"chengzi-agent-code-server/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestEnsureBuiltinTools(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:ensure_builtin?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Tool{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Tool{
		Name: "Task", DisplayName: "旧名称", Type: model.ToolTypeBuiltin, Parameters: "{}", Extra: "{}",
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := ensureBuiltinTools(db); err != nil {
		t.Fatal(err)
	}

	var rows []model.Tool
	if err := db.Where("type = ?", model.ToolTypeBuiltin).Order("name asc").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Name != "AskQuestion" || rows[1].Name != "Task" || rows[2].Name != "todowriter" {
		t.Fatalf("builtins=%v", rows)
	}
	if rows[1].DisplayName != "旧名称" {
		t.Fatalf("existing Task was overwritten: %s", rows[1].DisplayName)
	}
}
