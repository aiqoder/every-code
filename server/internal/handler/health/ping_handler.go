package health

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/health"
	"chengzi-agent-code-server/internal/svc"
)

func PingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := health.NewPingLogic(r.Context(), svcCtx)
		resp, err := l.Ping()
		if err != nil {
			hresp.WriteFail(w, "服务不可用", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
