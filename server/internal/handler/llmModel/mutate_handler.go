package llmModel

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/llmModel"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreateLlmModelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LlmModelReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		resp, err := llmModel.NewMutateLogic(r.Context(), svcCtx).Create(&req)
		if err != nil {
			hresp.WriteFail(w, "创建大模型失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}

func UpdateLlmModelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LlmModelReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		resp, err := llmModel.NewMutateLogic(r.Context(), svcCtx).Update(&req)
		if err != nil {
			hresp.WriteFail(w, "更新大模型失败", err)
			return
		}
		hresp.WriteOK(w, resp)
	}
}

func DeleteLlmModelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IdPathReq
		if err := httpx.Parse(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if err := llmModel.NewMutateLogic(r.Context(), svcCtx).Delete(&req); err != nil {
			hresp.WriteFail(w, "删除大模型失败", err)
			return
		}
		hresp.WriteOK(w, map[string]any{})
	}
}
