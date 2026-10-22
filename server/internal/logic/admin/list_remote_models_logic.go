package admin

import (
	"context"

	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRemoteModelsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListRemoteModelsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRemoteModelsLogic {
	return &ListRemoteModelsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListRemoteModelsLogic) ListRemoteModels(req *types.IdPathReq) (resp *types.RemoteModelsResp, err error) {
	if _, err := ensurePlatformProvider(l.ctx, l.svcCtx.LlmProviderRepo, req.Id); err != nil {
		return nil, err
	}
	list, err := FetchRemoteModels(l.ctx, l.svcCtx.LlmProviderRepo, req.Id)
	if err != nil {
		return nil, err
	}
	return &types.RemoteModelsResp{List: list}, nil
}
