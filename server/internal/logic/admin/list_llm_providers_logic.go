package admin

import (
	"context"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLlmProvidersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLlmProvidersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLlmProvidersLogic {
	return &ListLlmProvidersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLlmProvidersLogic) ListLlmProviders(req *types.LlmProviderListReq) (resp *types.LlmProviderListResp, err error) {
	if req.All {
		list, err := l.svcCtx.LlmProviderRepo.ListAll(l.ctx)
		if err != nil {
			return nil, err
		}
		return &types.LlmProviderListResp{List: list}, nil
	}
	page, pageSize := common.NormalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	list, total, err := l.svcCtx.LlmProviderRepo.List(l.ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}
	return &types.LlmProviderListResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
