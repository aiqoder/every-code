package authAuthed

import (
	"context"

	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MeLogic {
	return &MeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Me 返回当前登录用户信息（来自 user-feedback 令牌上下文）。
func (l *MeLogic) Me() (resp *types.UserResp, err error) {
	id := middleware.UserID(l.ctx)
	name := middleware.Name(l.ctx)
	roles := middleware.Roles(l.ctx)
	if roles == nil {
		roles = []string{}
	}
	return &types.UserResp{User: types.AuthUserVO{
		ID:       id,
		Username: name,
		Name:     name,
		Role:     middleware.Role(l.ctx),
		Roles:    roles,
	}}, nil
}
