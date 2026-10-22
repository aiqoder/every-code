package ade

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	adelogic "chengzi-agent-code-server/internal/logic/ade"
	"chengzi-agent-code-server/internal/svc"
)

// StationAgentsHandler GET /api/ade/station-agents 登录用户只读，只含登记目录所需字段。
func StationAgentsHandler(_ *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hresp.WriteOK(w, map[string]any{"list": adelogic.ListStationAgentsForClient()})
	}
}
