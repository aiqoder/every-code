package conversation

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/chatattach"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteConversationLogic {
	return &DeleteConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteConversationLogic) DeleteConversation(req *types.IdPathReq) error {
	c, err := l.svcCtx.ConversationRepo.GetByID(l.ctx, req.Id)
	if err != nil {
		return fmt.Errorf("会话不存在")
	}
	if c.UserID != middleware.UserID(l.ctx) {
		return fmt.Errorf("无权删除该会话")
	}
	if err := l.svcCtx.ConversationRepo.Delete(l.ctx, req.Id); err != nil {
		return err
	}
	chatattach.DiscardConversation(l.svcCtx.Config, req.Id)
	return nil
}
