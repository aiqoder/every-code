package tool

import (
	"context"
	"strings"
	"time"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/timex"
)

type addTodoTool struct {
	repo  repository.AdeRepository
	scope Scope
}

func (t *addTodoTool) Name() string { return AddTodo }

func (t *addTodoTool) Description() string {
	return "给当前任务追加一条待办。"
}

func (t *addTodoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title":  map[string]any{"type": "string"},
			"detail": map[string]any{"type": "string"},
		},
		"required":             []string{"title"},
		"additionalProperties": false,
	}
}

func (t *addTodoTool) Execute(ctx context.Context, argsJSON string, _ *agentloop.Runtime) (string, error) {
	var args planTodo
	if err := decode(argsJSON, &args); err != nil {
		return "", err
	}
	title := strings.TrimSpace(args.Title)
	if title == "" {
		return "", errf("待办标题不能为空")
	}
	task, err := t.scope.task(ctx, t.repo)
	if err != nil {
		return "", err
	}
	existing, err := t.repo.ListTodos(ctx, task.ID)
	if err != nil {
		return "", err
	}
	td := &model.Todo{
		TaskID: task.ID,
		Seq:    len(existing) + 1,
		Title:  title,
		Detail: strings.TrimSpace(args.Detail),
		Status: model.TodoStatusTodo,
		Source: model.TodoSourceAgent,
	}
	if err := t.repo.CreateTodo(ctx, td); err != nil {
		return "", err
	}
	return okJSON(map[string]any{"todoId": td.ID, "title": td.Title})
}

type removeTodoTool struct {
	repo  repository.AdeRepository
	scope Scope
}

func (t *removeTodoTool) Name() string { return RemoveTodo }

func (t *removeTodoTool) Description() string {
	return "删除当前任务的一条待办。todoId 来自 read_context。"
}

func (t *removeTodoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"todoId": map[string]any{"type": "number"},
		},
		"required":             []string{"todoId"},
		"additionalProperties": false,
	}
}

func (t *removeTodoTool) Execute(ctx context.Context, argsJSON string, _ *agentloop.Runtime) (string, error) {
	var args struct {
		TodoID uint64 `json:"todoId"`
	}
	if err := decode(argsJSON, &args); err != nil {
		return "", err
	}
	td, err := t.todoOnTask(ctx, args.TodoID)
	if err != nil {
		return "", err
	}
	if err := t.repo.DeleteTodo(ctx, td.ID); err != nil {
		return "", err
	}
	return okJSON(map[string]any{"todoId": td.ID})
}

type toggleTodoTool struct {
	repo  repository.AdeRepository
	scope Scope
}

func (t *toggleTodoTool) Name() string { return ToggleTodo }

func (t *toggleTodoTool) Description() string {
	return "把当前任务的一条待办在完成和未完成之间切换。status 传待办当前状态。"
}

func (t *toggleTodoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"todoId": map[string]any{"type": "number"},
			"status": map[string]any{"type": "string", "description": "当前状态，done 或其他"},
		},
		"required":             []string{"todoId", "status"},
		"additionalProperties": false,
	}
}

func (t *toggleTodoTool) Execute(ctx context.Context, argsJSON string, _ *agentloop.Runtime) (string, error) {
	var args struct {
		TodoID uint64 `json:"todoId"`
		Status string `json:"status"`
	}
	if err := decode(argsJSON, &args); err != nil {
		return "", err
	}
	td, err := t.todoOnTask(ctx, args.TodoID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(args.Status) == model.TodoStatusDone {
		td.Status = model.TodoStatusTodo
	} else {
		td.Status = model.TodoStatusDone
	}
	if err := t.repo.UpdateTodo(ctx, td); err != nil {
		return "", err
	}
	return okJSON(map[string]any{"todoId": td.ID, "status": td.Status})
}

func (t *removeTodoTool) todoOnTask(ctx context.Context, id uint64) (*model.Todo, error) {
	return todoOnTask(ctx, t.repo, t.scope, id)
}

func (t *toggleTodoTool) todoOnTask(ctx context.Context, id uint64) (*model.Todo, error) {
	return todoOnTask(ctx, t.repo, t.scope, id)
}

type completeTodoTool struct {
	repo  repository.AdeRepository
	scope Scope
}

func (t *completeTodoTool) Name() string { return CompleteTodo }

func (t *completeTodoTool) Description() string {
	return "把当前任务里你已经做完并提交的一条待办标成完成。todoId 来自 read_context。只标这一条，没做完的不要传。"
}

func (t *completeTodoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"todoId": map[string]any{"type": "number", "description": "read_context 返回的待办 id"},
		},
		"required":             []string{"todoId"},
		"additionalProperties": false,
	}
}

func (t *completeTodoTool) Execute(ctx context.Context, argsJSON string, _ *agentloop.Runtime) (string, error) {
	var args struct {
		TodoID uint64 `json:"todoId"`
	}
	if err := decode(argsJSON, &args); err != nil {
		return "", err
	}
	td, err := t.todoOnTask(ctx, args.TodoID)
	if err != nil {
		return "", err
	}
	if td.Status != model.TodoStatusDone {
		td.Status = model.TodoStatusDone
		td.UpdatedAt = timex.LocalTime(time.Now())
		if err := t.repo.UpdateTodo(ctx, td); err != nil {
			return "", err
		}
	}
	return okJSON(map[string]any{"todoId": td.ID, "status": td.Status})
}

func (t *completeTodoTool) todoOnTask(ctx context.Context, id uint64) (*model.Todo, error) {
	return todoOnTask(ctx, t.repo, t.scope, id)
}

func todoOnTask(ctx context.Context, repo repository.AdeRepository, scope Scope, id uint64) (*model.Todo, error) {
	if id == 0 {
		return nil, errf("请指定待办")
	}
	task, err := scope.task(ctx, repo)
	if err != nil {
		return nil, err
	}
	td, err := repo.GetTodo(ctx, id)
	if err != nil || td == nil || td.TaskID != task.ID {
		return nil, errf("待办不存在")
	}
	return td, nil
}
