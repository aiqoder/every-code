package handler

import (
	"net/http"

	ade "chengzi-agent-code-server/internal/handler/ade"
	"chengzi-agent-code-server/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

// RegisterAdeHandlers 注册 ADE 的目录、项目、任务、技能与部署路由。
func RegisterAdeHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.Auth},
			[]rest.Route{
				{Method: http.MethodGet, Path: "/catalog", Handler: ade.CatalogHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/station-agents", Handler: ade.StationAgentsHandler(serverCtx)},

				{Method: http.MethodGet, Path: "/scenes", Handler: ade.ListScenesHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/scenes", Handler: ade.CreateSceneHandler(serverCtx)},
				{Method: http.MethodPatch, Path: "/scenes/:id", Handler: ade.UpdateSceneHandler(serverCtx)},
				{Method: http.MethodDelete, Path: "/scenes/:id", Handler: ade.DeleteSceneHandler(serverCtx)},

				{Method: http.MethodGet, Path: "/projects", Handler: ade.ListProjectsHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/projects", Handler: ade.CreateProjectHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/projects/:id", Handler: ade.GetProjectHandler(serverCtx)},
				{Method: http.MethodPatch, Path: "/projects/:id", Handler: ade.UpdateProjectHandler(serverCtx)},
				{Method: http.MethodDelete, Path: "/projects/:id", Handler: ade.DeleteProjectHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/projects/:id/env-check", Handler: ade.EnvCheckHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/projects/:id/env-chat", Handler: ade.EnvChatHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/projects/:id/probe", Handler: ade.ProbeHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/projects/:id/tasks", Handler: ade.ListTasksHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/projects/:id/tasks", Handler: ade.CreateTaskHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/projects/:id/deploy-tasks", Handler: ade.ListDeployTasksHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/projects/:id/deploy-tasks", Handler: ade.CreateDeployTaskHandler(serverCtx)},
				{Method: http.MethodPatch, Path: "/deploy-tasks/:id", Handler: ade.UpdateDeployTaskHandler(serverCtx)},
				{Method: http.MethodDelete, Path: "/deploy-tasks/:id", Handler: ade.DeleteDeployTaskHandler(serverCtx)},

				{Method: http.MethodGet, Path: "/tasks/:id", Handler: ade.GetTaskHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/tasks/:id/description", Handler: ade.UpdateDescriptionHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/tasks/:id/plan", Handler: ade.ApplyPlanHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/tasks/:id/start", Handler: ade.StartTaskHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/tasks/:id/retry", Handler: ade.RetryTaskHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/tasks/:id/todos", Handler: ade.AddTodoHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/tasks/:id/events", Handler: ade.TaskEventsHandler(serverCtx)},

				{Method: http.MethodPatch, Path: "/todos/:id", Handler: ade.UpdateTodoHandler(serverCtx)},
				{Method: http.MethodDelete, Path: "/todos/:id", Handler: ade.DeleteTodoHandler(serverCtx)},

				{Method: http.MethodGet, Path: "/skills", Handler: ade.ListSkillsHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/skills/:id/files", Handler: ade.SkillFilesHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/skills/:id/file", Handler: ade.SkillFileHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/skills/:id/download", Handler: ade.DownloadSkillHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/skills/:id/upload", Handler: ade.UploadSkillHandler(serverCtx)},

				{Method: http.MethodGet, Path: "/hosts", Handler: ade.ListHostsHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/hosts", Handler: ade.SaveHostHandler(serverCtx)},
				{Method: http.MethodDelete, Path: "/hosts/:id", Handler: ade.DeleteHostHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/releases", Handler: ade.ReleasesHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/releases", Handler: ade.CreateReleaseHandler(serverCtx)},
				{Method: http.MethodGet, Path: "/audits", Handler: ade.ListAuditsHandler(serverCtx)},
				{Method: http.MethodPost, Path: "/audits", Handler: ade.AuditHandler(serverCtx)},
			}...,
		),
		rest.WithPrefix("/api/ade"),
	)
}
