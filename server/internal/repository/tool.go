package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
)

// ToolRepository 工具数据访问接口。
type ToolRepository interface {
	Create(ctx context.Context, t *model.Tool) error
	Update(ctx context.Context, t *model.Tool) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (*model.Tool, error)
	ListAdmin(ctx context.Context, offset, limit int) ([]model.Tool, int64, error)
	ExistsByName(ctx context.Context, name string, userID *uint64, excludeID uint64) (bool, error)
	ListAdminForChat(ctx context.Context) ([]model.Tool, error)
}

type toolRepository struct {
	db *gorm.DB
}

// NewToolRepository 构造工具仓储。
func NewToolRepository(db *gorm.DB) ToolRepository {
	return &toolRepository{db: db}
}

func (r *toolRepository) Create(ctx context.Context, t *model.Tool) error {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		return fmt.Errorf("create tool: %w", err)
	}
	return nil
}

func (r *toolRepository) Update(ctx context.Context, t *model.Tool) error {
	if err := r.db.WithContext(ctx).Save(t).Error; err != nil {
		return fmt.Errorf("update tool: %w", err)
	}
	return nil
}

func (r *toolRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&model.Tool{}, id).Error; err != nil {
		return fmt.Errorf("delete tool: %w", err)
	}
	return nil
}

func (r *toolRepository) GetByID(ctx context.Context, id uint64) (*model.Tool, error) {
	var t model.Tool
	if err := r.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, fmt.Errorf("get tool by id: %w", err)
	}
	return &t, nil
}

// ListAdmin 列出管理员工具（排除已废弃的 client 镜像）。
func (r *toolRepository) ListAdmin(ctx context.Context, offset, limit int) ([]model.Tool, int64, error) {
	return r.listWhere(ctx, "user_id IS NULL AND type <> ?", offset, limit, model.ToolTypeClient)
}

// ListAdminForChat 列出管理员工具全量，供 agent loop 加载（工具不再与用户/智能体绑定；不含 client）。
func (r *toolRepository) ListAdminForChat(ctx context.Context) ([]model.Tool, error) {
	var list []model.Tool
	if err := r.db.WithContext(ctx).
		Where("user_id IS NULL AND type <> ?", model.ToolTypeClient).
		Order("id asc").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list admin tools for chat: %w", err)
	}
	return list, nil
}

func (r *toolRepository) listWhere(ctx context.Context, where string, offset, limit int, args ...any) ([]model.Tool, int64, error) {
	var (
		list  []model.Tool
		total int64
	)
	q := r.db.WithContext(ctx).Model(&model.Tool{}).Where(where, args...)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tools: %w", err)
	}
	if err := q.Offset(offset).Limit(limit).Order("id desc").Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list tools: %w", err)
	}
	return list, total, nil
}

func (r *toolRepository) ExistsByName(ctx context.Context, name string, userID *uint64, excludeID uint64) (bool, error) {
	q := r.db.WithContext(ctx).Model(&model.Tool{}).Where("name = ?", name)
	if userID == nil {
		q = q.Where("user_id IS NULL")
	} else {
		q = q.Where("user_id = ?", *userID)
	}
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, fmt.Errorf("check tool name: %w", err)
	}
	return count > 0, nil
}
