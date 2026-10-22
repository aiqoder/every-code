package ade

import (
	"fmt"
	"strings"

	"chengzi-agent-code-server/agents"
)

// StationAgentVO 一份编译进程序的智能体。
type StationAgentVO struct {
	AgentID      string   `json:"agentId"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"systemPrompt"`
	MaxSteps     int      `json:"maxSteps"`
	Tools        []string `json:"tools"`
}

// StationAgentClientVO 执行端登记目录所需的字段。人设、说明和步数留在服务端。
type StationAgentClientVO struct {
	AgentID string   `json:"agentId"`
	Name    string   `json:"name"`
	Tools   []string `json:"tools"`
}

// ListStationAgents 返回四个通道的固定定义。
func ListStationAgents() []StationAgentVO {
	rows := agents.All()
	out := make([]StationAgentVO, 0, len(rows))
	for _, row := range rows {
		tools := append([]string(nil), row.Tools...)
		out = append(out, StationAgentVO{
			AgentID:      row.ID,
			Name:         row.Name,
			Description:  row.Description,
			SystemPrompt: row.SystemPrompt,
			MaxSteps:     row.MaxSteps,
			Tools:        tools,
		})
	}
	return out
}

// ListStationAgentsForClient 只返回执行端登记目录和界面展示要用的字段。
func ListStationAgentsForClient() []StationAgentClientVO {
	rows := ListStationAgents()
	out := make([]StationAgentClientVO, 0, len(rows))
	for _, row := range rows {
		tools := row.Tools
		if tools == nil {
			tools = []string{}
		}
		out = append(out, StationAgentClientVO{
			AgentID: row.AgentID,
			Name:    row.Name,
			Tools:   tools,
		})
	}
	return out
}

// ListPipelineAgents 返回拆解器与复核。人设同样编译在程序里。
func ListPipelineAgents() []StationAgentVO {
	rows := agents.Pipeline()
	out := make([]StationAgentVO, 0, len(rows))
	for _, row := range rows {
		tools := append([]string(nil), row.Tools...)
		out = append(out, StationAgentVO{
			AgentID:      row.ID,
			Name:         row.Name,
			Description:  row.Description,
			SystemPrompt: row.SystemPrompt,
			MaxSteps:     row.MaxSteps,
			Tools:        tools,
		})
	}
	return out
}

// AgentByID 用客户端传入的智能体 id 找到编译进程序的定义。
func AgentByID(id string) (agents.Agent, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return agents.Agent{}, fmt.Errorf("请传入智能体 id")
	}
	row, ok := agents.ByID(id)
	if !ok {
		return agents.Agent{}, fmt.Errorf("未知智能体：%s", id)
	}
	return row, nil
}
