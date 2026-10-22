package admin

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAdminConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAdminConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAdminConversationLogic {
	return &GetAdminConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAdminConversationLogic) GetAdminConversation(req *types.IdPathReq) (*common.ConversationDetailVO, error) {
	c, err := l.svcCtx.ConversationRepo.GetByID(l.ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("会话不存在")
	}
	msgs := make([]common.MessageVO, 0, len(c.Messages))
	for _, m := range c.Messages {
		msgs = append(msgs, common.MessageVO{
			ID:         m.ID,
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			ToolName:   m.ToolName,
			Metadata:   m.Metadata,
			CreatedAt:  m.CreatedAt,
		})
	}
	return &common.ConversationDetailVO{
		ConversationVO: *common.ToConversationVO(c, c.Model),
		Messages:       msgs,
	}, nil
}
