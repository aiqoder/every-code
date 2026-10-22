package apiKey

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/apiKey"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func DeleteApiKeyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IdPathReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		l := apiKey.NewDeleteApiKeyLogic(r.Context(), svcCtx)
		if err := l.DeleteApiKey(&req); err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOKMsg(w, "已删除", nil)
	}
}
