package ade

import (
	"context"
	"time"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// Scheduler 进程内调度。只领取执行中与重新执行的任务。
// 同一项目同时只开发一条任务；绑定的桌面不在线则跳过，不打成执行出错。
type Scheduler struct {
	svc      *Service
	interval time.Duration
}

// NewScheduler 构造调度器。
func NewScheduler(s *Service) *Scheduler {
	return &Scheduler{svc: s, interval: 8 * time.Second}
}

// Start 启动后台调度循环，进程退出时随 ctx 结束。
func (sch *Scheduler) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(sch.interval)
		defer ticker.Stop()
		// 启动后先补一轮，接管进程重启前遗留的执行中任务。
		sch.tick(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sch.tick(ctx)
			}
		}
	}()
}

func (sch *Scheduler) tick(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("ade scheduler panic: %v", r)
		}
	}()

	tasks, err := sch.svc.repo.ListTasksByStatus(ctx,
		model.TaskStatusRunning, model.TaskStatusRerun)
	if err != nil {
		return
	}
	if len(tasks) == 0 {
		return
	}

	// 同一项目同时只开发一条任务：项目已在跑的跳过。
	busy := make(map[uint64]struct{})
	for _, t := range tasks {
		if sch.svc.isRunning(t.ID) {
			busy[t.ProjectID] = struct{}{}
		}
	}
	for i := range tasks {
		t := tasks[i]
		if _, ok := busy[t.ProjectID]; ok {
			continue
		}
		if sch.svc.isRunning(t.ID) {
			continue
		}
		clientID := t.ClientID
		if clientID == "" {
			if p, err := sch.svc.repo.GetProject(ctx, t.ProjectID); err == nil {
				clientID = p.ClientID
			}
		}
		if clientID == "" {
			continue
		}
		if _, online := sch.svc.svcCtx.ClientHub.Get(t.UserID, clientID); !online {
			continue
		}
		// 原子领取：rerun → running；已是 running 则直接 dispatch。
		if t.Status == model.TaskStatusRerun {
			ok, err := sch.svc.Repo().ClaimTask(ctx, t.ID, model.TaskStatusRerun)
			if err != nil || !ok {
				continue
			}
		}
		busy[t.ProjectID] = struct{}{}
		sch.svc.dispatch(t.ID)
	}
}

// 保证 svc 包被引用（Service 持有 ServiceContext）。
var _ = svc.ServiceContext{}
