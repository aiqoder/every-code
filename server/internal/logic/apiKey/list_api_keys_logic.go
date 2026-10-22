package apiKey

import (
	"context"

	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListApiKeysLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListApiKeysLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListApiKeysLogic {
	return &ListApiKeysLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListApiKeysLogic) ListApiKeys() (resp *types.ApiKeyListResp, err error) {
	list, err := l.svcCtx.ApiKeyRepo.ListByUser(l.ctx, middleware.UserID(l.ctx))
	if err != nil {
		return nil, err
	}
	return &types.ApiKeyListResp{List: list}, nil
}
