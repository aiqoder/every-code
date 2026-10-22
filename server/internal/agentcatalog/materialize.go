// Package agentcatalog 把「一台客户端的能力」套成「某一个服务端智能体」的运行视图。
// 人设、步数和工具白名单来自编译进程序的智能体定义；工具参数、技能正文、操作系统和工作区来自客户端上报。
package agentcatalog

import (
	"context"
	"fmt"
	"strings"

	"chengzi-agent-code-server/agents"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/pkg/clienthub/protocol"
)

// Materialize 用服务端智能体定义覆盖人设，并按白名单和技能绑定裁剪客户端清单。
func Materialize(ctx context.Context, svcCtx *svc.ServiceContext, agentID string, client protocol.CatalogPayload) (protocol.CatalogPayload, error) {
	row, ok := agents.ByID(agentID)
	if !ok {
		return protocol.CatalogPayload{}, fmt.Errorf("未知智能体：%s", strings.TrimSpace(agentID))
	}
	view := client
	view.AgentID = row.ID
	view.Name = row.Name
	view.Description = row.Description
	view.MaxSteps = row.MaxSteps
	view.SystemPrompt = strings.ReplaceAll(row.SystemPrompt, "{{workspace}}", strings.TrimSpace(client.Workspace))
	view.Tools = selectTools(row, client)
	view.Skills = FilterSkills(row.ID, client.Skills, assignments(ctx, svcCtx), systemSkillIDs(svcCtx))
	return view, nil
}

// FilterSkills 系统技能须已绑定到该智能体；本机自定义技能不受绑定表限制。
func FilterSkills(agentID string, incoming []protocol.SkillMeta, assignments map[string]string, systemIDs map[string]struct{}) []protocol.SkillMeta {
	agentID = strings.TrimSpace(agentID)
	if assignments == nil {
		assignments = map[string]string{}
	}
	if systemIDs == nil {
		systemIDs = map[string]struct{}{}
	}
	out := make([]protocol.SkillMeta, 0, len(incoming))
	for _, sk := range incoming {
		id := strings.TrimSpace(sk.ID)
		if id == "" {
			continue
		}
		if _, isSystem := systemIDs[id]; isSystem {
			assigned := assignments[id]
			if assigned == "" || assigned != agentID {
				continue
			}
		}
		out = append(out, sk)
	}
	return out
}

func selectTools(agent agents.Agent, client protocol.CatalogPayload) []protocol.ToolMeta {
	allow := make(map[string]struct{}, len(agent.Tools))
	for _, name := range agent.Tools {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		allow[name] = struct{}{}
	}
	out := make([]protocol.ToolMeta, 0, len(client.Tools))
	for _, tool := range client.Tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" || strings.HasPrefix(name, "page_") || agents.IsBuiltinTool(name) {
			continue
		}
		if _, ok := allow[name]; ok {
			out = append(out, tool)
		}
	}
	return out
}

func assignments(ctx context.Context, svcCtx *svc.ServiceContext) map[string]string {
	if svcCtx == nil || svcCtx.SkillBindingRepo == nil {
		return map[string]string{}
	}
	m, err := svcCtx.SkillBindingRepo.MapBySkillID(ctx)
	if err != nil {
		return map[string]string{}
	}
	return m
}

type systemSkillSource interface {
	SystemSkillIDs() map[string]struct{}
}

func systemSkillIDs(svcCtx *svc.ServiceContext) map[string]struct{} {
	if svcCtx == nil || svcCtx.Ade == nil {
		return map[string]struct{}{}
	}
	src, ok := svcCtx.Ade.(systemSkillSource)
	if !ok || src == nil {
		return map[string]struct{}{}
	}
	ids := src.SystemSkillIDs()
	if ids == nil {
		return map[string]struct{}{}
	}
	return ids
}
