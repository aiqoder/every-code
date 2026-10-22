package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/agentloop/askgate"
)

const ToolAskQuestion = "AskQuestion"

// AskAnswerTimeout 用户作答时限；超时后工具回填「自行决策」结果，由模型继续。
const AskAnswerTimeout = 5 * time.Minute

const askTimeoutResult = `{"status":"timeout","message":"用户未在 5 分钟内提交答案，请根据已有信息自行做出合理决策并继续完成任务。"}`

type askQuestionTool struct{}

// NewAskQuestionTool 向用户收集结构化选择题答案。
// 发出 ask_question 事件后阻塞等待用户提交；超过 AskAnswerTimeout 未作答则回填超时结果，
// 提示模型自行决策，本轮 Run 继续推进（不中断）。
func NewAskQuestionTool() agentloop.Tool { return &askQuestionTool{} }

func (t *askQuestionTool) Name() string { return ToolAskQuestion }

func (t *askQuestionTool) Description() string {
	return strings.TrimSpace(`向用户收集结构化选择题答案。适合需要用户在明确选项中做决定的场景。
不要用它询问可以用工具查到的事实，也不要把它当成开放式问答。
每题至少两个选项。界面会额外提供「其他」，用户可以填写自定义文本。
调用后会等待用户作答（最长 5 分钟）；超时则你会收到 timeout 结果，此时请自行决策并继续。`)
}

func (t *askQuestionTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "可选。整组问题的标题",
			},
			"questions": map[string]any{
				"type":        "array",
				"description": "要问的题目",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "题目 id，本轮唯一",
						},
						"prompt": map[string]any{
							"type":        "string",
							"description": "题干",
						},
						"allow_multiple": map[string]any{
							"type":        "boolean",
							"description": "为 true 时允许多选",
						},
						"options": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"id":    map[string]any{"type": "string"},
									"label": map[string]any{"type": "string"},
								},
								"required": []string{"id", "label"},
							},
						},
					},
					"required": []string{"id", "prompt", "options"},
				},
			},
		},
		"required": []string{"questions"},
	}
}

// Timeout 覆盖默认工具超时，需覆盖作答等待窗口。
func (t *askQuestionTool) Timeout() time.Duration { return AskAnswerTimeout }

type askArgs struct {
	Title     string             `json:"title"`
	Questions []askgate.Question `json:"questions"`
}

func (t *askQuestionTool) Execute(ctx context.Context, argsJSON string, rt *agentloop.Runtime) (string, error) {
	if rt.Ask == nil || rt.AskWait == nil {
		return "", fmt.Errorf("当前会话不能向用户提问")
	}
	var args askArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("AskQuestion 参数不是合法 JSON: %w", err)
	}
	spec, err := askgate.Normalize(askgate.Spec{Title: args.Title, Questions: args.Questions})
	if err != nil {
		return "", err
	}
	rawSpec, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	id, err := rt.Ask(ctx, rt.ToolCallID, string(rawSpec))
	if err != nil {
		return "", err
	}

	deadline := time.Now().Add(AskAnswerTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	timeoutSec := int(time.Until(deadline).Round(time.Second) / time.Second)
	if timeoutSec < 0 {
		timeoutSec = 0
	}

	rt.Emit(agentloop.Event{
		Type:       agentloop.EventAskQuestion,
		ToolCallID: rt.ToolCallID,
		Data: map[string]any{
			"questionId": id,
			"title":      spec.Title,
			"questions":  spec.Questions,
			"deadlineAt": deadline.UnixMilli(),
			"timeoutSec": timeoutSec,
		},
	})

	raw, err := rt.AskWait(ctx, id)
	if err != nil {
		// 作答时限到达：回填超时结果，让模型自行决策并继续本轮。
		if errors.Is(err, context.DeadlineExceeded) {
			return askTimeoutResult, nil
		}
		// 用户取消整轮等：按工具错误回填，由 runToolCall 包装。
		return "", err
	}
	return raw, nil
}
