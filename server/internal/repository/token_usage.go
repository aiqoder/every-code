package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
)

// TokenUsageTotals 时间范围内的用量合计。
type TokenUsageTotals struct {
	PromptTokens        int64
	CompletionTokens    int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	RequestCount        int64
	Cost                float64
}

// TokenUsageBucket 趋势分桶。
type TokenUsageBucket struct {
	Bucket              string
	PromptTokens        int64
	CompletionTokens    int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	Cost                float64
}

// TokenUsageRepository 用量流水数据访问。
type TokenUsageRepository interface {
	Create(ctx context.Context, row *model.TokenUsage) error
	Summarize(ctx context.Context, userID uint64, from, to time.Time) (TokenUsageTotals, error)
	Trend(ctx context.Context, userID uint64, from, to time.Time, grain string) ([]TokenUsageBucket, error)
}

type tokenUsageRepository struct {
	db *gorm.DB
}

// NewTokenUsageRepository 构造用量仓储。
func NewTokenUsageRepository(db *gorm.DB) TokenUsageRepository {
	return &tokenUsageRepository{db: db}
}

func (r *tokenUsageRepository) Create(ctx context.Context, row *model.TokenUsage) error {
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return fmt.Errorf("create token usage: %w", err)
	}
	return nil
}

func (r *tokenUsageRepository) scoped(ctx context.Context, userID uint64) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&model.TokenUsage{})
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	return q
}

func (r *tokenUsageRepository) Summarize(ctx context.Context, userID uint64, from, to time.Time) (TokenUsageTotals, error) {
	var out TokenUsageTotals
	err := r.scoped(ctx, userID).
		Select(
			"COALESCE(SUM(prompt_tokens),0) AS prompt_tokens",
			"COALESCE(SUM(completion_tokens),0) AS completion_tokens",
			"COALESCE(SUM(cache_creation_tokens),0) AS cache_creation_tokens",
			"COALESCE(SUM(cache_read_tokens),0) AS cache_read_tokens",
			"COUNT(*) AS request_count",
			"COALESCE(SUM(cost),0) AS cost",
		).
		Where("created_at >= ? AND created_at < ?", from, to).
		Scan(&out).Error
	if err != nil {
		return TokenUsageTotals{}, fmt.Errorf("summarize token usage: %w", err)
	}
	return out, nil
}

func (r *tokenUsageRepository) Trend(ctx context.Context, userID uint64, from, to time.Time, grain string) ([]TokenUsageBucket, error) {
	expr := r.bucketExpr(grain)
	var rows []TokenUsageBucket
	err := r.scoped(ctx, userID).
		Select(
			expr+" AS bucket",
			"COALESCE(SUM(prompt_tokens),0) AS prompt_tokens",
			"COALESCE(SUM(completion_tokens),0) AS completion_tokens",
			"COALESCE(SUM(cache_creation_tokens),0) AS cache_creation_tokens",
			"COALESCE(SUM(cache_read_tokens),0) AS cache_read_tokens",
			"COALESCE(SUM(cost),0) AS cost",
		).
		Where("created_at >= ? AND created_at < ?", from, to).
		Group("bucket").
		Order("bucket ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("token usage trend: %w", err)
	}
	return rows, nil
}

func (r *tokenUsageRepository) bucketExpr(grain string) string {
	name := strings.ToLower(r.db.Dialector.Name())
	// SQLite 的 strftime 默认按 UTC 输出。流水带时区偏移时，不加 localtime 会和本地时间窗口对不上，趋势图整段为 0。
	if grain == "hour" {
		if name == "mysql" {
			return "DATE_FORMAT(created_at, '%Y-%m-%d %H:00:00')"
		}
		return "strftime('%Y-%m-%d %H:00:00', created_at, 'localtime')"
	}
	if name == "mysql" {
		return "DATE_FORMAT(created_at, '%Y-%m-%d')"
	}
	return "strftime('%Y-%m-%d', created_at, 'localtime')"
}
