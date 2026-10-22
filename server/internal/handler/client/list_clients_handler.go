package client

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/client"
	"chengzi-agent-code-server/internal/svc"
)

func ListClientsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := client.NewListClientsLogic(r.Context(), svcCtx)
		resp, err := l.ListClients()
		if err != nil {
			hresp.WriteFail(w, "获取客户端列表失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
