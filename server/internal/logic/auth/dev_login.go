package auth

import (
	"errors"

	"chengzi-agent-code-server/internal/types"
)

// 本机联调登录时固定返回的用户信息。
const (
	devUserID   = uint64(1)
	devRole     = "admin"
	devRolesStr = "admin"
)

// devLogin 在 DevAuth.Enabled 时替代 user-feedback 登录，
// 用户名与密码都不再校验，令牌取配置里的固定值。
func (l *LoginLogic) devLogin() (*types.AuthResp, error) {
	cfg := l.svcCtx.Config.DevAuth
	if !cfg.Enabled {
		return nil, errors.New("dev auth disabled")
	}
	token := cfg.Token
	if token == "" {
		token = "ade-dev-token"
	}
	username := cfg.Username
	if username == "" {
		username = "本地开发者"
	}
	userID := cfg.UserID
	if userID == 0 {
		userID = devUserID
	}
	return &types.AuthResp{
		Token: token,
		User: types.AuthUserVO{
			ID:       userID,
			Username: username,
			Name:     username,
			Role:     devRole,
			Roles:    []string{devRolesStr},
		},
	}, nil
}
