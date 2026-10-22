package auth

import (
	"context"

	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"chengzi-agent-code-server/pkg/identity"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Login 通过 user-feedback 校验账号密码，成功则透传其令牌。
// 打开 DevAuth 时直接发放本机联调令牌，不再访问身份服务。
func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.AuthResp, err error) {
	if l.svcCtx.Config.DevAuth.Enabled {
		return l.devLogin()
	}
	token, user, err := l.svcCtx.Identity.Login(l.ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &types.AuthResp{Token: token, User: toUserVO(user)}, nil
}

// toUserVO 将身份服务用户转成前端期望的用户视图。
func toUserVO(u *identity.User) types.AuthUserVO {
	if u == nil {
		return types.AuthUserVO{Roles: []string{}}
	}
	roles := u.Roles
	if roles == nil {
		roles = []string{}
	}
	return types.AuthUserVO{
		ID:       u.ID,
		Username: u.Username,
		Name:     u.Username,
		Role:     u.Role(),
		Roles:    roles,
	}
}
