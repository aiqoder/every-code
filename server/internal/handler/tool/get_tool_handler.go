package tool

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/tool"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetToolHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IdPathReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		l := tool.NewGetToolLogic(r.Context(), svcCtx)
		resp, err := l.GetTool(&req)
		if err != nil {
			hresp.WriteFail(w, "查询工具失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
