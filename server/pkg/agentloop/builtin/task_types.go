package builtin

import (
	"fmt"
	"strings"

	"chengzi-agent-code-server/pkg/agentloop"
)

const (
	subagentGeneral = "generalPurpose"
	subagentExplore = "explore"
	childMaxSteps   = 12
)

var exploreToolNames = map[string]struct{}{
	"read": {},
	"grep": {},
	"find": {},
}

func normalizeSubagentType(raw string) (string, error) {
	kind := strings.TrimSpace(raw)
	if kind == "" {
		return subagentGeneral, nil
	}
	switch kind {
	case subagentGeneral, subagentExplore:
		return kind, nil
	default:
		return "", fmt.Errorf("未知子代理类型 %s，允许：%s、%s", kind, subagentGeneral, subagentExplore)
	}
}

func childTools(parent []agentloop.Tool, kind string) []agentloop.Tool {
	out := make([]agentloop.Tool, 0, len(parent))
	for _, tool := range parent {
		if tool == nil {
			continue
		}
		name := tool.Name()
		if name == ToolTask {
			continue
		}
		if kind == subagentExplore {
			if _, ok := exploreToolNames[name]; !ok {
				continue
			}
		}
		out = append(out, tool)
	}
	return out
}

func childSystemPrompt(kind string) string {
	switch kind {
	case subagentExplore:
		return strings.TrimSpace(`你是代码探索子代理。只使用只读工具 read、grep、find 了解代码，不要修改文件，也不要执行命令。
你看不到用户原话和父代理的先前步骤，只根据任务说明工作。
完成后用简洁文字汇报：找到了什么、关键路径、结论。`)
	default:
		return strings.TrimSpace(`你是通用子代理。你看不到用户原话和父代理的先前步骤，只根据任务说明完成工作。
可以使用当前工具。不要再调用 Task。
完成后只返回父代理需要的结果，写清结论与依据。`)
	}
}
