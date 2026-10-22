package llmprovider

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/llmprovider"
	"chengzi-agent-code-server/internal/logic/pathutil"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListLlmProvidersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LlmProviderListReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		resp, err := llmprovider.New(r.Context(), svcCtx).List(&req)
		if err != nil {
			hresp.WriteFail(w, "查询供应商列表失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}

func GetLlmProviderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IdPathReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		resp, err := llmprovider.New(r.Context(), svcCtx).Get(&req)
		if err != nil {
			hresp.WriteFail(w, "查询供应商失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}

func CreateLlmProviderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LlmProviderReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		resp, err := llmprovider.New(r.Context(), svcCtx).Create(&req)
		if err != nil {
			hresp.WriteFail(w, "创建供应商失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}

func UpdateLlmProviderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LlmProviderReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		id, err := pathutil.Uint64(r.Context(), "id")
		if err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		resp, err := llmprovider.New(r.Context(), svcCtx).Update(&req, id)
		if err != nil {
			hresp.WriteFail(w, "更新供应商失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}

func DeleteLlmProviderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IdPathReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if err := llmprovider.New(r.Context(), svcCtx).Delete(&req); err != nil {
			hresp.WriteFail(w, "删除供应商失败", err)
			return
		}
		hresp.WriteOK(w, map[string]any{})
	}
}

func ListRemoteModelsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IdPathReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		resp, err := llmprovider.New(r.Context(), svcCtx).ListRemoteModels(&req)
		if err != nil {
			hresp.WriteFail(w, "查询供应商模型列表失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}
