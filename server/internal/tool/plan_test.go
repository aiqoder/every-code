package tool

import (
	"context"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/agentloop"
)

func setupToolDB(t *testing.T) repository.AdeRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=private"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Task{}, &model.Todo{}, &model.Scene{}); err != nil {
		t.Fatal(err)
	}
	return repository.NewAdeRepository(db)
}

func TestSavePlanWritesCurrentTask(t *testing.T) {
	repo := setupToolDB(t)
	ctx := context.Background()
	project := &model.Project{UserID: 7, Name: "演示", LocalDir: "/work"}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	task := &model.Task{UserID: 7, ProjectID: project.ID, Status: model.TaskStatusPending, RawRequest: "做一个列表"}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	scope := Scope{UserID: 7, ProjectID: project.ID, TaskID: task.ID, Channel: model.ChannelCreate}
	tools := For(repo, scope)
	var save agentloop.Tool
	for _, item := range tools {
		if item.Name() == SaveEnvCheck {
			t.Fatal("planning should not save an env check")
		}
		if item.Name() == SavePlan {
			save = item
		}
	}
	if save == nil {
		t.Fatal("missing save_plan")
	}
	_, err := save.Execute(ctx, `{
		"name":"列表",
		"overview":"打开就能看到数据",
		"plan":"做一个列表页",
		"todos":[{"title":"加接口","detail":"改 handler","acceptance":"接口返回列表","dependsOn":""}]
	}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "列表" || !got.Planned {
		t.Fatalf("task: %+v", got)
	}
	todos, err := repo.ListTodos(ctx, task.ID)
	if err != nil || len(todos) != 1 || todos[0].Title != "加接口" {
		t.Fatalf("todos: %+v err=%v", todos, err)
	}
}

func TestPlanningWithoutTaskOmitsEnvAndPlanTools(t *testing.T) {
	repo := setupToolDB(t)
	scope := Scope{UserID: 7, ProjectID: 1, Channel: model.ChannelCreate}
	names := map[string]struct{}{}
	for _, item := range For(repo, scope) {
		names[item.Name()] = struct{}{}
	}
	if _, ok := names[SaveEnvCheck]; ok {
		t.Fatal("planning should not save an env check")
	}
	if _, ok := names[SavePlan]; ok {
		t.Fatal("unbound conversation should not save a plan")
	}
	if _, ok := names[CreateTask]; !ok {
		t.Fatal("create task missing")
	}
}

func TestEnvCheckScopeOnlyHasEnvTools(t *testing.T) {
	repo := setupToolDB(t)
	scope := Scope{UserID: 7, ProjectID: 1, Channel: model.ChannelCreate, EnvCheck: true}
	names := map[string]struct{}{}
	for _, item := range For(repo, scope) {
		names[item.Name()] = struct{}{}
	}
	if _, ok := names[SaveEnvCheck]; !ok {
		t.Fatal("env check tool missing")
	}
	if _, ok := names[ReadContext]; !ok {
		t.Fatal("read context missing")
	}
	if _, ok := names[SavePlan]; ok || namesHas(names, CreateTask) {
		t.Fatal("env check should not plan or create a task")
	}
}

func namesHas(names map[string]struct{}, name string) bool {
	_, ok := names[name]
	return ok
}

func TestExecuteChannelReadsPlanAndRecordsWorktree(t *testing.T) {
	repo := setupToolDB(t)
	scope := Scope{UserID: 7, ProjectID: 1, TaskID: 2, Channel: model.ChannelExecute}
	names := map[string]struct{}{}
	for _, item := range For(repo, scope) {
		names[item.Name()] = struct{}{}
	}
	if _, ok := names[ReadContext]; !ok {
		t.Fatal("execute should read the plan")
	}
	if _, ok := names[SaveWorktree]; !ok {
		t.Fatal("execute should record the worktree")
	}
	if _, ok := names[CompleteTodo]; !ok {
		t.Fatal("execute should mark finished todos")
	}
	if _, ok := names[SavePlan]; ok || namesHas(names, ToggleTodo) || namesHas(names, AddTodo) {
		t.Fatal("execute should not rewrite the plan")
	}
}

func TestTestChannelOnlyReadsContext(t *testing.T) {
	repo := setupToolDB(t)
	scope := Scope{UserID: 7, ProjectID: 1, TaskID: 2, Channel: model.ChannelTest}
	names := map[string]struct{}{}
	for _, item := range For(repo, scope) {
		names[item.Name()] = struct{}{}
	}
	if len(names) != 1 || !namesHas(names, ReadContext) {
		t.Fatalf("test tools: %v", names)
	}
	if got := For(repo, Scope{UserID: 7, ProjectID: 1, Channel: model.ChannelTest}); len(got) != 0 {
		t.Fatalf("unbound test tools: %d", len(got))
	}
}

func TestExecuteWithoutTaskHasNoDataTools(t *testing.T) {
	repo := setupToolDB(t)
	scope := Scope{UserID: 7, ProjectID: 1, Channel: model.ChannelExecute}
	if got := For(repo, scope); len(got) != 0 {
		t.Fatalf("unbound execute tools: %d", len(got))
	}
}

func TestSaveWorktreeSwitchesWorkspace(t *testing.T) {
	repo := setupToolDB(t)
	ctx := context.Background()
	project := &model.Project{UserID: 7, Name: "演示", LocalDir: "/work/demo"}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	task := &model.Task{UserID: 7, ProjectID: project.ID, Status: model.TaskStatusRunning}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	scope := Scope{UserID: 7, ProjectID: project.ID, TaskID: task.ID, Channel: model.ChannelExecute}
	var save agentloop.Tool
	for _, item := range For(repo, scope) {
		if item.Name() == SaveWorktree {
			save = item
		}
	}
	if save == nil {
		t.Fatal("missing save_worktree")
	}
	rt := &agentloop.Runtime{Shared: map[string]any{}}
	if _, err := save.Execute(ctx, `{"path":"/Users/me/.every-code/worktrees/demo/3","branch":"ade/3"}`, rt); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Worktree == "" || got.Branch != "ade/3" {
		t.Fatalf("task: %+v", got)
	}
	if rt.Shared[agentloop.SharedWorkspace] != got.Worktree {
		t.Fatalf("workspace: %+v", rt.Shared)
	}
	if _, err := save.Execute(ctx, `{"path":"relative","branch":"ade/3"}`, rt); err == nil {
		t.Fatal("relative path should be rejected")
	}
}

func TestCompleteTodoMarksOnlyThatRow(t *testing.T) {
	repo := setupToolDB(t)
	ctx := context.Background()
	project := &model.Project{UserID: 7, Name: "演示", LocalDir: "/work/demo"}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	task := &model.Task{UserID: 7, ProjectID: project.ID, Status: model.TaskStatusRunning}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	other := &model.Task{UserID: 7, ProjectID: project.ID, Status: model.TaskStatusRunning}
	if err := repo.CreateTask(ctx, other); err != nil {
		t.Fatal(err)
	}
	done := &model.Todo{TaskID: task.ID, Seq: 1, Title: "加接口", Status: model.TodoStatusTodo, Source: model.TodoSourceAgent}
	keep := &model.Todo{TaskID: task.ID, Seq: 2, Title: "加页面", Status: model.TodoStatusTodo, Source: model.TodoSourceAgent}
	foreign := &model.Todo{TaskID: other.ID, Seq: 1, Title: "别的任务", Status: model.TodoStatusTodo, Source: model.TodoSourceAgent}
	for _, td := range []*model.Todo{done, keep, foreign} {
		if err := repo.CreateTodo(ctx, td); err != nil {
			t.Fatal(err)
		}
	}
	scope := Scope{UserID: 7, ProjectID: project.ID, TaskID: task.ID, Channel: model.ChannelExecute, ConversationID: 42}
	var complete agentloop.Tool
	for _, item := range For(repo, scope) {
		if item.Name() == CompleteTodo {
			complete = item
		}
	}
	if complete == nil {
		t.Fatal("missing complete_todo")
	}
	if _, err := complete.Execute(ctx, fmt.Sprintf(`{"todoId":%d}`, done.ID), nil); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTodo(ctx, done.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.TodoStatusDone || got.ConversationID != 42 {
		t.Fatalf("completed: %+v", got)
	}
	left, err := repo.GetTodo(ctx, keep.ID)
	if err != nil || left.Status != model.TodoStatusTodo {
		t.Fatalf("untouched: %+v err=%v", left, err)
	}
	if _, err := complete.Execute(ctx, fmt.Sprintf(`{"todoId":%d}`, foreign.ID), nil); err == nil {
		t.Fatal("other task todo should be rejected")
	}
	if _, err := complete.Execute(ctx, fmt.Sprintf(`{"todoId":%d}`, done.ID), nil); err != nil {
		t.Fatal(err)
	}
}
