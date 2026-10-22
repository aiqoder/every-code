package tool

import (
	"context"
	"encoding/json"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/agentloop"
)

type readContextTool struct {
	repo  repository.AdeRepository
	scope Scope
}

func (t *readContextTool) Name() string { return ReadContext }

func (t *readContextTool) Description() string {
	return "读取当前会话绑定的项目、任务、待办和可选场景。动手之前先看这里，不要猜当前是哪条任务。"
}

func (t *readContextTool) Parameters() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
}

func (t *readContextTool) Execute(ctx context.Context, _ string, _ *agentloop.Runtime) (string, error) {
	p, err := t.scope.project(ctx, t.repo)
	if err != nil {
		return "", err
	}
	scenes, err := t.repo.ListVisibleScenes(ctx, t.scope.UserID)
	if err != nil {
		return "", err
	}
	view := map[string]any{
		"project": projectView(p),
		"scenes":  sceneViews(scenes),
		"task":    nil,
	}
	if t.scope.TaskID > 0 {
		task, err := t.scope.task(ctx, t.repo)
		if err != nil {
			return "", err
		}
		todos, err := t.repo.ListTodos(ctx, task.ID)
		if err != nil {
			return "", err
		}
		view["task"] = taskView(task, todos)
	}
	return okJSON(view)
}

func projectView(p *model.Project) map[string]any {
	var env []envItem
	if p.EnvCheck != "" {
		_ = json.Unmarshal([]byte(p.EnvCheck), &env)
	}
	if env == nil {
		env = []envItem{}
	}
	return map[string]any{
		"id":          p.ID,
		"name":        p.Name,
		"description": p.Description,
		"localDir":    p.LocalDir,
		"sceneId":     p.SceneID,
		"envReady":    p.EnvReady,
		"envCheck":    env,
	}
}

func sceneViews(list []model.Scene) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, sc := range list {
		out = append(out, map[string]any{
			"id":          sc.ID,
			"title":       sc.Title,
			"description": sc.Description,
		})
	}
	return out
}

func taskView(t *model.Task, todos []model.Todo) map[string]any {
	rows := make([]map[string]any, 0, len(todos))
	for _, td := range todos {
		rows = append(rows, map[string]any{
			"id":         td.ID,
			"title":      td.Title,
			"detail":     td.Detail,
			"acceptance": td.Acceptance,
			"dependsOn":  td.DependsOn,
			"status":     td.Status,
			"source":     td.Source,
		})
	}
	return map[string]any{
		"id":          t.ID,
		"name":        t.Name,
		"overview":    t.Overview,
		"description": t.Description,
		"rawRequest":  t.RawRequest,
		"status":      t.Status,
		"planned":     t.Planned,
		"branch":      t.Branch,
		"worktree":    t.Worktree,
		"todos":       rows,
	}
}
