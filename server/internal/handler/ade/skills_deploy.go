package ade

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/rest/pathvar"

	"chengzi-agent-code-server/internal/handler/hresp"
	adelogic "chengzi-agent-code-server/internal/logic/ade"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/pkg/timex"
)

func newTicker() *time.Ticker { return time.NewTicker(15 * time.Second) }

// ---------------------------------------------------------------------------
// 系统技能：预览、下载、上传整包替换
// ---------------------------------------------------------------------------

// ListSkillsHandler GET /api/ade/skills
func ListSkillsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store := skillStore(svcCtx)
		list, err := store.List()
		if err != nil {
			hresp.WriteFail(w, "读取技能失败", err)
			return
		}
		assignments, _ := svcCtx.SkillBindingRepo.MapBySkillID(r.Context())
		out := make([]map[string]any, 0, len(list))
		for _, item := range list {
			out = append(out, map[string]any{
				"id":          item.ID,
				"name":        item.Name,
				"description": item.Description,
				"fileCount":   item.FileCount,
				"updatedAt":   item.UpdatedAt,
				"agentId":     assignments[item.ID],
			})
		}
		hresp.WriteOK(w, map[string]any{"list": out})
	}
}

// SkillFilesHandler GET /api/ade/skills/:id/files
func SkillFilesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store := skillStore(svcCtx)
		id := pathvar.Vars(r)["id"]
		files, err := store.Files(id)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, map[string]any{"id": id, "files": files})
	}
}

// SkillFileHandler GET /api/ade/skills/:id/file?path=SKILL.md 只读预览。
func SkillFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store := skillStore(svcCtx)
		id := pathvar.Vars(r)["id"]
		rel := strings.TrimSpace(r.URL.Query().Get("path"))
		content, err := store.ReadFile(id, rel)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, map[string]any{"id": id, "path": rel, "content": content})
	}
}

// DownloadSkillHandler GET /api/ade/skills/:id/download 打成 zip。
func DownloadSkillHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store := skillStore(svcCtx)
		id := pathvar.Vars(r)["id"]
		body, name, err := store.Zip(id)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// UploadSkillHandler POST /api/ade/skills/:id/upload?file=xxx.zip 整包替换。
func UploadSkillHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store := skillStore(svcCtx)
		id := pathvar.Vars(r)["id"]
		file, _, err := r.FormFile("file")
		if err != nil {
			hresp.WriteFail(w, "请选择 zip 文件", err)
			return
		}
		defer file.Close()
		body, err := io.ReadAll(io.LimitReader(file, 32<<20))
		if err != nil {
			hresp.WriteFail(w, "读取上传文件失败", err)
			return
		}
		meta, err := store.Replace(id, body)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		hresp.WriteOK(w, meta)
	}
}

func skillStore(svcCtx *svc.ServiceContext) *adelogic.SkillStore {
	if s, ok := svcCtx.Ade.(*adelogic.Service); ok && s != nil {
		return s.Skills()
	}
	return adelogic.NewSkillStore("skills")
}

// ---------------------------------------------------------------------------
// 部署：档案在服务端，私钥只在本机
// ---------------------------------------------------------------------------

// ListHostsHandler GET /api/ade/hosts
func ListHostsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		list, err := s.Repo().ListHosts(r.Context(), userID(r))
		if err != nil {
			hresp.WriteFail(w, "读取主机失败", err)
			return
		}
		hresp.WriteOK(w, map[string]any{"list": list})
	}
}

