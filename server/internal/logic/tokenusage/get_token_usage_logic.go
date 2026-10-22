package tokenusage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"chengzi-agent-code-server/pkg/llmchat"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTokenUsageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTokenUsageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTokenUsageLogic {
	return &GetTokenUsageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTokenUsageLogic) GetTokenUsage(req *types.TokenUsageStatsReq) (*types.TokenUsageStatsResp, error) {
	userID := middleware.UserID(l.ctx)
	if userID == 0 {
		return nil, fmt.Errorf("未登录")
	}
	from, to, grain := resolveUsageWindow(req.Range, time.Now())
	sum, err := l.svcCtx.TokenUsageRepo.Summarize(l.ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	rows, err := l.svcCtx.TokenUsageRepo.Trend(l.ctx, userID, from, to, grain)
	if err != nil {
		return nil, err
	}
	usage := llmchat.TokenUsage{
		PromptTokens:        sum.PromptTokens,
		CompletionTokens:    sum.CompletionTokens,
		CacheCreationTokens: sum.CacheCreationTokens,
		CacheReadTokens:     sum.CacheReadTokens,
	}
	return &types.TokenUsageStatsResp{
		PromptTokens:        sum.PromptTokens,
		CompletionTokens:    sum.CompletionTokens,
		CacheCreationTokens: sum.CacheCreationTokens,
		CacheReadTokens:     sum.CacheReadTokens,
		RealTokens:          usage.RealTokens(),
		RequestCount:        sum.RequestCount,
		Cost:                sum.Cost,
		CacheHitRate:        usage.CacheHitRate(),
		Range:               normalizeUsageRange(req.Range),
		Trend:               fillUsageTrend(from, to, grain, rows),
	}, nil
}

func normalizeUsageRange(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "7d", "7day", "week":
		return "7d"
	case "30d", "30day", "month":
		return "30d"
	default:
		return "today"
	}
}

func resolveUsageWindow(rangeKey string, now time.Time) (from, to time.Time, grain string) {
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	switch normalizeUsageRange(rangeKey) {
	case "7d":
		return today.AddDate(0, 0, -6), today.AddDate(0, 0, 1), "day"
	case "30d":
		return today.AddDate(0, 0, -29), today.AddDate(0, 0, 1), "day"
	default:
		return today, today.AddDate(0, 0, 1), "hour"
	}
}

func fillUsageTrend(from, to time.Time, grain string, rows []repository.TokenUsageBucket) []types.TokenUsageTrendPoint {
	byBucket := make(map[string]repository.TokenUsageBucket, len(rows))
	for _, row := range rows {
		byBucket[row.Bucket] = row
	}
	step := 24 * time.Hour
	layout := "2006-01-02"
	if grain == "hour" {
		step = time.Hour
		layout = "2006-01-02 15:00:00"
	}
	out := make([]types.TokenUsageTrendPoint, 0, 32)
	for t := from; t.Before(to); t = t.Add(step) {
		key := t.Format(layout)
		row := byBucket[key]
		out = append(out, types.TokenUsageTrendPoint{
			At:                  key,
			PromptTokens:        row.PromptTokens,
			CompletionTokens:    row.CompletionTokens,
			CacheCreationTokens: row.CacheCreationTokens,
			CacheReadTokens:     row.CacheReadTokens,
			Cost:                row.Cost,
		})
	}
	return out
}
