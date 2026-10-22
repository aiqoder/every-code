package admin

import (
	"context"

	toollogic "chengzi-agent-code-server/internal/logic/tool"
	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateAdminToolLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateAdminToolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAdminToolLogic {
	return &CreateAdminToolLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateAdminToolLogic) CreateAdminTool(req *types.ToolReq) (*common.ToolVO, error) {
	t, err := toollogic.BuildToolEntity(l.ctx, l.svcCtx.ToolRepo, toToolInput(req), nil, 0)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.ToolRepo.Create(l.ctx, t); err != nil {
		return nil, err
	}
	return common.ToToolVO(t, 0, true), nil
}
