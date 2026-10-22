package client

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	clientlogic "chengzi-agent-code-server/internal/logic/client"
	"chengzi-agent-code-server/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateCatalogHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req clientlogic.UpdateCatalogReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		l := clientlogic.NewUpdateCatalogLogic(r.Context(), svcCtx)
		resp, err := l.UpdateCatalog(&req)
		if err != nil {
			hresp.WriteFail(w, "注册 tools 失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
