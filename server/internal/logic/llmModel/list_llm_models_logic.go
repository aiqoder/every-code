package llmModel

import (
	"context"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLlmModelsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLlmModelsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLlmModelsLogic {
	return &ListLlmModelsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLlmModelsLogic) ListLlmModels(req *types.LlmModelListReq) (resp *types.LlmModelListResp, err error) {
	page, pageSize := common.NormalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	userID := middleware.UserID(l.ctx)
	list, total, err := l.svcCtx.LlmModelRepo.ListForUser(l.ctx, userID, req.ProviderId, offset, pageSize)
	if err != nil {
		return nil, err
	}
	vos := make([]common.LlmModelVO, 0, len(list))
	for i := range list {
		vos = append(vos, *common.ToLlmModelVOFor(&list[i], userID, false))
	}
	return &types.LlmModelListResp{
		List:     vos,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
