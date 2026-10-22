package apiKey

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/apiKey"
	"chengzi-agent-code-server/internal/svc"
)

func ListApiKeysHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := apiKey.NewListApiKeysLogic(r.Context(), svcCtx)
		resp, err := l.ListApiKeys()
		if err != nil {
			hresp.WriteFail(w, "获取 API Key 失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
