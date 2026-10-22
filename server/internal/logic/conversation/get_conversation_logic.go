package conversation

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationLogic {
	return &GetConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConversationLogic) GetConversation(req *types.IdPathReq) (interface{}, error) {
	c, err := l.svcCtx.ConversationRepo.GetByID(l.ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("会话不存在")
	}
	if c.UserID != middleware.UserID(l.ctx) {
		return nil, fmt.Errorf("无权访问该会话")
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
	vo := common.ToConversationVO(c, c.Model)
	if usage := resolveConversationUsage(c); usage != nil {
		vo.ContextUsage = usage
	}
	return &common.ConversationDetailVO{
		ConversationVO: *vo,
		Messages:       msgs,
	}, nil
}
