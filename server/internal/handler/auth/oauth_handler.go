package auth

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/auth"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// OAuthAuthorizeHandler 返回 user-feedback 授权页地址。
func OAuthAuthorizeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OAuthAuthorizeReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		l := auth.NewOAuthLogic(r.Context(), svcCtx)
		resp, err := l.AuthorizeURL(&req)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}

// OAuthCallbackHandler 处理 OAuth2 授权码回调。
func OAuthCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OAuthCallbackReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		l := auth.NewOAuthLogic(r.Context(), svcCtx)
		resp, err := l.Callback(&req)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
