// Package agents 是智能体的固定定义。
// 工位三条、拆解、复核、环境检测各有一个目录，目录里只有 AGENT.md：YAML 头加系统提示词。
// 这些文件随程序编译，不再写入数据库。
package agents

import (
	"embed"
	"fmt"
	"strconv"
	"strings"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/pkg/agentloop/builtin"
)

//go:embed */AGENT.md
var files embed.FS

// Agent 一份智能体。
type Agent struct {
	ID           string
	Name         string
	Description  string
	SystemPrompt string
	MaxSteps     int
	Tools        []string
}

var (
	all      []Agent
	pipeline []Agent
	byID     map[string]Agent
)

// pipelineDirs 流水线智能体目录。它们不是工位通道，不出现在工位列表里。
var pipelineDirs = []string{"planner", "reviewer"}

// sideDirs 不进入工位和流水线列表的智能体。环境检测只在按钮触发的会话里使用。
var sideDirs = []string{"envcheck"}

// EnvCheckID 环境检测智能体。只在「检查开发环境」时使用。
const EnvCheckID = "ade-envcheck"

// allowedTools 是智能体 tools 白名单。客户端工具按清单裁剪，内置工具按同一份名单挂载。
var allowedTools = map[string]struct{}{
	"read": {}, "write": {}, "edit": {}, "find": {}, "grep": {}, "shell_exec": {},
	builtin.ToolAskQuestion: {},
	builtin.ToolTask:        {},
	builtin.ToolTodoWriter:  {},
}

// IsBuiltinTool 报告该名称是服务端内置工具，不从客户端能力清单里取。
func IsBuiltinTool(name string) bool {
	switch strings.TrimSpace(name) {
	case builtin.ToolAskQuestion, builtin.ToolTask, builtin.ToolTodoWriter:
		return true
	default:
		return false
	}
}

func init() {
	stations, pipelines, sides, err := load()
	if err != nil {
		panic(err)
	}
	all = stations
	pipeline = pipelines
	byID = make(map[string]Agent, len(stations)+len(pipelines)+len(sides))
	for _, agent := range stations {
		byID[agent.ID] = agent
	}
	for _, agent := range pipelines {
		byID[agent.ID] = agent
	}
	for _, agent := range sides {
		byID[agent.ID] = agent
	}
}

// All 按创建、执行、部署的固定顺序返回。
func All() []Agent {
	out := make([]Agent, len(all))
	copy(out, all)
	return out
}

// Pipeline 返回拆解器与复核。顺序固定，不进入工位列表。
func Pipeline() []Agent {
	out := make([]Agent, len(pipeline))
	copy(out, pipeline)
	return out
}

// ByID 按智能体 id 查找。工位、流水线和环境检测都能查到。
func ByID(id string) (Agent, bool) {
	agent, ok := byID[strings.TrimSpace(id)]
	return agent, ok
}

func load() ([]Agent, []Agent, []Agent, error) {
	seenID := map[string]struct{}{}
	stations, err := readAgents(model.StationChannels(), seenID)
	if err != nil {
		return nil, nil, nil, err
	}
	pipelines, err := readAgents(pipelineDirs, seenID)
	if err != nil {
		return nil, nil, nil, err
	}
	sides, err := readAgents(sideDirs, seenID)
	if err != nil {
		return nil, nil, nil, err
	}
	return stations, pipelines, sides, nil
}

func readAgents(dirs []string, seenID map[string]struct{}) ([]Agent, error) {
	out := make([]Agent, 0, len(dirs))
	for _, dir := range dirs {
		body, err := files.ReadFile(dir + "/AGENT.md")
		if err != nil {
			return nil, fmt.Errorf("缺少智能体文件 %s/AGENT.md", dir)
		}
		agent, err := parse(string(body))
		if err != nil {
			return nil, fmt.Errorf("%s/AGENT.md: %w", dir, err)
		}
		if _, ok := seenID[agent.ID]; ok {
			return nil, fmt.Errorf("智能体 id 重复：%s", agent.ID)
		}
		seenID[agent.ID] = struct{}{}
		out = append(out, agent)
	}
	return out, nil
}

func parse(raw string) (Agent, error) {
	fm, body, err := splitFrontmatter(raw)
	if err != nil {
		return Agent{}, err
	}
	id := fm["id"]
	name := fm["name"]
	if id == "" || name == "" {
		return Agent{}, fmt.Errorf("id 与 name 不能为空")
	}
	steps, err := strconv.Atoi(fm["maxSteps"])
	if err != nil || steps < 1 || steps > 200 {
		return Agent{}, fmt.Errorf("maxSteps 需在 1 到 200 之间")
	}
	tools, err := parseTools(fm["tools"])
	if err != nil {
		return Agent{}, err
	}
	if !strings.Contains(body, "{{workspace}}") {
		return Agent{}, fmt.Errorf("系统提示词需要包含 {{workspace}}")
	}
	return Agent{
		ID:           id,
		Name:         name,
		Description:  fm["description"],
		SystemPrompt: body,
		MaxSteps:     steps,
		Tools:        tools,
	}, nil
}

func splitFrontmatter(raw string) (map[string]string, string, error) {
	trim := strings.TrimLeft(raw, " \t\r\n")
	if !strings.HasPrefix(trim, "---") {
		return nil, "", fmt.Errorf("缺少 YAML 头")
	}
	rest := trim[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, "", fmt.Errorf("YAML 头没有结束")
	}
	fm := map[string]string{}
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			return nil, "", fmt.Errorf("无法解析：%s", line)
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)
		fm[key] = val
	}
	body := strings.TrimSpace(rest[end+4:])
	if body == "" {
		return nil, "", fmt.Errorf("系统提示词为空")
	}
	return fm, body, nil
}

func parseTools(raw string) ([]string, error) {
	out := make([]string, 0, 4)
	seen := map[string]struct{}{}
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := allowedTools[name]; !ok {
			return nil, fmt.Errorf("不允许的工具：%s", name)
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("至少指定一个工具")
	}
	return out, nil
}