// SaveHostHandler POST /api/ade/hosts 新建或更新主机档案。
func SaveHostHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			ID          uint64 `json:"id"`
			Name        string `json:"name"`
			Host        string `json:"host"`
			Port        int    `json:"port"`
			Username    string `json:"username"`
			Fingerprint string `json:"fingerprint"`
			RemoteDir   string `json:"remoteDir"`
			Playbook    string `json:"playbook"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Host) == "" {
			hresp.WriteFail(w, "主机名称与地址不能为空", nil)
			return
		}
		if req.Port <= 0 {
			req.Port = 22
		}
		now := time.Now()
		var h *model.DeployHost
		if req.ID > 0 {
			existing, err := s.Repo().GetHost(r.Context(), req.ID)
			if err != nil || existing.UserID != userID(r) {
				hresp.WriteFail(w, "主机不存在", err)
				return
			}
			h = existing
		} else {
			h = &model.DeployHost{UserID: userID(r), CreatedAt: timex.LocalTime(now)}
		}
		h.Name = strings.TrimSpace(req.Name)
		h.Host = strings.TrimSpace(req.Host)
		h.Port = req.Port
		h.Username = strings.TrimSpace(req.Username)
		if req.Fingerprint != "" {
			h.Fingerprint = strings.TrimSpace(req.Fingerprint)
		}
		h.RemoteDir = strings.TrimSpace(req.RemoteDir)
		if req.Playbook != "" || req.ID == 0 {
			h.Playbook = req.Playbook
			if strings.TrimSpace(h.Playbook) == "" {
				h.Playbook = adelogic.DefaultPlaybook
			}
		}
		h.UpdatedAt = timex.LocalTime(now)
		if req.ID > 0 {
			err = s.Repo().UpdateHost(r.Context(), h)
		} else {
			err = s.Repo().CreateHost(r.Context(), h)
		}
		if err != nil {
			hresp.WriteFail(w, "保存主机失败", err)
			return
		}
		hresp.WriteOK(w, h)
	}
}

// DeleteHostHandler DELETE /api/ade/hosts/:id
func DeleteHostHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			hresp.WriteFail(w, "无效的主机 ID", err)
			return
		}
		h, err := s.Repo().GetHost(r.Context(), id)
		if err != nil || h.UserID != userID(r) {
			hresp.WriteFail(w, "主机不存在", err)
			return
		}
		if err := s.Repo().DeleteHost(r.Context(), id); err != nil {
			hresp.WriteFail(w, "删除主机失败", err)
			return
		}
		hresp.WriteOK(w, map[string]any{"id": id})
	}
}

// ReleasesHandler GET /api/ade/releases
func ReleasesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		list, err := s.Repo().ListReleases(r.Context(), userID(r), 80)
		if err != nil {
			hresp.WriteFail(w, "读取发布记录失败", err)
			return
		}
		hresp.WriteOK(w, map[string]any{"list": list})
	}
}

// CreateReleaseHandler POST /api/ade/releases 发布与回滚记录回写。
func CreateReleaseHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			TaskID   uint64 `json:"taskId"`
			HostID   uint64 `json:"hostId"`
			Version  string `json:"version"`
			Status   string `json:"status"`
			Detail   string `json:"detail"`
			Previous string `json:"previous"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		rel := &model.Release{
			UserID:    userID(r),
			TaskID:    req.TaskID,
			HostID:    req.HostID,
			Version:   strings.TrimSpace(req.Version),
			Status:    strings.TrimSpace(req.Status),
			Detail:    req.Detail,
			Previous:  strings.TrimSpace(req.Previous),
			CreatedAt: timex.LocalTime(time.Now()),
		}
		if err := s.Repo().CreateRelease(r.Context(), rel); err != nil {
			hresp.WriteFail(w, "写入发布记录失败", err)
			return
		}
		hresp.WriteOK(w, rel)
	}
}

// ListAuditsHandler GET /api/ade/audits 命令审计列表。
func ListAuditsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		q := r.URL.Query()
		hostID, _ := strconv.ParseUint(q.Get("hostId"), 10, 64)
		taskID, _ := strconv.ParseUint(q.Get("taskId"), 10, 64)
		limit, _ := strconv.Atoi(q.Get("limit"))
		list, err := s.Repo().ListCommandAudits(r.Context(), userID(r), hostID, taskID, limit)
		if err != nil {
			hresp.WriteFail(w, "读取审计失败", err)
			return
		}
		hresp.WriteOK(w, map[string]any{"list": list})
	}
}

