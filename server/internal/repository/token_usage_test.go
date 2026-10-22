package repository

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/pkg/timex"
)

func setupTokenUsageDB(t *testing.T) TokenUsageRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:token-usage?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.TokenUsage{}); err != nil {
		t.Fatal(err)
	}
	return NewTokenUsageRepository(db)
}

func TestTokenUsageCreateVisibleInTodayWindow(t *testing.T) {
	repo := setupTokenUsageDB(t)
	ctx := context.Background()
	row := &model.TokenUsage{UserID: 42, PromptTokens: 11, CompletionTokens: 4, CacheReadTokens: 6}
	if err := repo.Create(ctx, row); err != nil {
		t.Fatal(err)
	}
	if row.CreatedAt.IsZero() {
		t.Fatal("created_at was not filled")
	}

	now := time.Now()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	to := from.Add(24 * time.Hour)
	sum, err := repo.Summarize(ctx, 42, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if sum.PromptTokens != 11 || sum.CompletionTokens != 4 || sum.CacheReadTokens != 6 || sum.RequestCount != 1 {
		t.Fatalf("today sum %#v", sum)
	}

	trend, err := repo.Trend(ctx, 42, from, to, "hour")
	if err != nil {
		t.Fatal(err)
	}
	if len(trend) != 1 || trend[0].PromptTokens != 11 {
		t.Fatalf("trend %#v", trend)
	}
	hourKey := row.CreatedAt.Time().Format("2006-01-02 15:00:00")
	if trend[0].Bucket != hourKey {
		t.Fatalf("bucket %q want %q", trend[0].Bucket, hourKey)
	}
}

func TestTokenUsageSummarizeAndTrend(t *testing.T) {
	repo := setupTokenUsageDB(t)
	ctx := context.Background()
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.Local)
	rows := []*model.TokenUsage{
		{PromptTokens: 100, CompletionTokens: 10, CacheReadTokens: 50, Cost: 0.01, CreatedAt: timex.LocalTime(day.Add(9 * time.Hour))},
		{PromptTokens: 40, CompletionTokens: 5, CacheReadTokens: 10, Cost: 0.02, CreatedAt: timex.LocalTime(day.Add(9 * time.Hour))},
		{PromptTokens: 7, CompletionTokens: 1, CreatedAt: timex.LocalTime(day.Add(15 * time.Hour))},
	}
	for _, row := range rows {
		if err := repo.Create(ctx, row); err != nil {
			t.Fatal(err)
		}
	}

	sum, err := repo.Summarize(ctx, 0, day, day.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if sum.PromptTokens != 147 || sum.CompletionTokens != 16 || sum.CacheReadTokens != 60 || sum.RequestCount != 3 {
		t.Fatalf("sum %#v", sum)
	}

	trend, err := repo.Trend(ctx, 0, day, day.Add(24*time.Hour), "hour")
	if err != nil {
		t.Fatal(err)
	}
	if len(trend) != 2 {
		t.Fatalf("trend len=%d %#v", len(trend), trend)
	}
	if trend[0].PromptTokens != 140 || trend[1].PromptTokens != 7 {
		t.Fatalf("trend %#v", trend)
	}

	mine := &model.TokenUsage{
		UserID: 9, PromptTokens: 3, CreatedAt: timex.LocalTime(day.Add(10 * time.Hour)),
	}
	if err := repo.Create(ctx, mine); err != nil {
		t.Fatal(err)
	}
	onlyMine, err := repo.Summarize(ctx, 9, day, day.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if onlyMine.PromptTokens != 3 || onlyMine.RequestCount != 1 {
		t.Fatalf("user filter %#v", onlyMine)
	}
}
