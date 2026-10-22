package builtin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/llmchat"
)

const ToolTask = "Task"

type taskTool struct{}

// NewTaskTool 按 Cursor Task 的本地契约启动子代理。
func NewTaskTool() agentloop.Tool { return &taskTool{} }

func (t *taskTool) Name() string { return ToolTask }

func (t *taskTool) Timeout() time.Duration { return 10 * time.Minute }

func (t *taskTool) Description() string {
	return strings.TrimSpace(`启动子代理处理可以独立完成的多步任务。
窄问题、读单个文件、在少量文件里搜一个符号时，直接用 read、grep、find，不要用 Task。
同一轮里多个 Task 会按调用顺序依次执行。
子代理看不到用户原话和你之前的步骤，prompt 必须写清目标、路径和需要返回的信息。
description 是展示给用户的短标题（约 3 到 5 个词）。
subagent_type：generalPurpose 继承当前工具（不含 Task）；explore 只读探索代码（仅 read、grep、find）。省略时为 generalPurpose。`)
}

func (t *taskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description": map[string]any{
				"type":        "string",
				"description": "给用户看的短标题，具体且不要和最近的标题重复",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "交给子代理的完整任务说明。子代理看不到当前对话，必须自包含",
			},
			"subagent_type": map[string]any{
				"type":        "string",
				"enum":        []string{subagentGeneral, subagentExplore},
				"description": "generalPurpose 或 explore。省略时为 generalPurpose",
			},
		},
		"required": []string{"description", "prompt"},
	}
}

type taskArgs struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
}

func (t *taskTool) Execute(ctx context.Context, argsJSON string, rt *agentloop.Runtime) (string, error) {
	var args taskArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("Task 参数不是合法 JSON: %w", err)
	}
	description := strings.TrimSpace(args.Description)
	prompt := strings.TrimSpace(args.Prompt)
	if description == "" || prompt == "" {
		return "", fmt.Errorf("description 和 prompt 都不能为空")
	}
	kind, err := normalizeSubagentType(args.SubagentType)
	if err != nil {
		return "", err
	}
	if rt.Depth+1 > rt.MaxDepth {
		return "", fmt.Errorf("子代理嵌套已达上限（%d）", rt.MaxDepth)
	}

	agentID := newAgentID()
	data := map[string]any{
		"agentId":      agentID,
		"description":  description,
		"subagentType": kind,
	}
	rt.Emit(agentloop.Event{
		Type:             agentloop.EventSubagentStarted,
		Goal:             description,
		ToolCallID:       rt.ToolCallID,
		ParentToolCallID: rt.ToolCallID,
		Data:             data,
	})

	res, err := agentloop.Run(ctx, agentloop.Config{
		Chat:               rt.Chat,
		Model:              rt.Model,
		Messages:           []llmchat.Message{{Role: llmchat.RoleUser, Content: prompt}},
		Tools:              childTools(rt.Tools, kind),
		MaxSteps:           childMaxSteps,
		Depth:              rt.Depth + 1,
		MaxDepth:           rt.MaxDepth,
		Shared:             map[string]any{},
		InputContextLimit:  rt.InputContextLimit,
		OutputContextLimit: rt.OutputContextLimit,
		Reasoning:          rt.Reasoning,
		ReasoningEffort:    rt.ReasoningEffort,
		SystemPrompt:       childSystemPrompt(kind),
		Ask:                rt.Ask,
		AskWait:            rt.AskWait,
		OnEvent: func(ev agentloop.Event) {
			forwardChildEvent(rt, agentID, description, kind, ev)
		},
	})
	if err != nil {
		rt.Emit(agentloop.Event{
			Type:             agentloop.EventSubagentFinished,
			Goal:             description,
			ToolCallID:       rt.ToolCallID,
			ParentToolCallID: rt.ToolCallID,
			Error:            err.Error(),
			Data:             data,
		})
		return "", err
	}
	text := strings.TrimSpace(res.FinalContent)
	if text == "" {
		text = "子代理没有产出文字结果。"
	}
	rt.Emit(agentloop.Event{
		Type:             agentloop.EventSubagentFinished,
		Goal:             description,
		ToolCallID:       rt.ToolCallID,
		ParentToolCallID: rt.ToolCallID,
		Result:           text,
		Data:             data,
	})
	return llmchat.MustJSON(map[string]any{
		"agentId":      agentID,
		"description":  description,
		"subagentType": kind,
		"result":       text,
	}), nil
}

func forwardChildEvent(rt *agentloop.Runtime, agentID, description, kind string, ev agentloop.Event) {
	switch ev.Type {
	case agentloop.EventPlanUpdated,
		agentloop.EventToolCallStart, agentloop.EventToolCallEnd,
		agentloop.EventError, agentloop.EventAskQuestion, agentloop.EventRunWarning,
		agentloop.EventSubagentStarted, agentloop.EventSubagentFinished:
	default:
		return
	}
	if ev.ParentToolCallID == "" {
		ev.ParentToolCallID = rt.ToolCallID
	}
	copied := make(map[string]any, len(ev.Data)+3)
	for k, v := range ev.Data {
		copied[k] = v
	}
	copied["agentId"] = agentID
	copied["description"] = description
	copied["subagentType"] = kind
	ev.Data = copied
	ev.Goal = description
	rt.Emit(ev)
}

func newAgentID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "agent"
	}
	return hex.EncodeToString(b[:])
}
