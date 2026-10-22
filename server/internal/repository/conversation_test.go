package repository

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"chengzi-agent-code-server/internal/model"
)

func setupConversationDB(t *testing.T) (*gorm.DB, ConversationRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=private"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Conversation{}, &model.ConversationLink{}, &model.Message{}); err != nil {
		t.Fatal(err)
	}
	return db, NewConversationRepository(db)
}

func TestListByUserIsolatesClient(t *testing.T) {
	_, repo := setupConversationDB(t)
	ctx := context.Background()
	rows := []*model.Conversation{
		{UserID: 1, Title: "本机", ClientID: "desk", ModelID: 1},
		{UserID: 1, Title: "另一台", ClientID: "phone", ModelID: 1},
		{UserID: 2, Title: "别人的本机", ClientID: "desk", ModelID: 1},
	}
	for _, row := range rows {
		if err := repo.Create(ctx, row); err != nil {
			t.Fatal(err)
		}
	}

	list, total, err := repo.ListByUser(ctx, 1, "desk", 0, "", 0, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Title != "本机" {
		t.Fatalf("client filter: total=%d list=%+v", total, list)
	}

	all, allTotal, err := repo.ListByUser(ctx, 1, "", 0, "", 0, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if allTotal != 2 || len(all) != 2 {
		t.Fatalf("user filter without client: total=%d list=%+v", allTotal, all)
	}
}

func TestListByUserIsolatesProjectChannel(t *testing.T) {
	db, repo := setupConversationDB(t)
	ctx := context.Background()
	rows := []*model.Conversation{
		{UserID: 1, Title: "创建", ClientID: "desk", ModelID: 1, Link: &model.ConversationLink{ProjectID: 7, Channel: "create"}},
		{UserID: 1, Title: "执行", ClientID: "desk", ModelID: 1, Link: &model.ConversationLink{ProjectID: 7, Channel: "execute"}},
		{UserID: 1, Title: "任务", ClientID: "desk", ModelID: 1, Link: &model.ConversationLink{ProjectID: 7, Channel: "create", TaskID: 9}},
		{UserID: 1, Title: "流水线", ClientID: "desk", ModelID: 1},
	}
	for _, row := range rows {
		if err := repo.Create(ctx, row); err != nil {
			t.Fatal(err)
		}
	}
	list, total, err := repo.ListByUser(ctx, 1, "desk", 7, "create", 0, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Title != "创建" || list[0].Link == nil || list[0].Link.Channel != "create" {
		t.Fatalf("channel filter: total=%d list=%+v", total, list)
	}
	owned, ownedTotal, err := repo.ListByUser(ctx, 1, "desk", 7, "create", 9, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if ownedTotal != 1 || len(owned) != 1 || owned[0].Title != "任务" || owned[0].Link == nil || owned[0].Link.TaskID != 9 {
		t.Fatalf("task filter: total=%d list=%+v", ownedTotal, owned)
	}

	if err := repo.Delete(ctx, list[0].ID); err != nil {
		t.Fatal(err)
	}
	again, againTotal, err := repo.ListByUser(ctx, 1, "desk", 7, "create", 0, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if againTotal != 0 || len(again) != 0 {
		t.Fatalf("link survived delete: total=%d list=%+v", againTotal, again)
	}
	var left int64
	if err := db.Model(&model.ConversationLink{}).Where("conversation_id = ?", list[0].ID).Count(&left).Error; err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("conversation link rows left: %d", left)
	}
}
