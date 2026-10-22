package apiKey

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/apiKey"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func RevealApiKeyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IdPathReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		l := apiKey.NewRevealApiKeyLogic(r.Context(), svcCtx)
		resp, err := l.RevealApiKey(&req)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
