package admin

import (
	"context"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAdminToolsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAdminToolsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAdminToolsLogic {
	return &ListAdminToolsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAdminToolsLogic) ListAdminTools(req *types.PageReq) (*types.ToolListResp, error) {
	page, pageSize := common.NormalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize

	list, total, err := l.svcCtx.ToolRepo.ListAdmin(l.ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}

	vos := make([]common.ToolVO, 0, len(list))
	for i := range list {
		vos = append(vos, *common.ToToolVO(&list[i], 0, true))
	}
	return &types.ToolListResp{
		List:     vos,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
