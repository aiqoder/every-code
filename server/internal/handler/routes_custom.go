package handler

import (
	"net/http"

	ade "chengzi-agent-code-server/internal/handler/ade"
	"chengzi-agent-code-server/internal/handler/brand"
	"chengzi-agent-code-server/internal/handler/conversation"
	custom "chengzi-agent-code-server/internal/handler/custom"
	"chengzi-agent-code-server/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

// RegisterCustomHandlers 注册 SSE 聊天与 WebSocket 等非标准路由。
func RegisterCustomHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/brand",
				Handler: brand.GetBrandHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.Auth},
			[]rest.Route{
				{
					Method:  http.MethodPost,
					Path:    "/conversations/:id/chat",
					Handler: custom.ChatHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/conversations/:id/questions/:questionId",
					Handler: custom.AnswerQuestionHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/clients/ws",
					Handler: custom.WebSocketHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/clients/events",
					Handler: custom.ClientEventsHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/clients/:clientId/rpc",
					Handler: custom.ClientRPCHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/conversations/:id/revert",
					Handler: conversation.RevertConversationHandler(serverCtx),
				},
				{
					Method:  http.MethodPut,
					Path:    "/conversations/:id/model",
					Handler: conversation.UpdateConversationModelHandler(serverCtx),
				},
			}...,
		),
		rest.WithPrefix("/api"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.Auth, serverCtx.AuthAdmin},
			[]rest.Route{
				{
					Method:  http.MethodGet,
					Path:    "/admin/system-skills",
					Handler: ade.AdminListSystemSkillsHandler(serverCtx),
				},
				{
					Method:  http.MethodPut,
					Path:    "/admin/system-skills",
					Handler: ade.AdminSaveSystemSkillsHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/admin/scenes",
					Handler: ade.AdminListScenesHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/admin/scenes",
					Handler: ade.AdminCreateSceneHandler(serverCtx),
				},
				{
					Method:  http.MethodPut,
					Path:    "/admin/scenes/:id",
					Handler: ade.AdminUpdateSceneHandler(serverCtx),
				},
				{
					Method:  http.MethodDelete,
					Path:    "/admin/scenes/:id",
					Handler: ade.AdminDeleteSceneHandler(serverCtx),
				},
			}...,
		),
		rest.WithPrefix("/api"),
	)
}
