package admin

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/admin"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetLlmProviderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IdPathReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		l := admin.NewGetLlmProviderLogic(r.Context(), svcCtx)
		data, err := l.GetLlmProvider(&req)
		if err != nil {
			hresp.WriteFail(w, "查询供应商失败", err)
			return
		}
		hresp.WriteOK(w, data)
	}
}
