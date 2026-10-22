package client

import (
	"context"

	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListClientsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListClientsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListClientsLogic {
	return &ListClientsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListClientsLogic) ListClients() (resp *types.ClientListResp, err error) {
	list := l.svcCtx.ClientHub.List(middleware.UserID(l.ctx))
	return &types.ClientListResp{List: list}, nil
}
