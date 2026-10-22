package ade

import (
	"context"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/model"
)

// runTask 执行中：由 ade-coder 读取计划，在项目目录里完成待办。
// 做完哪几条由智能体调用 complete_todo 写回，这里不代标。
func (s *Service) runTask(ctx context.Context, taskID uint64) {
	t, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		s.forgetQueued(taskID)
		return
	}
	p, err := s.repo.GetProject(ctx, t.ProjectID)
	if err != nil {
		s.forgetQueued(taskID)
		return
	}
	clientID := strings.TrimSpace(p.ClientID)
	if clientID == "" {
		s.fail(ctx, t, nil, "没有可下发的客户端")
		return
	}
	if _, online := s.svcCtx.ClientHub.Get(t.UserID, clientID); !online {
		// 绑定的桌面不在线则跳过，不把任务打成执行出错。
		return
	}

	coder, err := AgentByID("ade-coder")
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

	workspace := strings.TrimSpace(p.LocalDir)
	if workspace == "" {
		s.fail(ctx, t, run, "项目还没有本机目录，无法执行")
		return
	}

	s.logLine(t, run, "开始执行")
	_, err = s.runAgent(ctx, t, p, run, coder.ID, workspace, "开始执行当前任务。")
	if err != nil {
		s.fail(ctx, t, run, err.Error())
		return
	}

	s.forgetQueued(t.ID)
	s.logLine(t, run, "开发完成")
}

func (s *Service) fail(_ context.Context, t *model.Task, run *agentRun, msg string) {
	if t == nil {
		return
	}
	s.forgetQueued(t.ID)
	if run == nil {
		s.bus.Begin(t.ID)
	}
	s.logLine(t, run, "执行出错："+msg)
}
