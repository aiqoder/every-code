package conversation

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/logic/llmprovider"
	"chengzi-agent-code-server/internal/logic/pathutil"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateConversationModelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateConversationModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateConversationModelLogic {
	return &UpdateConversationModelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateConversationModelLogic) UpdateConversationModel(req *types.UpdateConversationModelReq) (*common.ConversationVO, error) {
	id, err := pathutil.Uint64(l.ctx, "id")
	if err != nil {
		return nil, err
	}
	if req.ModelId == 0 {
		return nil, fmt.Errorf("请选择模型")
	}
	m, err := l.svcCtx.LlmModelRepo.GetByID(l.ctx, req.ModelId)
	if err != nil {
		return nil, fmt.Errorf("模型不存在")
	}
	userID := middleware.UserID(l.ctx)
	if !llmprovider.CanAccessModel(m, userID) {
		return nil, fmt.Errorf("模型不存在或无权使用")
	}
	c, err := l.svcCtx.ConversationRepo.GetByID(l.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("会话不存在")
	}
	if c.UserID != userID {
		return nil, fmt.Errorf("无权访问该会话")
	}
	if c.ModelID != req.ModelId {
		if err := l.svcCtx.ConversationRepo.UpdateModelID(l.ctx, id, req.ModelId); err != nil {
			return nil, err
		}
		c, err = l.svcCtx.ConversationRepo.GetByID(l.ctx, id)
		if err != nil {
			return nil, err
		}
	}
	return common.ToConversationVO(c, c.Model), nil
}
