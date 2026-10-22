package agent

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/agent"
	"chengzi-agent-code-server/internal/svc"
)

func ListSkillsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := agent.NewListSkillsLogic(r.Context(), svcCtx)
		resp, err := l.ListSkills(agent.ParseListSkillsReq(r))
		if err != nil {
			hresp.WriteFail(w, "加载技能列表失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
