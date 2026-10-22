package admin

import (
	"context"

	toollogic "chengzi-agent-code-server/internal/logic/tool"
	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAdminToolLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAdminToolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAdminToolLogic {
	return &GetAdminToolLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAdminToolLogic) GetAdminTool(req *types.IdPathReq) (*common.ToolVO, error) {
	t, err := toollogic.EnsureAdminTool(l.ctx, l.svcCtx.ToolRepo, req.Id)
	if err != nil {
		return nil, err
	}
	return common.ToToolVO(t, 0, true), nil
}
