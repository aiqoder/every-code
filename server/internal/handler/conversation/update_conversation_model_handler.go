package conversation

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/conversation"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateConversationModelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateConversationModelReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		l := conversation.NewUpdateConversationModelLogic(r.Context(), svcCtx)
		resp, err := l.UpdateConversationModel(&req)
		if err != nil {
			hresp.WriteFail(w, "更新会话模型失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
