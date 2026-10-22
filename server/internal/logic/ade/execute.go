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
	run := s.startRun(ctx, t, model.RunKindCoder, coder.ID)
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

	s.logLine(ctx, t, run, "开始执行，由开发智能体用终端准备 worktree")
	out, err := s.runAgent(ctx, t, p, run, coder.ID, workspace, "开始执行当前任务。")
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
	t.UpdatedAt = timex.LocalTime(nowTime())
	_ = s.repo.UpdateTask(ctx, t)
	s.finishRun(ctx, run, model.RunStatusDone, summarize(out), "")
	s.logLine(ctx, t, run, "开发完成，进入待测试")
}

func (s *Service) fail(ctx context.Context, t *model.Task, run *model.Run, msg string) {
	t.Status = model.TaskStatusFailed
	t.UpdatedAt = timex.LocalTime(nowTime())
	_ = s.repo.UpdateTask(ctx, t)
	s.finishRun(ctx, run, model.RunStatusFailed, "", msg)
	s.logLine(ctx, t, run, "执行出错："+msg)
}

func firstNonEmpty(list ...string) string {
	for _, v := range list {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func summarize(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= 2000 {
		return strings.TrimSpace(s)
	}
	return string(r[:2000]) + "\n…（已截断）"
}
