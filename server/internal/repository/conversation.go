package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
)

// ConversationRepository 会话数据访问。
type ConversationRepository interface {
	Create(ctx context.Context, c *model.Conversation) error
	GetByID(ctx context.Context, id uint64) (*model.Conversation, error)
	List(ctx context.Context, offset, limit int) ([]model.Conversation, int64, error)
	ListByUser(ctx context.Context, userID uint64, clientID string, projectID uint64, channel string, taskID uint64, offset, limit int) ([]model.Conversation, int64, error)
	Delete(ctx context.Context, id uint64) error
	UpdateTitle(ctx context.Context, id uint64, title string) error
	UpdateModelID(ctx context.Context, id, modelID uint64) error
	UpdateWorkspace(ctx context.Context, id uint64, workspace string) error
	UpdateContextUsage(ctx context.Context, id uint64, usageJSON string) error
	DeleteMessagesFromID(ctx context.Context, conversationID, fromMessageID uint64) error
	AddMessage(ctx context.Context, m *model.Message) error
	UpdateMessageContent(ctx context.Context, id uint64, content string) error
	UpdateMessageBody(ctx context.Context, id uint64, content, metadata string) error
	ListMessages(ctx context.Context, conversationID uint64) ([]model.Message, error)
}

type conversationRepository struct {
	db *gorm.DB
}

// NewConversationRepository 构造会话仓储。
func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepository{db: db}
}

func (r *conversationRepository) Create(ctx context.Context, c *model.Conversation) error {
	link := c.Link
	c.Link = nil
	if link != nil && link.ProjectID == 0 && strings.TrimSpace(link.Channel) == "" && link.TaskID == 0 {
		link = nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(c).Error; err != nil {
			return fmt.Errorf("create conversation: %w", err)
		}
		if link == nil {
			return nil
		}
		link.ConversationID = c.ID
		if err := tx.Create(link).Error; err != nil {
			return fmt.Errorf("create conversation link: %w", err)
		}
		c.Link = link
		return nil
	})
}

func (r *conversationRepository) GetByID(ctx context.Context, id uint64) (*model.Conversation, error) {
	var c model.Conversation
	if err := r.db.WithContext(ctx).
		Preload("Link").
		Preload("Model").
		Preload("Model.Provider").
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("id asc")
		}).
		First(&c, id).Error; err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	return &c, nil
}

func (r *conversationRepository) List(ctx context.Context, offset, limit int) ([]model.Conversation, int64, error) {
	var (
		list  []model.Conversation
		total int64
	)
	q := r.db.WithContext(ctx).Model(&model.Conversation{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count conversations: %w", err)
	}
	if err := q.Preload("Link").Preload("Model.Provider").Offset(offset).Limit(limit).Order("id desc").Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list conversations: %w", err)
	}
	return list, total, nil
}

func (r *conversationRepository) ListByUser(ctx context.Context, userID uint64, clientID string, projectID uint64, channel string, taskID uint64, offset, limit int) ([]model.Conversation, int64, error) {
	var (
		list  []model.Conversation
		total int64
	)
	q := r.db.WithContext(ctx).Model(&model.Conversation{}).Where("user_id = ?", userID)
	if clientID != "" {
		q = q.Where("client_id = ?", clientID)
	}
	if projectID > 0 || channel != "" || taskID > 0 {
		linkQ := r.db.WithContext(ctx).Model(&model.ConversationLink{}).Select("conversation_id")
		if projectID > 0 {
			linkQ = linkQ.Where("project_id = ?", projectID)
		}
		if channel != "" {
			linkQ = linkQ.Where("channel = ?", channel)
		}
		if taskID > 0 {
			linkQ = linkQ.Where("task_id = ?", taskID)
		} else {
			linkQ = linkQ.Where("task_id = ?", 0)
		}
		q = q.Where("id IN (?)", linkQ)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count conversations by user: %w", err)
	}
	if err := q.Preload("Link").Preload("Model.Provider").Offset(offset).Limit(limit).Order("id desc").Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list conversations by user: %w", err)
	}
	return list, total, nil
}

func (r *conversationRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("conversation_id = ?", id).Delete(&model.Message{}).Error; err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}
		if err := tx.Where("conversation_id = ?", id).Delete(&model.ConversationLink{}).Error; err != nil {
			return fmt.Errorf("delete conversation link: %w", err)
		}
		if err := tx.Delete(&model.Conversation{}, id).Error; err != nil {
			return fmt.Errorf("delete conversation: %w", err)
		}
		return nil
	})
}

func (r *conversationRepository) UpdateTitle(ctx context.Context, id uint64, title string) error {
	if err := r.db.WithContext(ctx).Model(&model.Conversation{}).Where("id = ?", id).Update("title", title).Error; err != nil {
		return fmt.Errorf("update conversation title: %w", err)
	}
	return nil
}

func (r *conversationRepository) UpdateModelID(ctx context.Context, id, modelID uint64) error {
	if err := r.db.WithContext(ctx).Model(&model.Conversation{}).Where("id = ?", id).Update("model_id", modelID).Error; err != nil {
		return fmt.Errorf("update conversation model: %w", err)
	}
	return nil
}

func (r *conversationRepository) UpdateWorkspace(ctx context.Context, id uint64, workspace string) error {
	if err := r.db.WithContext(ctx).Model(&model.Conversation{}).Where("id = ?", id).Update("workspace", workspace).Error; err != nil {
		return fmt.Errorf("update conversation workspace: %w", err)
	}
	return nil
}

func (r *conversationRepository) UpdateContextUsage(ctx context.Context, id uint64, usageJSON string) error {
	if err := r.db.WithContext(ctx).Model(&model.Conversation{}).Where("id = ?", id).Update("context_usage", usageJSON).Error; err != nil {
		return fmt.Errorf("update conversation context usage: %w", err)
	}
	return nil
}

func (r *conversationRepository) DeleteMessagesFromID(ctx context.Context, conversationID, fromMessageID uint64) error {
	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND id >= ?", conversationID, fromMessageID).
		Delete(&model.Message{}).Error
}

func (r *conversationRepository) AddMessage(ctx context.Context, m *model.Message) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	return nil
}

func (r *conversationRepository) UpdateMessageContent(ctx context.Context, id uint64, content string) error {
	if err := r.db.WithContext(ctx).Model(&model.Message{}).Where("id = ?", id).Update("content", content).Error; err != nil {
		return fmt.Errorf("update message content: %w", err)
	}
	return nil
}

func (r *conversationRepository) UpdateMessageBody(ctx context.Context, id uint64, content, metadata string) error {
	err := r.db.WithContext(ctx).Model(&model.Message{}).Where("id = ?", id).Updates(map[string]any{
		"content":  content,
		"metadata": metadata,
	}).Error
	if err != nil {
		return fmt.Errorf("update message body: %w", err)
	}
	return nil
}

func (r *conversationRepository) ListMessages(ctx context.Context, conversationID uint64) ([]model.Message, error) {
	var list []model.Message
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("id asc").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	return list, nil
}
