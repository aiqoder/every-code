package admin

import (
	"context"

	toollogic "chengzi-agent-code-server/internal/logic/tool"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteAdminToolLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteAdminToolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAdminToolLogic {
	return &DeleteAdminToolLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteAdminToolLogic) DeleteAdminTool(req *types.IdPathReq) error {
	if _, err := toollogic.EnsureEditableAdminTool(l.ctx, l.svcCtx.ToolRepo, req.Id); err != nil {
		return err
	}
	return l.svcCtx.ToolRepo.Delete(l.ctx, req.Id)
}
