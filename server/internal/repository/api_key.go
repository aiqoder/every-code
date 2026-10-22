package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/pkg/timex"
)

// ApiKeyRepository API Key 数据访问。
type ApiKeyRepository interface {
	Create(ctx context.Context, key *model.ApiKey) error
	ListByUser(ctx context.Context, userID uint64) ([]model.ApiKey, error)
	GetByID(ctx context.Context, id uint64) (*model.ApiKey, error)
	GetByHash(ctx context.Context, keyHash string) (*model.ApiKey, error)
	Delete(ctx context.Context, id, userID uint64) error
	TouchLastUsed(ctx context.Context, id uint64) error
}

type apiKeyRepository struct {
	db *gorm.DB
}

// NewApiKeyRepository 构造 API Key 仓储。
func NewApiKeyRepository(db *gorm.DB) ApiKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Create(ctx context.Context, key *model.ApiKey) error {
	if err := r.db.WithContext(ctx).Create(key).Error; err != nil {
		return fmt.Errorf("create api key: %w", err)
	}
	return nil
}

func (r *apiKeyRepository) ListByUser(ctx context.Context, userID uint64) ([]model.ApiKey, error) {
	var list []model.ApiKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	return list, nil
}

func (r *apiKeyRepository) GetByID(ctx context.Context, id uint64) (*model.ApiKey, error) {
	var key model.ApiKey
	if err := r.db.WithContext(ctx).First(&key, id).Error; err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	return &key, nil
}

func (r *apiKeyRepository) GetByHash(ctx context.Context, keyHash string) (*model.ApiKey, error) {
	var key model.ApiKey
	if err := r.db.WithContext(ctx).Where("key_hash = ?", keyHash).First(&key).Error; err != nil {
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}
	return &key, nil
}

func (r *apiKeyRepository) Delete(ctx context.Context, id, userID uint64) error {
	res := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.ApiKey{})
	if res.Error != nil {
		return fmt.Errorf("delete api key: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *apiKeyRepository) TouchLastUsed(ctx context.Context, id uint64) error {
	t := timex.LocalTime(time.Now())
	if err := r.db.WithContext(ctx).Model(&model.ApiKey{}).Where("id = ?", id).Update("last_used_at", &t).Error; err != nil {
		return fmt.Errorf("touch api key: %w", err)
	}
	return nil
}
