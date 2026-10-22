package conversation

import (
	"context"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/logic/pathutil"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateConversationWorkspaceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateConversationWorkspaceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateConversationWorkspaceLogic {
	return &UpdateConversationWorkspaceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateConversationWorkspaceLogic) UpdateConversationWorkspace(req *types.UpdateConversationWorkspaceReq) (*common.ConversationVO, error) {
	id, err := pathutil.Uint64(l.ctx, "id")
	if err != nil {
		return nil, err
	}
	workspace := strings.TrimSpace(req.Workspace)
	if workspace == "" {
		return nil, fmt.Errorf("工作目录路径不能为空")
	}
	c, err := l.svcCtx.ConversationRepo.GetByID(l.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("会话不存在")
	}
	if c.UserID != middleware.UserID(l.ctx) {
		return nil, fmt.Errorf("无权访问该会话")
	}
	if err := l.svcCtx.ConversationRepo.UpdateWorkspace(l.ctx, id, workspace); err != nil {
		return nil, err
	}
	c.Workspace = workspace
	return common.ToConversationVO(c, c.Model), nil
}
