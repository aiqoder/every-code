package conversation

import (
	"context"
	"strings"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListConversationsLogic {
	return &ListConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListConversationsLogic) ListConversations(req *types.ListConversationsReq) (resp *types.ConversationListResp, err error) {
	if req == nil {
		req = &types.ListConversationsReq{}
	}
	page, pageSize := common.NormalizeListPage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	// 用户侧接口仅返回当前用户会话；传入 clientId 时再按客户端隔离。全量会话走 /admin/conversations。
	channel, err := normalizeChannel(req.Channel)
	if err != nil {
		return nil, err
	}
	list, total, err := l.svcCtx.ConversationRepo.ListByUser(l.ctx, middleware.UserID(l.ctx), strings.TrimSpace(req.ClientId), req.ProjectId, channel, req.TaskId, offset, pageSize)
	if err != nil {
		return nil, err
	}
	vos := make([]common.ConversationVO, 0, len(list))
	for _, c := range list {
		vos = append(vos, *common.ToConversationVO(&c, c.Model))
	}
	return &types.ConversationListResp{
		List:     vos,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
