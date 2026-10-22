package tokenusage

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	logic "chengzi-agent-code-server/internal/logic/tokenusage"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetTokenUsageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TokenUsageStatsReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		l := logic.NewGetTokenUsageLogic(r.Context(), svcCtx)
		resp, err := l.GetTokenUsage(&req)
		if err != nil {
			hresp.WriteFail(w, "查询用量统计失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
