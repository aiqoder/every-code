package tokenusage

import (
	"context"
	"testing"
	"time"

	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
)

func TestResolveUsageWindowToday(t *testing.T) {
	now := time.Date(2026, 9, 20, 21, 10, 0, 0, time.Local)
	from, to, grain := resolveUsageWindow("today", now)
	if grain != "hour" {
		t.Fatalf("grain=%s", grain)
	}
	if from.Hour() != 0 || from.Day() != 20 {
		t.Fatalf("from=%s", from)
	}
	if to.Day() != 21 || to.Hour() != 0 {
		t.Fatalf("to=%s", to)
	}
}

func TestResolveUsageWindow7d(t *testing.T) {
	now := time.Date(2026, 9, 20, 10, 0, 0, 0, time.Local)
	from, to, grain := resolveUsageWindow("7d", now)
	if grain != "day" {
		t.Fatalf("grain=%s", grain)
	}
	if from.Day() != 14 || to.Day() != 21 {
		t.Fatalf("from=%s to=%s", from, to)
	}
}

func TestFillUsageTrendHourly(t *testing.T) {
	from := time.Date(2026, 9, 20, 0, 0, 0, 0, time.Local)
	to := from.Add(24 * time.Hour)
	rows := []repository.TokenUsageBucket{
		{Bucket: "2026-09-20 09:00:00", PromptTokens: 100, CompletionTokens: 8, Cost: 0.01},
	}
	got := fillUsageTrend(from, to, "hour", rows)
	if len(got) != 24 {
		t.Fatalf("len=%d", len(got))
	}
	if got[9].PromptTokens != 100 || got[9].CompletionTokens != 8 {
		t.Fatalf("hour 9: %#v", got[9])
	}
	if got[0].PromptTokens != 0 {
		t.Fatalf("hour 0 should be empty")
	}
}

func TestNormalizeUsageRange(t *testing.T) {
	if normalizeUsageRange("") != "today" || normalizeUsageRange("week") != "7d" {
		t.Fatal("unexpected range")
	}
}

type fakeUsageRepo struct {
	userID uint64
	sum    repository.TokenUsageTotals
	trend  []repository.TokenUsageBucket
}

func (f *fakeUsageRepo) Create(context.Context, *model.TokenUsage) error { return nil }
func (f *fakeUsageRepo) Summarize(_ context.Context, userID uint64, _ time.Time, _ time.Time) (repository.TokenUsageTotals, error) {
	f.userID = userID
	return f.sum, nil
}
func (f *fakeUsageRepo) Trend(_ context.Context, userID uint64, _ time.Time, _ time.Time, _ string) ([]repository.TokenUsageBucket, error) {
	f.userID = userID
	return f.trend, nil
}

func TestGetTokenUsageRequiresLogin(t *testing.T) {
	l := &GetTokenUsageLogic{
		ctx:    context.Background(),
		svcCtx: &svc.ServiceContext{TokenUsageRepo: &fakeUsageRepo{}},
	}
	if _, err := l.GetTokenUsage(&types.TokenUsageStatsReq{}); err == nil {
		t.Fatal("expected login error")
	}
}

func TestGetTokenUsageBuildsResp(t *testing.T) {
	repo := &fakeUsageRepo{
		sum: repository.TokenUsageTotals{
			PromptTokens: 5310, CompletionTokens: 218, CacheReadTokens: 19845, RequestCount: 3, Cost: 1.25,
		},
	}
	l := &GetTokenUsageLogic{
		ctx: middleware.ContextWithUser(context.Background(), 42),
		svcCtx: &svc.ServiceContext{
			TokenUsageRepo: repo,
		},
	}
	resp, err := l.GetTokenUsage(&types.TokenUsageStatsReq{Range: "today"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.userID != 42 {
		t.Fatalf("userID=%d", repo.userID)
	}
	if resp.RealTokens != 5310+218+19845 || resp.RequestCount != 3 {
		t.Fatalf("resp %#v", resp)
	}
	trend, ok := resp.Trend.([]types.TokenUsageTrendPoint)
	if !ok || len(trend) != 24 {
		t.Fatalf("trend %T len=%v", resp.Trend, len(trend))
	}
}
