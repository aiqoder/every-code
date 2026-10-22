package tool

import (
	"context"
	"strings"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/agentloop"
)

type planTodo struct {
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Acceptance string `json:"acceptance"`
	DependsOn  string `json:"dependsOn"`
}

type planArgs struct {
	Name     string     `json:"name"`
	Overview string     `json:"overview"`
	Plan     string     `json:"plan"`
	Todos    []planTodo `json:"todos"`
}

type createTaskArgs struct {
	Description string `json:"description"`
	planArgs
}

func cleanTodos(items []planTodo) []planTodo {
	out := make([]planTodo, 0, len(items))
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		item.Title = title
		item.Detail = strings.TrimSpace(item.Detail)
		item.Acceptance = strings.TrimSpace(item.Acceptance)
		item.DependsOn = strings.TrimSpace(item.DependsOn)
		out = append(out, item)
	}
	return out
}

type savePlanTool struct {
	repo  repository.AdeRepository
	scope Scope
}

func (t *savePlanTool) Name() string { return SavePlan }

func (t *savePlanTool) Description() string {
	return "把整理好的名称、概览、润色需求和待办写回当前会话绑定的任务。不要新建任务。只适用于待执行的任务。"
}

func (t *savePlanTool) Parameters() map[string]any {
	return planParameters(true)
}

func (t *savePlanTool) Execute(ctx context.Context, argsJSON string, _ *agentloop.Runtime) (string, error) {
	var args planArgs
	if err := decode(argsJSON, &args); err != nil {
		return "", err
	}
	task, err := t.scope.task(ctx, t.repo)
	if err != nil {
		return "", err
	}
	if task.Status != model.TaskStatusPending {
		return "", errf("只有待执行的任务可以写入计划")
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return "", errf("请填写任务名称")
	}
	todos := cleanTodos(args.Todos)
	if len(todos) == 0 {
		return "", errf("计划至少要有一条待办")
	}
	polished := strings.TrimSpace(args.Plan)
	if polished == "" {
		polished = strings.TrimSpace(task.RawRequest)
	}
	if polished == "" {
		return "", errf("请填写任务描述")
	}
	task.Name = name
	task.Overview = strings.TrimSpace(args.Overview)
	task.Description = polished
	task.Planned = true
	task.Mode = model.TaskModePlan
	rows := make([]*model.Todo, 0, len(todos))
	for _, item := range todos {
		rows = append(rows, &model.Todo{
			Title:      item.Title,
			Detail:     item.Detail,
			Acceptance: item.Acceptance,
			DependsOn:  item.DependsOn,
		})
	}
	if err := t.repo.SaveTaskPlan(ctx, task, rows); err != nil {
		return "", err
	}
	return okJSON(map[string]any{"taskId": task.ID, "name": task.Name, "todoCount": len(rows)})
}

type createTaskTool struct {
	repo  repository.AdeRepository
	scope Scope
}

func (t *createTaskTool) Name() string { return CreateTask }

func (t *createTaskTool) Description() string {
	return "在当前项目下新建一条待执行任务。当前会话已经绑定任务时不要调用，应改用 save_plan。"
}

func (t *createTaskTool) Parameters() map[string]any {
	props := planParameters(false)
	properties, _ := props["properties"].(map[string]any)
	if properties == nil {
		properties = map[string]any{}
		props["properties"] = properties
	}
	properties["description"] = map[string]any{"type": "string", "description": "用户原始需求"}
	props["required"] = []string{"description", "name", "overview", "plan", "todos"}
	return props
}

func (t *createTaskTool) Execute(ctx context.Context, argsJSON string, _ *agentloop.Runtime) (string, error) {
	if t.scope.TaskID != 0 {
		return "", errf("当前会话已经绑定任务，请调用 save_plan")
	}
	var args createTaskArgs
	if err := decode(argsJSON, &args); err != nil {
		return "", err
	}
	description := strings.TrimSpace(args.Description)
	if description == "" {
		return "", errf("请填写任务描述")
	}
	p, err := t.scope.project(ctx, t.repo)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(args.Name)
	todos := cleanTodos(args.Todos)
	if len(todos) > 0 && name == "" {
		return "", errf("请填写任务名称")
	}
	task := &model.Task{
		UserID:      t.scope.UserID,
		ProjectID:   p.ID,
		Name:        name,
		Overview:    strings.TrimSpace(args.Overview),
		RawRequest:  description,
		Description: strings.TrimSpace(args.Plan),
		Mode:        model.TaskModePlan,
		Status:      model.TaskStatusPending,
		Planned:     len(todos) > 0,
		ClientID:    firstNonEmpty(t.scope.ClientID, p.ClientID),
	}
	if err := t.repo.CreateTask(ctx, task); err != nil {
		return "", err
	}
	for i, item := range todos {
		td := &model.Todo{
			TaskID:     task.ID,
			Seq:        i + 1,
			Title:      item.Title,
			Detail:     item.Detail,
			Acceptance: item.Acceptance,
			DependsOn:  item.DependsOn,
			Status:     model.TodoStatusTodo,
			Source:     model.TodoSourceAgent,
		}
		if err := t.repo.CreateTodo(ctx, td); err != nil {
			return "", err
		}
	}
	return okJSON(map[string]any{"taskId": task.ID, "name": task.Name})
}

type updateDescriptionTool struct {
	repo  repository.AdeRepository
	scope Scope
}

func (t *updateDescriptionTool) Name() string { return UpdateDescription }

func (t *updateDescriptionTool) Description() string {
	return "保存当前任务的需求原文。不改润色后的说明，也不改待办。"
}

func (t *updateDescriptionTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description": map[string]any{"type": "string"},
		},
		"required":             []string{"description"},
		"additionalProperties": false,
	}
}

func (t *updateDescriptionTool) Execute(ctx context.Context, argsJSON string, _ *agentloop.Runtime) (string, error) {
	var args struct {
		Description string `json:"description"`
	}
	if err := decode(argsJSON, &args); err != nil {
		return "", err
	}
	text := strings.TrimSpace(args.Description)
	if text == "" {
		return "", errf("需求描述不能为空")
	}
	task, err := t.scope.task(ctx, t.repo)
	if err != nil {
		return "", err
	}
	task.RawRequest = text
	task.Mode = model.TaskModePlan
	if err := t.repo.UpdateTask(ctx, task); err != nil {
		return "", err
	}
	return okJSON(map[string]any{"taskId": task.ID})
}

func planParameters(required bool) map[string]any {
	spec := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":     map[string]any{"type": "string", "description": "不超过 20 个汉字"},
			"overview": map[string]any{"type": "string", "description": "做完之后是什么样"},
			"plan":     map[string]any{"type": "string", "description": "润色后的工程需求"},
			"todos": map[string]any{
				"type":        "array",
				"description": "按依赖排序的待办",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title":      map[string]any{"type": "string"},
						"detail":     map[string]any{"type": "string"},
						"acceptance": map[string]any{"type": "string"},
						"dependsOn":  map[string]any{"type": "string"},
					},
					"required":             []string{"title"},
					"additionalProperties": false,
				},
			},
		},
		"additionalProperties": false,
	}
	if required {
		spec["required"] = []string{"name", "overview", "plan", "todos"}
	}
	return spec
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func errf(msg string) error { return &toolError{msg: msg} }

type toolError struct{ msg string }

func (e *toolError) Error() string { return e.msg }
