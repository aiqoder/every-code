package ade

import (
	"context"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/pkg/timex"
)

// runTask 执行中：由 ade-coder 读取计划，在客户端用 shell 准备 worktree 并完成待办。
// 做完哪几条由智能体调用 complete_todo 写回，这里不代标。
func (s *Service) runTask(ctx context.Context, taskID uint64) {
	t, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return
	}
	p, err := s.repo.GetProject(ctx, t.ProjectID)
	if err != nil {
		return
	}
	clientID := firstNonEmpty(t.ClientID, p.ClientID)
	if clientID == "" {
		s.fail(ctx, t, nil, "没有可下发的客户端")
		return
	}
	if _, online := s.svcCtx.ClientHub.Get(t.UserID, clientID); !online {
		// 绑定的桌面不在线则跳过，不把任务打成执行出错。
		return
	}

	coder, err := AgentByID(t.CoderAgentID)
	if err != nil {
		s.fail(ctx, t, nil, err.Error())
		return
	}
	run := s.startRun(t, model.RunKindCoder, coder.ID)
	defer func() {
		if r := recover(); r != nil {
			s.fail(ctx, t, run, fmt.Sprintf("执行异常：%v", r))
		}
	}()

	todos, _ := s.repo.ListTodos(ctx, t.ID)
	if len(todos) == 0 {
		s.fail(ctx, t, run, "任务没有待办，无法执行")
		return
	}

	// 已有 worktree 就直接在里面干；否则从仓库目录开始，让智能体用 shell 自己建。
	workspace := strings.TrimSpace(t.Worktree)
	if workspace == "" {
		workspace = strings.TrimSpace(p.LocalDir)
	}
	if workspace == "" {
		s.fail(ctx, t, run, "项目还没有本机目录，无法执行")
		return
	}

	s.logLine(t, run, "开始执行，由开发智能体用终端准备 worktree")
	_, err = s.runAgent(ctx, t, p, run, coder.ID, workspace, "开始执行当前任务。")
	if fresh, ferr := s.repo.GetTask(ctx, t.ID); ferr == nil {
		t = fresh
	}
	if err != nil {
		s.fail(ctx, t, run, err.Error())
		return
	}
	if strings.TrimSpace(t.Worktree) == "" {
		s.fail(ctx, t, run, "开发结束但没有记下 worktree")
		return
	}

	t.Status = model.TaskStatusWaitTest
	t.ErrorKind = ""
	t.ErrorAgentID = ""
	t.ErrorMessage = ""
	t.ErrorAt = timex.LocalTime{}
	t.UpdatedAt = timex.LocalTime(nowTime())
	_ = s.repo.UpdateTask(ctx, t)
	s.logLine(t, run, "开发完成，进入待测试")
}

func (s *Service) fail(ctx context.Context, t *model.Task, run *agentRun, msg string) {
	if run == nil {
		s.bus.Begin(t.ID)
	}
	t.Status = model.TaskStatusFailed
	t.ErrorMessage = msg
	t.ErrorAt = timex.LocalTime(nowTime())
	t.ErrorKind = ""
	t.ErrorAgentID = ""
	if run != nil {
		t.ErrorKind = run.Kind
		t.ErrorAgentID = run.AgentID
	}
	t.UpdatedAt = timex.LocalTime(nowTime())
	_ = s.repo.UpdateTask(ctx, t)
	s.logLine(t, run, "执行出错："+msg)
}

func firstNonEmpty(list ...string) string {
	for _, v := range list {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
