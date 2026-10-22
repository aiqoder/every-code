package brand

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	brandlogic "chengzi-agent-code-server/internal/logic/brand"
	"chengzi-agent-code-server/internal/svc"
)

// GetBrandHandler 返回品牌名称与 logo，不要求登录。
func GetBrandHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := brandlogic.NewGetBrandLogic(r.Context(), svcCtx)
		hresp.WriteOK(w, l.GetBrand())
	}
}
