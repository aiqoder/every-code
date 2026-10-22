package ade

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chengzi-agent-code-server/agents"
	"chengzi-agent-code-server/internal/agentcatalog"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/pkg/clienthub/protocol"
	"chengzi-agent-code-server/pkg/timex"
)

// SystemSkillVO 管理端展示的系统技能与绑定。
type SystemSkillVO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FileCount   int    `json:"fileCount"`
	UpdatedAt   string `json:"updatedAt"`
	AgentID     string `json:"agentId"`
}

// SystemSkillInput 保存一行绑定。
type SystemSkillInput struct {
	SkillID string `json:"skillId"`
	AgentID string `json:"agentId"`
}

// AgentOption 可选智能体（工位 + 内置流水线）。
type AgentOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListSystemSkills 列出磁盘上的系统技能并合并绑定。
func ListSystemSkills(ctx context.Context, svcCtx *svc.ServiceContext, store *SkillStore) ([]SystemSkillVO, error) {
	if store == nil {
		return nil, fmt.Errorf("技能仓库未就绪")
	}
	metas, err := store.List()
	if err != nil {
		return nil, err
	}
	assignments := skillAssignmentMap(ctx, svcCtx)
	out := make([]SystemSkillVO, 0, len(metas))
	for _, meta := range metas {
		out = append(out, SystemSkillVO{
			ID:          meta.ID,
			Name:        meta.Name,
			Description: meta.Description,
			FileCount:   meta.FileCount,
			UpdatedAt:   meta.UpdatedAt,
			AgentID:     assignments[meta.ID],
		})
	}
	return out, nil
}

// SaveSystemSkills 按技能 id 更新绑定。未出现在列表里的技能保持原样。
func SaveSystemSkills(ctx context.Context, svcCtx *svc.ServiceContext, store *SkillStore, items []SystemSkillInput) ([]SystemSkillVO, error) {
	if svcCtx == nil || svcCtx.SkillBindingRepo == nil {
		return nil, fmt.Errorf("技能绑定未就绪")
	}
	if store == nil {
		return nil, fmt.Errorf("技能仓库未就绪")
	}
	metas, err := store.List()
	if err != nil {
		return nil, err
	}
	known := make(map[string]struct{}, len(metas))
	for _, meta := range metas {
		known[meta.ID] = struct{}{}
	}
	now := timex.LocalTime(time.Now())
	rows := make([]model.SkillBinding, 0, len(items))
	for _, item := range items {
		skillID := strings.TrimSpace(item.SkillID)
		if skillID == "" {
			continue
		}
		if _, ok := known[skillID]; !ok {
			return nil, fmt.Errorf("未知技能：%s", skillID)
		}
		agentID := strings.TrimSpace(item.AgentID)
		if agentID != "" {
			if !agentKnown(agentID) {
				return nil, fmt.Errorf("未知智能体：%s", agentID)
			}
		}
		rows = append(rows, model.SkillBinding{
			SkillID:   skillID,
			AgentID:   agentID,
			UpdatedAt: now,
			CreatedAt: now,
		})
	}
	if err := svcCtx.SkillBindingRepo.SaveAll(ctx, rows); err != nil {
		return nil, err
	}
	return ListSystemSkills(ctx, svcCtx, store)
}

// AgentOptions 管理端下拉：工位智能体 + 拆解器/复核。
func AgentOptions(context.Context, *svc.ServiceContext) []AgentOption {
	return agentOptionsFromStation()
}

func agentOptionsFromStation() []AgentOption {
	rows := append(agents.Pipeline(), agents.All()...)
	out := make([]AgentOption, 0, len(rows))
	seen := map[string]struct{}{}
	for _, row := range rows {
		id := strings.TrimSpace(row.ID)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		name := strings.TrimSpace(row.Name)
		if name == "" {
			name = id
		}
		out = append(out, AgentOption{ID: id, Name: name})
	}
	return out
}

func agentKnown(agentID string) bool {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return false
	}
	for _, opt := range agentOptionsFromStation() {
		if opt.ID == agentID {
			return true
		}
	}
	return false
}

func skillAssignmentMap(ctx context.Context, svcCtx *svc.ServiceContext) map[string]string {
	if svcCtx == nil || svcCtx.SkillBindingRepo == nil {
		return map[string]string{}
	}
	m, err := svcCtx.SkillBindingRepo.MapBySkillID(ctx)
	if err != nil {
		return map[string]string{}
	}
	return m
}

// FilterSkillsForAgent 系统技能须已绑定到该智能体；本机自定义技能不受绑定表限制。
func FilterSkillsForAgent(ctx context.Context, svcCtx *svc.ServiceContext, agentID string, incoming []protocol.SkillMeta) []protocol.SkillMeta {
	return agentcatalog.FilterSkills(agentID, incoming, skillAssignmentMap(ctx, svcCtx), systemSkillIDs(svcCtx))
}

func systemSkillIDs(svcCtx *svc.ServiceContext) map[string]struct{} {
	if svcCtx == nil || svcCtx.Ade == nil {
		return map[string]struct{}{}
	}
	s, ok := svcCtx.Ade.(*Service)
	if !ok || s == nil {
		return map[string]struct{}{}
	}
	return s.SystemSkillIDs()
}
