package ade

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest/pathvar"

	"chengzi-agent-code-server/internal/handler/hresp"
	adelogic "chengzi-agent-code-server/internal/logic/ade"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/pkg/response"
)

// Service 从 svc 取 ADE 服务；未初始化时给出可读错误。
func getService(svcCtx *svc.ServiceContext) (*adelogic.Service, error) {
	s, ok := svcCtx.Ade.(*adelogic.Service)
	if !ok || s == nil {
		return nil, fmt.Errorf("ADE 服务未就绪")
	}
	return s, nil
}

func decodePipelineAgents(r *http.Request) (adelogic.PipelineAgents, error) {
	var req struct {
		Coder    string `json:"coder"`
		Verifier string `json:"verifier"`
		Reviewer string `json:"reviewer"`
	}
	if err := decode(r, &req); err != nil {
		return adelogic.PipelineAgents{}, err
	}
	return adelogic.PipelineAgents{Coder: req.Coder, Verifier: req.Verifier, Reviewer: req.Reviewer}, nil
}

func decode(r *http.Request, dest any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}
	return json.Unmarshal(body, dest)
}

func pathID(r *http.Request, name string) (uint64, error) {
	raw := pathvar.Vars(r)[name]
	if raw == "" {
		return 0, fmt.Errorf("缺少 %s", name)
	}
	return strconv.ParseUint(raw, 10, 64)
}

func userID(r *http.Request) uint64 { return middleware.UserID(r.Context()) }

// ---------------------------------------------------------------------------
// 目录
// ---------------------------------------------------------------------------

// CatalogHandler GET /api/ade/catalog 技术清单、场景与全部智能体。
func CatalogHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows := append(adelogic.ListStationAgents(), adelogic.ListPipelineAgents()...)
		listed := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			listed = append(listed, map[string]string{
				"id": row.AgentID, "name": row.Name, "description": row.Description,
			})
		}
		hresp.WriteOK(w, map[string]any{
			"technologies": adelogic.TechnologyCatalog(),
			"scenarios":    adelogic.ScenarioCatalog(),
			"agents":       listed,
		})
	}
}

// ---------------------------------------------------------------------------
// 项目
// ---------------------------------------------------------------------------

// ListProjectsHandler GET /api/ade/projects
func ListProjectsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		list, err := s.Repo().ListProjects(r.Context(), userID(r))
		if err != nil {
			hresp.WriteFail(w, "读取项目失败", err)
			return
		}
		vos := make([]*adelogic.ProjectVO, 0, len(list))
		for i := range list {
			vos = append(vos, s.ToProjectVO(r.Context(), &list[i]))
		}
		hresp.WriteOK(w, map[string]any{"list": vos})
	}
}

// CreateProjectHandler POST /api/ade/projects
func CreateProjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			LocalDir     string   `json:"localDir"`
			Scenario     string   `json:"scenario"`
			Technologies []string `json:"technologies"`
			ModelID      uint64   `json:"modelId"`
			SceneID      uint64   `json:"sceneId"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		p, err := s.CreateProject(r.Context(), userID(r), req.Name, req.Description, req.LocalDir, req.Scenario,
			req.Technologies, req.ModelID, req.SceneID)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, s.ToProjectVO(r.Context(), p))
	}
}

func loadProject(r *http.Request, svcCtx *svc.ServiceContext, s *adelogic.Service) (*model.Project, error) {
	id, err := pathID(r, "id")
	if err != nil {
		return nil, err
	}
	p, err := s.Repo().GetProject(r.Context(), id)
	if err != nil {
		return nil, fmt.Errorf("项目不存在")
	}
	if p.UserID != userID(r) {
		return nil, fmt.Errorf("无权访问该项目")
	}
	return p, nil
}

// UpdateProjectHandler PATCH /api/ade/projects/:id
func UpdateProjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		p, err := loadProject(r, svcCtx, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			Name         *string  `json:"name"`
			Description  *string  `json:"description"`
			LocalDir     *string  `json:"localDir"`
			Scenario     *string  `json:"scenario"`
			Technologies []string `json:"technologies"`
			ModelID      *uint64  `json:"modelId"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		in := adelogic.UpdateProjectInput{
			Name:         req.Name,
			Description:  req.Description,
			LocalDir:     req.LocalDir,
			Scenario:     req.Scenario,
			Technologies: req.Technologies,
			TechsSet:     req.Technologies != nil,
			ModelID:      req.ModelID,
		}
		updated, err := s.UpdateProject(r.Context(), p, in)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, s.ToProjectVO(r.Context(), updated))
	}
}

