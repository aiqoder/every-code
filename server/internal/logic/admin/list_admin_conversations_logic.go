package admin

import (
	"context"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAdminConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAdminConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAdminConversationsLogic {
	return &ListAdminConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAdminConversationsLogic) ListAdminConversations(req *types.PageReq) (*types.ConversationListResp, error) {
	page, pageSize := common.NormalizeListPage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	list, total, err := l.svcCtx.ConversationRepo.List(l.ctx, offset, pageSize)
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
