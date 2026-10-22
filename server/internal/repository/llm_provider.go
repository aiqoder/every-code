package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
)

// LlmProviderRepository 大模型供应商数据访问接口。
type LlmProviderRepository interface {
	Create(ctx context.Context, provider *model.LlmProvider) error
	Update(ctx context.Context, provider *model.LlmProvider) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (*model.LlmProvider, error)
	List(ctx context.Context, offset, limit int) ([]model.LlmProvider, int64, error)
	ListAll(ctx context.Context) ([]model.LlmProvider, error)
	ListForUser(ctx context.Context, userID uint64, offset, limit int) ([]model.LlmProvider, int64, error)
	ListAllForUser(ctx context.Context, userID uint64) ([]model.LlmProvider, error)
}

type llmProviderRepository struct {
	db *gorm.DB
}

// NewLlmProviderRepository 构造供应商仓储。
func NewLlmProviderRepository(db *gorm.DB) LlmProviderRepository {
	return &llmProviderRepository{db: db}
}

func (r *llmProviderRepository) Create(ctx context.Context, provider *model.LlmProvider) error {
	if err := r.db.WithContext(ctx).Create(provider).Error; err != nil {
		return fmt.Errorf("create llm provider: %w", err)
	}
	return nil
}

func (r *llmProviderRepository) Update(ctx context.Context, provider *model.LlmProvider) error {
	if err := r.db.WithContext(ctx).Save(provider).Error; err != nil {
		return fmt.Errorf("update llm provider: %w", err)
	}
	return nil
}

func (r *llmProviderRepository) GetByID(ctx context.Context, id uint64) (*model.LlmProvider, error) {
	var provider model.LlmProvider
	if err := r.db.WithContext(ctx).First(&provider, id).Error; err != nil {
		return nil, fmt.Errorf("get llm provider by id: %w", err)
	}
	return &provider, nil
}

func (r *llmProviderRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&model.LlmProvider{}, id).Error; err != nil {
		return fmt.Errorf("delete llm provider: %w", err)
	}
	return nil
}

func (r *llmProviderRepository) List(ctx context.Context, offset, limit int) ([]model.LlmProvider, int64, error) {
	return r.listWhere(ctx, "user_id IS NULL", offset, limit)
}

func (r *llmProviderRepository) ListAll(ctx context.Context) ([]model.LlmProvider, error) {
	return r.listAllWhere(ctx, "user_id IS NULL")
}

func (r *llmProviderRepository) ListForUser(ctx context.Context, userID uint64, offset, limit int) ([]model.LlmProvider, int64, error) {
	return r.listWhere(ctx, "user_id IS NULL OR user_id = ?", offset, limit, userID)
}

func (r *llmProviderRepository) ListAllForUser(ctx context.Context, userID uint64) ([]model.LlmProvider, error) {
	return r.listAllWhere(ctx, "user_id IS NULL OR user_id = ?", userID)
}

func (r *llmProviderRepository) listWhere(ctx context.Context, cond string, offset, limit int, args ...any) ([]model.LlmProvider, int64, error) {
	var (
		list  []model.LlmProvider
		total int64
	)
	q := r.db.WithContext(ctx).Model(&model.LlmProvider{}).Where(cond, args...)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count llm providers: %w", err)
	}
	if err := q.Offset(offset).Limit(limit).Order("id desc").Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list llm providers: %w", err)
	}
	return list, total, nil
}

func (r *llmProviderRepository) listAllWhere(ctx context.Context, cond string, args ...any) ([]model.LlmProvider, error) {
	var list []model.LlmProvider
	if err := r.db.WithContext(ctx).Where(cond, args...).Order("id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list llm providers: %w", err)
	}
	return list, nil
}
