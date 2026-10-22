package health

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/database"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PingLogic {
	return &PingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PingLogic) Ping() (resp *types.HealthResp, err error) {
	if err := database.Ping(l.svcCtx.DB); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}
	return &types.HealthResp{Status: "ok"}, nil
}
