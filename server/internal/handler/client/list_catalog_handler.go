package client

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	clientlogic "chengzi-agent-code-server/internal/logic/client"
	"chengzi-agent-code-server/internal/svc"
)

func ListCatalogHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := clientlogic.NewListCatalogLogic(r.Context(), svcCtx)
		resp, err := l.ListCatalog()
		if err != nil {
			hresp.WriteFail(w, "读取智能体失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
