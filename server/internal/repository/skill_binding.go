package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
)

// SkillBindingRepository 系统技能与智能体绑定。
type SkillBindingRepository interface {
	MapBySkillID(ctx context.Context) (map[string]string, error)
	SaveAll(ctx context.Context, bindings []model.SkillBinding) error
}

type skillBindingRepository struct {
	db *gorm.DB
}

func NewSkillBindingRepository(db *gorm.DB) SkillBindingRepository {
	return &skillBindingRepository{db: db}
}

func (r *skillBindingRepository) MapBySkillID(ctx context.Context) (map[string]string, error) {
	var rows []model.SkillBinding
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list skill bindings: %w", err)
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		id := strings.TrimSpace(row.SkillID)
		if id == "" {
			continue
		}
		out[id] = strings.TrimSpace(row.AgentID)
	}
	return out, nil
}

func (r *skillBindingRepository) SaveAll(ctx context.Context, bindings []model.SkillBinding) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range bindings {
			skillID := strings.TrimSpace(row.SkillID)
			if skillID == "" {
				continue
			}
			agentID := strings.TrimSpace(row.AgentID)
			var existing model.SkillBinding
			err := tx.Where("skill_id = ?", skillID).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				row.SkillID = skillID
				row.AgentID = agentID
				if err := tx.Create(&row).Error; err != nil {
					return fmt.Errorf("create skill binding %s: %w", skillID, err)
				}
				continue
			}
			if err != nil {
				return fmt.Errorf("get skill binding %s: %w", skillID, err)
			}
			if err := tx.Model(&existing).Updates(map[string]any{
				"agent_id": agentID,
			}).Error; err != nil {
				return fmt.Errorf("update skill binding %s: %w", skillID, err)
			}
		}
		return nil
	})
}
