package admin

import (
	"context"

	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLlmProviderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetLlmProviderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLlmProviderLogic {
	return &GetLlmProviderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetLlmProviderLogic) GetLlmProvider(req *types.IdPathReq) (interface{}, error) {
	provider, err := ensurePlatformProvider(l.ctx, l.svcCtx.LlmProviderRepo, req.Id)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