// GetProjectHandler GET /api/ade/projects/:id
func GetProjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		p, err := loadProject(r, svcCtx, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, s.ToProjectVO(r.Context(), p))
	}
}

// DeleteProjectHandler DELETE /api/ade/projects/:id
func DeleteProjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		p, err := loadProject(r, svcCtx, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		if err := s.Repo().DeleteProject(r.Context(), p.ID); err != nil {
			hresp.WriteFail(w, "删除项目失败", err)
			return
		}
		hresp.WriteOK(w, map[string]any{"id": p.ID})
	}
}

// EnvCheckHandler POST /api/ade/projects/:id/env-check 保存本机检测结果。
func EnvCheckHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		p, err := loadProject(r, svcCtx, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			Items []adelogic.EnvCheck `json:"items"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if err := s.SaveEnvCheck(r.Context(), p, req.Items); err != nil {
			hresp.WriteFail(w, "保存检测结果失败", err)
			return
		}
		hresp.WriteOK(w, s.ToProjectVO(r.Context(), p))
	}
}

// ProbeHandler GET /api/ade/projects/:id/probe 返回该项目的固定探测命令。
func ProbeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		p, err := loadProject(r, svcCtx, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, map[string]any{
			"items": adelogic.ProbeCommands(adelogic.SplitTechs(p.Technologies)),
		})
	}
}

// ---------------------------------------------------------------------------
// 任务
// ---------------------------------------------------------------------------

// ListTasksHandler GET /api/ade/projects/:id/tasks
func ListTasksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		p, err := loadProject(r, svcCtx, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		list, err := s.Repo().ListTasksByProject(r.Context(), p.ID)
		if err != nil {
			hresp.WriteFail(w, "读取任务失败", err)
			return
		}
		vos := make([]*adelogic.TaskVO, 0, len(list))
		for i := range list {
			vos = append(vos, s.ToTaskVO(r.Context(), &list[i], p.Name))
		}
		hresp.WriteOK(w, map[string]any{"list": vos})
	}
}

// CreateTaskHandler POST /api/ade/projects/:id/tasks
func CreateTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		p, err := loadProject(r, svcCtx, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			Description string                   `json:"description"`
			Name        string                   `json:"name"`
			Overview    string                   `json:"overview"`
			Plan        string                   `json:"plan"`
			Todos       []adelogic.TaskTodoInput `json:"todos"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		// clientId 记在会话表；建任务时不传，等挂任务的会话创建后再回填。
		t, err := s.CreateTask(r.Context(), userID(r), p.ID, req.Description, "", adelogic.TaskPlanInput{
			Name:     req.Name,
			Overview: req.Overview,
			Plan:     req.Plan,
			Todos:    req.Todos,
		})
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, s.ToTaskVO(r.Context(), t, p.Name))
	}
}

func loadTask(r *http.Request, s *adelogic.Service) (*model.Task, *model.Project, error) {
	id, err := pathID(r, "id")
	if err != nil {
		return nil, nil, err
	}
	t, err := s.Repo().GetTask(r.Context(), id)
	if err != nil {
		return nil, nil, fmt.Errorf("任务不存在")
	}
	if t.UserID != userID(r) {
		return nil, nil, fmt.Errorf("无权访问该任务")
	}
	p, err := s.Repo().GetProject(r.Context(), t.ProjectID)
	if err != nil {
		return nil, nil, fmt.Errorf("项目不存在")
	}
	return t, p, nil
}

// GetTaskHandler GET /api/ade/tasks/:id
func GetTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		t, p, err := loadTask(r, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, s.ToTaskVO(r.Context(), t, p.Name))
	}
}

