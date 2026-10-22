package ade

import (
	"context"
	"strings"
	"time"

	"chengzi-agent-code-server/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// Scheduler 进程内调度。继续拉起已经点过开始、但当时桌面不在线的任务。
// 同一项目同时只开发一条任务；绑定的桌面不在线则跳过。
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
		// 启动后先补一轮，拉起本进程里还在等待桌面的任务。
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

	for _, id := range sch.svc.queuedIDs() {
		if sch.svc.isRunning(id) {
			continue
		}
		t, err := sch.svc.repo.GetTask(ctx, id)
		if err != nil {
			sch.svc.forgetQueued(id)
			continue
		}
		p, err := sch.svc.repo.GetProject(ctx, t.ProjectID)
		if err != nil {
			continue
		}
		clientID := strings.TrimSpace(p.ClientID)
		if clientID == "" {
			continue
		}
		if _, online := sch.svc.svcCtx.ClientHub.Get(t.UserID, clientID); !online {
			continue
		}
		sch.svc.dispatch(t.ID)
	}
}

// 保证 svc 包被引用（Service 持有 ServiceContext）。
var _ = svc.ServiceContext{}
