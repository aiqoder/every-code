package ade

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	adelogic "chengzi-agent-code-server/internal/logic/ade"
	"chengzi-agent-code-server/internal/svc"
)

// AdminListSystemSkillsHandler GET /api/admin/system-skills
func AdminListSystemSkillsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store := skillStore(svcCtx)
		list, err := adelogic.ListSystemSkills(r.Context(), svcCtx, store)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		agents := adelogic.AgentOptions(r.Context(), svcCtx)
		hresp.WriteOK(w, map[string]any{"list": list, "agents": agents})
	}
}

// AdminSaveSystemSkillsHandler PUT /api/admin/system-skills
func AdminSaveSystemSkillsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			List []adelogic.SystemSkillInput `json:"list"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		store := skillStore(svcCtx)
		list, err := adelogic.SaveSystemSkills(r.Context(), svcCtx, store, req.List)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, map[string]any{"list": list})
	}
}
