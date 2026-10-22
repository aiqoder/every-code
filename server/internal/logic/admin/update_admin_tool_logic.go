package admin

import (
	"context"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/logic/pathutil"
	toollogic "chengzi-agent-code-server/internal/logic/tool"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAdminToolLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAdminToolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAdminToolLogic {
	return &UpdateAdminToolLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateAdminToolLogic) UpdateAdminTool(req *types.ToolReq) (*common.ToolVO, error) {
	id, err := pathutil.Uint64(l.ctx, "id")
	if err != nil {
		return nil, err
	}
	existing, err := toollogic.EnsureEditableAdminTool(l.ctx, l.svcCtx.ToolRepo, id)
	if err != nil {
		return nil, err
	}

	updated, err := toollogic.BuildToolEntity(l.ctx, l.svcCtx.ToolRepo, toToolInput(req), nil, id)
	if err != nil {
		return nil, err
	}
	updated.ID = existing.ID
	updated.CreatedAt = existing.CreatedAt
	if req.Version <= 0 {
		updated.Version = existing.Version
	}
	if err := l.svcCtx.ToolRepo.Update(l.ctx, updated); err != nil {
		return nil, err
	}
	return common.ToToolVO(updated, 0, true), nil
}
