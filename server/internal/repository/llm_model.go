package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
)

// LlmModelRepository 大模型数据访问接口。
type LlmModelRepository interface {
	Create(ctx context.Context, m *model.LlmModel) error
	Update(ctx context.Context, m *model.LlmModel) error
	Delete(ctx context.Context, id uint64) error
	DeleteByProviderID(ctx context.Context, providerID uint64) error
	GetByID(ctx context.Context, id uint64) (*model.LlmModel, error)
	List(ctx context.Context, providerID uint64, offset, limit int) ([]model.LlmModel, int64, error)
	ListForUser(ctx context.Context, userID uint64, providerID uint64, offset, limit int) ([]model.LlmModel, int64, error)
}

type llmModelRepository struct {
	db *gorm.DB
}

// NewLlmModelRepository 构造大模型仓储。
func NewLlmModelRepository(db *gorm.DB) LlmModelRepository {
	return &llmModelRepository{db: db}
}

func (r *llmModelRepository) Create(ctx context.Context, m *model.LlmModel) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("create llm model: %w", err)
	}
	return nil
}

func (r *llmModelRepository) Update(ctx context.Context, m *model.LlmModel) error {
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("update llm model: %w", err)
	}
	return nil
}

func (r *llmModelRepository) GetByID(ctx context.Context, id uint64) (*model.LlmModel, error) {
	var m model.LlmModel
	if err := r.db.WithContext(ctx).Preload("Provider").First(&m, id).Error; err != nil {
		return nil, fmt.Errorf("get llm model by id: %w", err)
	}
	return &m, nil
}

func (r *llmModelRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&model.LlmModel{}, id).Error; err != nil {
		return fmt.Errorf("delete llm model: %w", err)
	}
	return nil
}

func (r *llmModelRepository) DeleteByProviderID(ctx context.Context, providerID uint64) error {
	if err := r.db.WithContext(ctx).Where("provider_id = ?", providerID).Delete(&model.LlmModel{}).Error; err != nil {
		return fmt.Errorf("delete llm models by provider: %w", err)
	}
	return nil
}

func (r *llmModelRepository) List(ctx context.Context, providerID uint64, offset, limit int) ([]model.LlmModel, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.LlmModel{}).
		Joins("JOIN llm_providers ON llm_providers.id = llm_models.provider_id").
		Where("llm_providers.user_id IS NULL")
	if providerID > 0 {
		q = q.Where("llm_models.provider_id = ?", providerID)
	}
	return r.finishList(q, offset, limit)
}

func (r *llmModelRepository) ListForUser(ctx context.Context, userID uint64, providerID uint64, offset, limit int) ([]model.LlmModel, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.LlmModel{}).
		Joins("JOIN llm_providers ON llm_providers.id = llm_models.provider_id").
		Where("llm_providers.user_id IS NULL OR llm_providers.user_id = ?", userID)
	if providerID > 0 {
		q = q.Where("llm_models.provider_id = ?", providerID)
	}
	return r.finishList(q, offset, limit)
}

func (r *llmModelRepository) finishList(q *gorm.DB, offset, limit int) ([]model.LlmModel, int64, error) {
	var (
		list  []model.LlmModel
		total int64
	)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count llm models: %w", err)
	}
	if err := q.Preload("Provider").Offset(offset).Limit(limit).Order("llm_models.id desc").Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list llm models: %w", err)
	}
	return list, total, nil
}
