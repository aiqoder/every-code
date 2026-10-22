package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/llmchat"
)

const ToolTodoWriter = "todowriter"

// TodoWriterOnlyToolkit 返回仅含 todowriter 的工具包。
func TodoWriterOnlyToolkit() []agentloop.Tool {
	return []agentloop.Tool{
		NewTodoWriterTool(),
	}
}

type todoWriterTool struct{}

// NewTodoWriterTool 创建/更新结构化任务计划。
func NewTodoWriterTool() agentloop.Tool { return &todoWriterTool{} }

func (t *todoWriterTool) Name() string { return ToolTodoWriter }

func (t *todoWriterTool) Description() string {
	return "使用此工具为当前会话创建结构化任务列表"
}

func (t *todoWriterTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "计划标题",
			},
			"steps": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":     map[string]any{"type": "string", "description": "步骤 id"},
						"title":  map[string]any{"type": "string", "description": "步骤说明"},
						"status": map[string]any{"type": "string", "enum": []string{"pending", "in_progress", "done", "cancelled"}},
					},
					"required": []string{"id", "title", "status"},
				},
			},
			"merge": map[string]any{
				"type":        "boolean",
				"description": "是否按 id 合并进现有列表。true：同 id 覆盖、新 id 追加、未出现的旧项保留；false：用本次 steps 整表替换。",
			},
		},
		"required": []string{"title", "steps", "merge"},
	}
}

type todoArgs struct {
	Title string     `json:"title"`
	Steps []todoStep `json:"steps"`
	Merge *bool      `json:"merge"`
}

type todoPayload struct {
	Title string     `json:"title"`
	Steps []todoStep `json:"steps"`
}

type todoStep struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

func (t *todoWriterTool) Execute(_ context.Context, argsJSON string, rt *agentloop.Runtime) (string, error) {
	var args todoArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid todowriter args: %w", err)
	}
	if args.Merge == nil {
		return "", fmt.Errorf("merge is required")
	}
	if args.Title == "" {
		return "", fmt.Errorf("title is required")
	}

	plan := todoPayload{Title: args.Title, Steps: args.Steps}
	if *args.Merge {
		if existing, ok := rt.Shared["plan"].(todoPayload); ok {
			plan.Steps = mergeStepsByID(existing.Steps, args.Steps)
		}
	}

	rt.Shared["plan"] = plan
	rt.Emit(agentloop.Event{Type: agentloop.EventPlanUpdated, Plan: plan})
	return llmchat.MustJSON(map[string]any{
		"ok":    true,
		"merge": *args.Merge,
		"plan":  plan,
		"hint":  "计划已更新，请继续执行下一步。",
	}), nil
}

func mergeStepsByID(existing, incoming []todoStep) []todoStep {
	out := make([]todoStep, len(existing))
	copy(out, existing)
	index := make(map[string]int, len(existing)+len(incoming))
	for i, s := range existing {
		if s.ID != "" {
			index[s.ID] = i
		}
	}
	for _, s := range incoming {
		if i, ok := index[s.ID]; ok {
			out[i] = s
			continue
		}
		index[s.ID] = len(out)
		out = append(out, s)
	}
	return out
}
