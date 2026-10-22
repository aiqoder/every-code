package tool

import (
	"context"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListToolsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListToolsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListToolsLogic {
	return &ListToolsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListToolsLogic) ListTools(req *types.PageReq) (*types.ToolListResp, error) {
	userID := middleware.UserID(l.ctx)
	page, pageSize := common.NormalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize

	// 工具在管理员后台维护 http/builtin；客户端工具走 catalog，列表只读管理员工具
	list, total, err := l.svcCtx.ToolRepo.ListAdmin(l.ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}

	vos := make([]common.ToolVO, 0, len(list))
	for i := range list {
		vos = append(vos, *common.ToToolVO(&list[i], userID, false))
	}
	return &types.ToolListResp{
		List:     vos,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
