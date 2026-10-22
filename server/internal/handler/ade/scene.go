package ade

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/svc"
)

type sceneBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ListScenesHandler GET /api/ade/scenes
func ListScenesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		list, err := s.ListScenes(r.Context(), userID(r))
		if err != nil {
			hresp.WriteFail(w, "读取场景失败", err)
			return
		}
		hresp.WriteOK(w, map[string]any{"list": list})
	}
}

// CreateSceneHandler POST /api/ade/scenes
func CreateSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req sceneBody
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		sc, err := s.CreateUserScene(r.Context(), userID(r), req.Title, req.Description)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, sc)
	}
}

// UpdateSceneHandler PATCH /api/ade/scenes/:id
func UpdateSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		var req sceneBody
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		sc, err := s.UpdateUserScene(r.Context(), userID(r), id, req.Title, req.Description)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, sc)
	}
}

// DeleteSceneHandler DELETE /api/ade/scenes/:id
func DeleteSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if err := s.DeleteUserScene(r.Context(), userID(r), id); err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, map[string]any{"id": id})
	}
}

// AdminListScenesHandler GET /api/admin/scenes
func AdminListScenesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		list, err := s.ListAdminScenes(r.Context())
		if err != nil {
			hresp.WriteFail(w, "读取场景失败", err)
			return
		}
		hresp.WriteOK(w, map[string]any{"list": list})
	}
}

// AdminCreateSceneHandler POST /api/admin/scenes
func AdminCreateSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req sceneBody
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		sc, err := s.CreateAdminScene(r.Context(), req.Title, req.Description)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, sc)
	}
}

// AdminUpdateSceneHandler PUT /api/admin/scenes/:id
func AdminUpdateSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		var req sceneBody
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		sc, err := s.UpdateAdminScene(r.Context(), id, req.Title, req.Description)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, sc)
	}
}

// AdminDeleteSceneHandler DELETE /api/admin/scenes/:id
func AdminDeleteSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if err := s.DeleteAdminScene(r.Context(), id); err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, map[string]any{"id": id})
	}
}
