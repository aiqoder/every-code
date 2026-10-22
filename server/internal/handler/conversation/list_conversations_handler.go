package conversation

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/conversation"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListConversationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListConversationsReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		l := conversation.NewListConversationsLogic(r.Context(), svcCtx)
		resp, err := l.ListConversations(&req)
		if err != nil {
			hresp.WriteFail(w, "获取会话列表失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
