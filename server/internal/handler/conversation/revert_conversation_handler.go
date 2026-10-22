package conversation

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/conversation"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func RevertConversationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RevertConversationReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		l := conversation.NewRevertConversationLogic(r.Context(), svcCtx)
		data, err := l.RevertConversation(&req)
		if err != nil {
			hresp.WriteFail(w, "回退会话失败", err)
			return
		}
		hresp.WriteOK(w, data)
	}
}
