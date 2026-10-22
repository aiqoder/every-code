package admin

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/admin"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateAdminToolHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ToolReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		l := admin.NewUpdateAdminToolLogic(r.Context(), svcCtx)
		resp, err := l.UpdateAdminTool(&req)
		if err != nil {
			hresp.WriteFail(w, "更新工具失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
