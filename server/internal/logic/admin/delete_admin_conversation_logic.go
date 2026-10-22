package admin

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/chatattach"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteAdminConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteAdminConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAdminConversationLogic {
	return &DeleteAdminConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteAdminConversationLogic) DeleteAdminConversation(req *types.IdPathReq) error {
	if _, err := l.svcCtx.ConversationRepo.GetByID(l.ctx, req.Id); err != nil {
		return fmt.Errorf("会话不存在")
	}
	if err := l.svcCtx.ConversationRepo.Delete(l.ctx, req.Id); err != nil {
		return err
	}
	chatattach.DiscardConversation(l.svcCtx.Config, req.Id)
	return nil
}
