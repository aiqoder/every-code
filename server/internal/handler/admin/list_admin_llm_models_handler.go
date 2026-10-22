package admin

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/admin"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListAdminLlmModelsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LlmModelListReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		resp, err := admin.NewListAdminLlmModelsLogic(r.Context(), svcCtx).ListAdminLlmModels(&req)
		if err != nil {
			hresp.WriteFail(w, "查询大模型列表失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
