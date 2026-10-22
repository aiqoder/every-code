package ade

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/pkg/timex"
)

func normalizeSceneText(title, description string) (string, string, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" {
		return "", "", errors.New("请填写场景标题")
	}
	if utf8.RuneCountInString(title) > 64 {
		return "", "", errors.New("场景标题不能超过 64 个字")
	}
	if description == "" {
		return "", "", errors.New("请填写场景描述")
	}
	if utf8.RuneCountInString(description) > 20000 {
		return "", "", errors.New("场景描述过长")
	}
	return title, description, nil
}

func (s *Service) sceneForUser(ctx context.Context, userID, id uint64) (*model.Scene, error) {
	sc, err := s.repo.GetScene(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "record not found") {
			return nil, errors.New("场景不存在")
		}
		return nil, err
	}
	if sc.UserID != 0 && sc.UserID != userID {
		return nil, errors.New("无权使用该场景")
	}
	return sc, nil
}

func (s *Service) ownScene(ctx context.Context, userID, id uint64) (*model.Scene, error) {
	sc, err := s.repo.GetScene(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "record not found") {
			return nil, errors.New("场景不存在")
		}
		return nil, err
	}
	if sc.UserID != userID || userID == 0 {
		return nil, errors.New("只能修改自己的场景")
	}
	return sc, nil
}

// ListScenes 公共场景加上当前用户自己的场景。
func (s *Service) ListScenes(ctx context.Context, userID uint64) ([]model.Scene, error) {
	return s.repo.ListVisibleScenes(ctx, userID)
}

// CreateUserScene 用户自定义场景。
func (s *Service) CreateUserScene(ctx context.Context, userID uint64, title, description string) (*model.Scene, error) {
	if userID == 0 {
		return nil, errors.New("请先登录")
	}
	title, description, err := normalizeSceneText(title, description)
	if err != nil {
		return nil, err
	}
	now := timex.LocalTime(time.Now())
	sc := &model.Scene{
		UserID:      userID,
		Title:       title,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.CreateScene(ctx, sc); err != nil {
		return nil, err
	}
	return sc, nil
}

// UpdateUserScene 只允许改自己的场景。
func (s *Service) UpdateUserScene(ctx context.Context, userID, id uint64, title, description string) (*model.Scene, error) {
	sc, err := s.ownScene(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	title, description, err = normalizeSceneText(title, description)
	if err != nil {
		return nil, err
	}
	sc.Title = title
	sc.Description = description
	sc.UpdatedAt = timex.LocalTime(time.Now())
	if err := s.repo.UpdateScene(ctx, sc); err != nil {
		return nil, err
	}
	return sc, nil
}

// DeleteUserScene 删除自己的场景，已创建的项目保留当时复制的描述。
func (s *Service) DeleteUserScene(ctx context.Context, userID, id uint64) error {
	if _, err := s.ownScene(ctx, userID, id); err != nil {
		return err
	}
	return s.repo.DeleteScene(ctx, id)
}

// ListAdminScenes 管理后台只看公共场景。
func (s *Service) ListAdminScenes(ctx context.Context) ([]model.Scene, error) {
	return s.repo.ListAdminScenes(ctx)
}

// CreateAdminScene 管理员创建公共场景。
func (s *Service) CreateAdminScene(ctx context.Context, title, description string) (*model.Scene, error) {
	title, description, err := normalizeSceneText(title, description)
	if err != nil {
		return nil, err
	}
	now := timex.LocalTime(time.Now())
	sc := &model.Scene{
		UserID:      0,
		Title:       title,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.CreateScene(ctx, sc); err != nil {
		return nil, err
	}
	return sc, nil
}

// UpdateAdminScene 只改公共场景。
func (s *Service) UpdateAdminScene(ctx context.Context, id uint64, title, description string) (*model.Scene, error) {
	sc, err := s.repo.GetScene(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "record not found") {
			return nil, errors.New("场景不存在")
		}
		return nil, err
	}
	if sc.UserID != 0 {
		return nil, errors.New("只能修改公共场景")
	}
	title, description, err = normalizeSceneText(title, description)
	if err != nil {
		return nil, err
	}
	sc.Title = title
	sc.Description = description
	sc.UpdatedAt = timex.LocalTime(time.Now())
	if err := s.repo.UpdateScene(ctx, sc); err != nil {
		return nil, err
	}
	return sc, nil
}

// DeleteAdminScene 删除公共场景。
func (s *Service) DeleteAdminScene(ctx context.Context, id uint64) error {
	sc, err := s.repo.GetScene(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "record not found") {
			return errors.New("场景不存在")
		}
		return err
	}
	if sc.UserID != 0 {
		return errors.New("只能删除公共场景")
	}
	return s.repo.DeleteScene(ctx, id)
}
