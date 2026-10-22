package llmModel

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/llmModel"
	"chengzi-agent-code-server/internal/svc"
)

func InputCapabilitiesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := llmModel.NewInputCapabilitiesLogic(r.Context(), svcCtx)
		resp, err := l.InputCapabilities()
		if err != nil {
			hresp.WriteFail(w, "获取输入能力列表失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
