package authAuthed

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/authAuthed"
	"chengzi-agent-code-server/internal/svc"
)

func MeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := authAuthed.NewMeLogic(r.Context(), svcCtx)
		resp, err := l.Me()
		if err != nil {
			hresp.WriteFail(w, "获取用户信息失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