// ApplyPlanHandler POST /api/ade/tasks/:id/plan
func ApplyPlanHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		t, p, err := loadTask(r, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			Name     string                   `json:"name"`
			Overview string                   `json:"overview"`
			Plan     string                   `json:"plan"`
			Todos    []adelogic.TaskTodoInput `json:"todos"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if err := s.ApplyTaskPlan(r.Context(), t, adelogic.TaskPlanInput{
			Name:     req.Name,
			Overview: req.Overview,
			Plan:     req.Plan,
			Todos:    req.Todos,
		}); err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, s.ToTaskVO(r.Context(), t, p.Name))
	}
}

// UpdateDescriptionHandler POST /api/ade/tasks/:id/description
func UpdateDescriptionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		t, p, err := loadTask(r, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			Description string `json:"description"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if err := s.UpdateDescription(r.Context(), t, req.Description); err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, s.ToTaskVO(r.Context(), t, p.Name))
	}
}

// StartTaskHandler POST /api/ade/tasks/:id/start
func StartTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		t, p, err := loadTask(r, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		agents, err := decodePipelineAgents(r)
		if err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if err := s.StartTask(r.Context(), t, agents); err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, s.ToTaskVO(r.Context(), t, p.Name))
	}
}

// RetryTaskHandler POST /api/ade/tasks/:id/retry
func RetryTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		t, p, err := loadTask(r, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		agents, err := decodePipelineAgents(r)
		if err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if err := s.RetryTask(r.Context(), t, agents); err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, s.ToTaskVO(r.Context(), t, p.Name))
	}
}

// AddTodoHandler POST /api/ade/tasks/:id/todos 用户追加待办。
func AddTodoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		t, _, err := loadTask(r, s)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			Title      string `json:"title"`
			Detail     string `json:"detail"`
			Acceptance string `json:"acceptance"`
			DependsOn  string `json:"dependsOn"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		td, err := s.AddUserTodo(r.Context(), t.ID, req.Title, req.Detail, req.Acceptance, req.DependsOn)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, td)
	}
}

// UpdateTodoHandler PATCH /api/ade/todos/:id 只能改用户自己加的。
func UpdateTodoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			hresp.WriteFail(w, "无效的待办 ID", err)
			return
		}
		td, err := s.Repo().GetTodo(r.Context(), id)
		if err != nil {
			hresp.WriteFail(w, "待办不存在", err)
			return
		}
		if td.Source != model.TodoSourceUser {
			hresp.WriteFail(w, "Agent 生成的待办不可修改", errors.New("locked"))
			return
		}
		var req struct {
			Title      string `json:"title"`
			Detail     string `json:"detail"`
			Acceptance string `json:"acceptance"`
			Status     string `json:"status"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if err := s.UpdateUserTodo(r.Context(), td, req.Title, req.Detail, req.Acceptance, req.Status); err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, td)
	}
}

// DeleteTodoHandler DELETE /api/ade/todos/:id 只能删用户自己加的。
func DeleteTodoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			hresp.WriteFail(w, "无效的待办 ID", err)
			return
		}
		td, err := s.Repo().GetTodo(r.Context(), id)
		if err != nil {
			hresp.WriteFail(w, "待办不存在", err)
			return
		}
		if err := s.DeleteUserTodo(r.Context(), td); err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, map[string]any{"id": td.ID})
	}
}

// TaskEventsHandler GET /api/ade/tasks/:id/events SSE 订阅运行事件，先回放再实时。
func TaskEventsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			response.WriteFail(w, err.Error(), err.Error())
			return
		}
		t, _, err := loadTask(r, s)
		if err != nil {
			response.WriteFail(w, err.Error(), err.Error())
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			response.WriteFail(w, "当前连接不支持流式响应", "")
			return
		}

		// 先订阅再回放，避免回放期间丢帧。
		events, recent, cancel := s.Bus().Subscribe(t.ID)
		defer cancel()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		writeEvent := func(ev adelogic.Event) error {
			raw, _ := json.Marshal(ev)
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, raw); err != nil {
				return err
			}
			flusher.Flush()
			return nil
		}

		for _, ev := range recent {
			if err := writeEvent(ev); err != nil {
				return
			}
		}

		ticker := newTicker()
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case ev, ok := <-events:
				if !ok {
					return
				}
				if err := writeEvent(ev); err != nil {
					return
				}
			case <-ticker.C:
				if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}
