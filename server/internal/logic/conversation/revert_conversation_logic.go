package conversation

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/chatattach"
	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RevertConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevertConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevertConversationLogic {
	return &RevertConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevertConversationLogic) RevertConversation(req *types.RevertConversationReq) (interface{}, error) {
	c, err := l.svcCtx.ConversationRepo.GetByID(l.ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("会话不存在")
	}
	if c.UserID != middleware.UserID(l.ctx) {
		return nil, fmt.Errorf("无权访问该会话")
	}
	if req.MessageId == 0 {
		return nil, fmt.Errorf("消息 ID 无效")
	}
	found := false
	for _, m := range c.Messages {
		if m.ID == req.MessageId {
			found = true
			if m.Role != "user" {
				return nil, fmt.Errorf("只能回退到用户消息")
			}
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("消息不存在")
	}
	for _, m := range c.Messages {
		if m.ID >= req.MessageId {
			chatattach.DiscardMessage(l.svcCtx.Config, req.Id, m.ID)
		}
	}
	if err := l.svcCtx.ConversationRepo.DeleteMessagesFromID(l.ctx, req.Id, req.MessageId); err != nil {
		return nil, fmt.Errorf("回退会话失败")
	}
	c, err = l.svcCtx.ConversationRepo.GetByID(l.ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败")
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
