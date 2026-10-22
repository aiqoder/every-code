package tool

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetToolLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetToolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetToolLogic {
	return &GetToolLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetToolLogic) GetTool(req *types.IdPathReq) (*common.ToolVO, error) {
	userID := middleware.UserID(l.ctx)
	t, err := l.svcCtx.ToolRepo.GetByID(l.ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("工具不存在")
	}
	// 工具仅在管理员后台维护，用户侧只能查看管理员工具（只读）
	if t.UserID != nil {
		return nil, fmt.Errorf("工具不存在")
	}
	return common.ToToolVO(t, userID, false), nil
}
