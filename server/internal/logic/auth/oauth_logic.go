package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type OAuthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewOAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OAuthLogic {
	return &OAuthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// AuthorizeURL 生成 user-feedback 授权地址（OAuth2 授权码模式）。
func (l *OAuthLogic) AuthorizeURL(req *types.OAuthAuthorizeReq) (*types.OAuthAuthorizeResp, error) {
	if !l.svcCtx.Identity.Configured() {
		return nil, fmt.Errorf("未配置 user-feedback 服务地址")
	}
	state := req.State
	if state == "" {
		state = randomState()
	}
	return &types.OAuthAuthorizeResp{
		Url:   l.svcCtx.Identity.AuthorizeURL(state),
		State: state,
	}, nil
}

// Callback 处理授权码回调：换取访问令牌并透传，同时返回本地落地页地址。
//
// 同一个授权码被重复回调时（刷新、后退、重复点击），直接复用首次兑换结果，
// 避免再次向 user-feedback 提交一次性授权码而报 invalid_grant。
func (l *OAuthLogic) Callback(req *types.OAuthCallbackReq) (*types.OAuthCallbackResp, error) {
	if raw, ok := l.svcCtx.OAuthCodeCache.Get(req.Code); ok {
		if cached, valid := raw.(*types.OAuthCallbackResp); valid {
			l.Infow("oauth callback 命中授权码缓存，直接复用首次兑换结果",
				logx.Field("code", maskCode(req.Code)),
				logx.Field("username", cached.User.Username))
			return cached, nil
		}
	}

	token, user, err := l.svcCtx.Identity.ExchangeCode(l.ctx, req.Code)
	if err != nil {
		return nil, err
	}
	resp := &types.OAuthCallbackResp{
		Token:    token,
		User:     toUserVO(user),
		Redirect: l.svcCtx.Config.UserFeedback.RedirectURI,
	}
	l.svcCtx.OAuthCodeCache.Set(req.Code, resp)
	return resp, nil
}

// maskCode 授权码脱敏，避免完整凭证写入日志。
func maskCode(code string) string {
	if len(code) <= 12 {
		return "***"
	}
	return code[:12] + "***"
}

// randomState 生成随机的 OAuth2 state（防 CSRF）。
func randomState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
