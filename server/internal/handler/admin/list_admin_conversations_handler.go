package admin

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/admin"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListAdminConversationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PageReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		l := admin.NewListAdminConversationsLogic(r.Context(), svcCtx)
		resp, err := l.ListAdminConversations(&req)
		if err != nil {
			hresp.WriteFail(w, "获取会话列表失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