// AuditHandler POST /api/ade/audits 远程命令审计回写。
func AuditHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			TaskID   uint64 `json:"taskId"`
			HostID   uint64 `json:"hostId"`
			Command  string `json:"command"`
			ExitCode int    `json:"exitCode"`
			Output   string `json:"output"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		audit := &model.CommandAudit{
			UserID:    userID(r),
			TaskID:    req.TaskID,
			HostID:    req.HostID,
			Command:   req.Command,
			ExitCode:  req.ExitCode,
			Output:    req.Output,
			CreatedAt: timex.LocalTime(time.Now()),
		}
		if err := s.Repo().AddCommandAudit(r.Context(), audit); err != nil {
			hresp.WriteFail(w, "写入审计失败", err)
			return
		}
		hresp.WriteOK(w, audit)
	}
}

// ListDeployTasksHandler GET /api/ade/projects/:id/deploy-tasks
func ListDeployTasksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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
		list, err := s.Repo().ListDeployTasks(r.Context(), userID(r), p.ID)
		if err != nil {
			hresp.WriteFail(w, "读取部署任务失败", err)
			return
		}
		if list == nil {
			list = []model.DeployTask{}
		}
		hresp.WriteOK(w, map[string]any{"list": list})
	}
}

// CreateDeployTaskHandler POST /api/ade/projects/:id/deploy-tasks
func CreateDeployTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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
			Title  string `json:"title"`
			Prompt string `json:"prompt"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		title := strings.TrimSpace(req.Title)
		prompt := strings.TrimSpace(req.Prompt)
		if title == "" || prompt == "" {
			hresp.WriteFail(w, "标题和说明都不能空", nil)
			return
		}
		now := time.Now()
		task := &model.DeployTask{
			UserID:    userID(r),
			ProjectID: p.ID,
			Title:     title,
			Prompt:    prompt,
			CreatedAt: timex.LocalTime(now),
			UpdatedAt: timex.LocalTime(now),
		}
		if err := s.Repo().CreateDeployTask(r.Context(), task); err != nil {
			hresp.WriteFail(w, "创建部署任务失败", err)
			return
		}
		hresp.WriteOK(w, task)
	}
}

// UpdateDeployTaskHandler PATCH /api/ade/deploy-tasks/:id
func UpdateDeployTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			hresp.WriteFail(w, "无效的部署任务", err)
			return
		}
		task, err := s.Repo().GetDeployTask(r.Context(), id)
		if err != nil || task.UserID != userID(r) {
			hresp.WriteFail(w, "部署任务不存在", err)
			return
		}
		var req struct {
			Title  string `json:"title"`
			Prompt string `json:"prompt"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		title := strings.TrimSpace(req.Title)
		prompt := strings.TrimSpace(req.Prompt)
		if title == "" || prompt == "" {
			hresp.WriteFail(w, "标题和说明都不能空", nil)
			return
		}
		task.Title = title
		task.Prompt = prompt
		task.UpdatedAt = timex.LocalTime(time.Now())
		if err := s.Repo().UpdateDeployTask(r.Context(), task); err != nil {
			hresp.WriteFail(w, "保存部署任务失败", err)
			return
		}
		hresp.WriteOK(w, task)
	}
}

// DeleteDeployTaskHandler DELETE /api/ade/deploy-tasks/:id
func DeleteDeployTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			hresp.WriteFail(w, "无效的部署任务", err)
			return
		}
		task, err := s.Repo().GetDeployTask(r.Context(), id)
		if err != nil || task.UserID != userID(r) {
			hresp.WriteFail(w, "部署任务不存在", err)
			return
		}
		if err := s.Repo().DeleteDeployTask(r.Context(), id); err != nil {
			hresp.WriteFail(w, "删除部署任务失败", err)
			return
		}
		hresp.WriteOK(w, map[string]any{"id": id})
	}
}
