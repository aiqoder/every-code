package admin

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
)

func ensurePlatformProvider(ctx context.Context, repo repository.LlmProviderRepository, id uint64) (*model.LlmProvider, error) {
	p, err := repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("供应商不存在")
		}
		return nil, err
	}
	if p.UserID != nil {
		return nil, fmt.Errorf("供应商不存在")
	}
	return p, nil
}
