package tool

import (
	"context"
	"strings"
	"time"

	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/timex"
)

type saveWorktreeTool struct {
	repo  repository.AdeRepository
	scope Scope
}

type worktreeArgs struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
}

func (t *saveWorktreeTool) Name() string { return SaveWorktree }

func (t *saveWorktreeTool) Description() string {
	return "用 shell 创建或确认 worktree 之后，记下它的绝对路径和分支。调用成功后，后续文件工具和 shell 都会改到这个目录。"
}

func (t *saveWorktreeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "worktree 的绝对路径",
			},
			"branch": map[string]any{
				"type":        "string",
				"description": "任务分支，例如 ade/<任务id>",
			},
		},
		"required":             []string{"path", "branch"},
		"additionalProperties": false,
	}
}

func (t *saveWorktreeTool) Execute(ctx context.Context, argsJSON string, rt *agentloop.Runtime) (string, error) {
	var args worktreeArgs
	if err := decode(argsJSON, &args); err != nil {
		return "", err
	}
	path, err := absoluteClientPath(args.Path)
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(args.Branch)
	if branch == "" || strings.ContainsAny(branch, " \t\r\n") {
		return "", errf("请填写分支名")
	}
	task, err := t.scope.task(ctx, t.repo)
	if err != nil {
		return "", err
	}
	task.Worktree = path
	task.Branch = branch
	task.UpdatedAt = timex.LocalTime(time.Now())
	if err := t.repo.UpdateTask(ctx, task); err != nil {
		return "", err
	}
	if rt != nil {
		if rt.Shared == nil {
			rt.Shared = map[string]any{}
		}
		rt.Shared[agentloop.SharedWorkspace] = path
	}
	return okJSON(map[string]any{
		"taskId":   task.ID,
		"worktree": path,
		"branch":   branch,
	})
}

// absoluteClientPath 只判断像不像绝对路径。服务端和桌面可能不是同一套系统。
func absoluteClientPath(raw string) (string, error) {
	p := strings.TrimSpace(raw)
	if p == "" || strings.ContainsAny(p, "\r\n") {
		return "", errf("请填写 worktree 的绝对路径")
	}
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\\`) {
		return p, nil
	}
	if len(p) >= 3 && ((p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= 'a' && p[0] <= 'z')) && p[1] == ':' && (p[2] == '\\' || p[2] == '/') {
		return p, nil
	}
	return "", errf("worktree 必须是绝对路径")
}
