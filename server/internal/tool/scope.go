// Package tool 是服务端自建的对话工具，和 handler、logic 同级。
// 这些工具按会话绑定的项目和任务直接改数据库，不经过桌面页面。
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/agentloop"
)

// 创建工位上，智能体用来读写项目和任务的工具名。
const (
	ReadContext       = "read_context"
	SaveWorktree      = "save_worktree"
	SavePlan          = "save_plan"
	CreateTask        = "create_task"
	SaveEnvCheck      = "save_env_check"
	UpdateDescription = "update_description"
	AddTodo           = "add_todo"
	RemoveTodo        = "remove_todo"
	ToggleTodo        = "toggle_todo"
	CompleteTodo      = "complete_todo"
)

// Scope 一次对话能碰到的数据范围，来自会话归属。
type Scope struct {
	UserID    uint64
	ClientID  string
	ProjectID uint64
	TaskID    uint64
	Channel   string
	// EnvCheck 为真时只挂环境检测工具。任务规划不带这些工具。
	EnvCheck bool
}

// For 返回这次对话应带上的服务端工具。创建工位改计划和任务，执行工位读计划、切换本轮工作目录，并自行把做完的待办标完成。
func For(repo repository.AdeRepository, scope Scope) []agentloop.Tool {
	if repo == nil || scope.UserID == 0 || scope.ProjectID == 0 {
		return nil
	}
	if scope.EnvCheck {
		return []agentloop.Tool{
			&readContextTool{repo: repo, scope: scope},
			&saveEnvCheckTool{repo: repo, scope: scope},
		}
	}
	// 执行工位读取计划、把本轮工作目录切到 worktree，并用 complete_todo 把做完的待办写回。建目录交给客户端 shell。
	if strings.TrimSpace(scope.Channel) == model.ChannelExecute {
		if scope.TaskID == 0 {
			return nil
		}
		return []agentloop.Tool{
			&readContextTool{repo: repo, scope: scope},
			&saveWorktreeTool{repo: repo, scope: scope},
			&completeTodoTool{repo: repo, scope: scope},
		}
	}
	if strings.TrimSpace(scope.Channel) != model.ChannelCreate {
		return nil
	}
	out := []agentloop.Tool{
		&readContextTool{repo: repo, scope: scope},
	}
	if scope.TaskID == 0 {
		out = append(out, &createTaskTool{repo: repo, scope: scope})
		return out
	}
	out = append(out,
		&savePlanTool{repo: repo, scope: scope},
		&updateDescriptionTool{repo: repo, scope: scope},
		&addTodoTool{repo: repo, scope: scope},
		&removeTodoTool{repo: repo, scope: scope},
		&toggleTodoTool{repo: repo, scope: scope},
	)
	return out
}

func decode(argsJSON string, dest any) error {
	argsJSON = strings.TrimSpace(argsJSON)
	if argsJSON == "" {
		argsJSON = "{}"
	}
	if err := json.Unmarshal([]byte(argsJSON), dest); err != nil {
		return fmt.Errorf("参数不是合法 JSON")
	}
	return nil
}

func (s Scope) project(ctx context.Context, repo repository.AdeRepository) (*model.Project, error) {
	if s.ProjectID == 0 {
		return nil, fmt.Errorf("当前会话没有绑定项目")
	}
	p, err := repo.GetProject(ctx, s.ProjectID)
	if err != nil || p == nil || p.UserID != s.UserID {
		return nil, fmt.Errorf("项目不存在")
	}
	return p, nil
}

func (s Scope) task(ctx context.Context, repo repository.AdeRepository) (*model.Task, error) {
	if s.TaskID == 0 {
		return nil, fmt.Errorf("当前会话没有绑定任务")
	}
	t, err := repo.GetTask(ctx, s.TaskID)
	if err != nil || t == nil || t.UserID != s.UserID || t.ProjectID != s.ProjectID {
		return nil, fmt.Errorf("任务不存在")
	}
	return t, nil
}

func okJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
