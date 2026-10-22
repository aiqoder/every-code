package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
)

// AdeRepository ADE 项目、任务、待办、运行与部署的数据访问。
type AdeRepository interface {
	CreateProject(ctx context.Context, p *model.Project) error
	UpdateProject(ctx context.Context, p *model.Project) error
	DeleteProject(ctx context.Context, id uint64) error
	GetProject(ctx context.Context, id uint64) (*model.Project, error)
	ListProjects(ctx context.Context, userID uint64) ([]model.Project, error)

	CreateTask(ctx context.Context, t *model.Task) error
	UpdateTask(ctx context.Context, t *model.Task) error
	GetTask(ctx context.Context, id uint64) (*model.Task, error)
	ListTasksByProject(ctx context.Context, projectID uint64) ([]model.Task, error)
	ListTasksByStatus(ctx context.Context, statuses ...string) ([]model.Task, error)
	// ClaimTask 原子地领取执行中 / 重新执行的任务，避免同一任务被两次调度。
	ClaimTask(ctx context.Context, id uint64, from string) (bool, error)

	CreateTodo(ctx context.Context, t *model.Todo) error
	// SaveTaskPlan 保存任务计划，并替换这条任务上智能体生成的待办。用户自己加的待办留着。
	SaveTaskPlan(ctx context.Context, t *model.Task, todos []*model.Todo) error
	UpdateTodo(ctx context.Context, t *model.Todo) error
	DeleteTodo(ctx context.Context, id uint64) error
	GetTodo(ctx context.Context, id uint64) (*model.Todo, error)
	ListTodos(ctx context.Context, taskID uint64) ([]model.Todo, error)
	ResetTodos(ctx context.Context, taskID uint64) error

	CreateRun(ctx context.Context, r *model.Run) error
	UpdateRun(ctx context.Context, r *model.Run) error
	GetRun(ctx context.Context, id uint64) (*model.Run, error)
	ListRuns(ctx context.Context, userID uint64, limit int) ([]model.Run, error)
	ListRunsByTask(ctx context.Context, taskID uint64) ([]model.Run, error)

	AppendRunEvent(ctx context.Context, e *model.RunEvent) error
	ListRunEvents(ctx context.Context, runID uint64, afterSeq int) ([]model.RunEvent, error)

	CreateHost(ctx context.Context, h *model.DeployHost) error
	UpdateHost(ctx context.Context, h *model.DeployHost) error
	DeleteHost(ctx context.Context, id uint64) error
	GetHost(ctx context.Context, id uint64) (*model.DeployHost, error)
	ListHosts(ctx context.Context, userID uint64) ([]model.DeployHost, error)

	CreateDeployTask(ctx context.Context, t *model.DeployTask) error
	UpdateDeployTask(ctx context.Context, t *model.DeployTask) error
	DeleteDeployTask(ctx context.Context, id uint64) error
	GetDeployTask(ctx context.Context, id uint64) (*model.DeployTask, error)
	ListDeployTasks(ctx context.Context, userID, projectID uint64) ([]model.DeployTask, error)

	CreateRelease(ctx context.Context, r *model.Release) error
	ListReleases(ctx context.Context, userID uint64, limit int) ([]model.Release, error)
	LatestReleases(ctx context.Context, hostID uint64, limit int) ([]model.Release, error)

	AddCommandAudit(ctx context.Context, a *model.CommandAudit) error
	ListCommandAudits(ctx context.Context, userID uint64, hostID, taskID uint64, limit int) ([]model.CommandAudit, error)

	CreateScene(ctx context.Context, sc *model.Scene) error
	UpdateScene(ctx context.Context, sc *model.Scene) error
	DeleteScene(ctx context.Context, id uint64) error
	GetScene(ctx context.Context, id uint64) (*model.Scene, error)
	ListVisibleScenes(ctx context.Context, userID uint64) ([]model.Scene, error)
	ListAdminScenes(ctx context.Context) ([]model.Scene, error)
}

type adeRepository struct {
	db *gorm.DB
}

// NewAdeRepository 构造 ADE 仓储。
func NewAdeRepository(db *gorm.DB) AdeRepository {
	return &adeRepository{db: db}
}

func (r *adeRepository) CreateProject(ctx context.Context, p *model.Project) error {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

func (r *adeRepository) UpdateProject(ctx context.Context, p *model.Project) error {
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

func (r *adeRepository) DeleteProject(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", id).Delete(&model.Task{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", id).Delete(&model.DeployTask{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", id).Delete(&model.ConversationLink{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Project{}, id).Error
	})
}

func (r *adeRepository) GetProject(ctx context.Context, id uint64) (*model.Project, error) {
	var p model.Project
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &p, nil
}

func (r *adeRepository) ListProjects(ctx context.Context, userID uint64) ([]model.Project, error) {
	var list []model.Project
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id desc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return list, nil
}

func (r *adeRepository) CreateTask(ctx context.Context, t *model.Task) error {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (r *adeRepository) UpdateTask(ctx context.Context, t *model.Task) error {
	if err := r.db.WithContext(ctx).Save(t).Error; err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func (r *adeRepository) GetTask(ctx context.Context, id uint64) (*model.Task, error) {
	var t model.Task
	if err := r.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return &t, nil
}

func (r *adeRepository) ListTasksByProject(ctx context.Context, projectID uint64) ([]model.Task, error) {
	var list []model.Task
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("id desc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return list, nil
}

func (r *adeRepository) ListTasksByStatus(ctx context.Context, statuses ...string) ([]model.Task, error) {
	var list []model.Task
	if err := r.db.WithContext(ctx).
		Where("status IN ?", statuses).
		Order("id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list tasks by status: %w", err)
	}
	return list, nil
}

func (r *adeRepository) ClaimTask(ctx context.Context, id uint64, from string) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.Task{}).
		Where("id = ? AND status = ?", id, from).
		Update("status", model.TaskStatusRunning)
	if res.Error != nil {
		return false, fmt.Errorf("claim task: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *adeRepository) CreateTodo(ctx context.Context, t *model.Todo) error {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		return fmt.Errorf("create todo: %w", err)
	}
	return nil
}

func (r *adeRepository) SaveTaskPlan(ctx context.Context, t *model.Task, todos []*model.Todo) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(t).Error; err != nil {
			return fmt.Errorf("save task plan: %w", err)
		}
		if err := tx.Where("task_id = ? AND source = ?", t.ID, model.TodoSourceAgent).Delete(&model.Todo{}).Error; err != nil {
			return fmt.Errorf("replace agent todos: %w", err)
		}
		for i, td := range todos {
			td.TaskID = t.ID
			td.Seq = i + 1
			td.Source = model.TodoSourceAgent
			td.Status = model.TodoStatusTodo
			if err := tx.Create(td).Error; err != nil {
				return fmt.Errorf("create plan todo: %w", err)
			}
		}
		var userTodos []model.Todo
		if err := tx.Where("task_id = ? AND source = ?", t.ID, model.TodoSourceUser).
			Order("seq asc, id asc").Find(&userTodos).Error; err != nil {
			return fmt.Errorf("list user todos: %w", err)
		}
		for i := range userTodos {
			seq := len(todos) + i + 1
			if err := tx.Model(&userTodos[i]).Update("seq", seq).Error; err != nil {
				return fmt.Errorf("resequence user todo: %w", err)
			}
		}
		return nil
	})
}

func (r *adeRepository) UpdateTodo(ctx context.Context, t *model.Todo) error {
	if err := r.db.WithContext(ctx).Save(t).Error; err != nil {
		return fmt.Errorf("update todo: %w", err)
	}
	return nil
}

func (r *adeRepository) DeleteTodo(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.Todo{}, id).Error
}

func (r *adeRepository) GetTodo(ctx context.Context, id uint64) (*model.Todo, error) {
	var t model.Todo
	if err := r.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, fmt.Errorf("get todo: %w", err)
	}
	return &t, nil
}

func (r *adeRepository) ListTodos(ctx context.Context, taskID uint64) ([]model.Todo, error) {
	var list []model.Todo
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).Order("seq asc, id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}
	return list, nil
}

func (r *adeRepository) ResetTodos(ctx context.Context, taskID uint64) error {
	return r.db.WithContext(ctx).Model(&model.Todo{}).
		Where("task_id = ? AND source = ?", taskID, model.TodoSourceUser).
		Update("status", model.TodoStatusTodo).Error
}

func (r *adeRepository) CreateRun(ctx context.Context, run *model.Run) error {
	if err := r.db.WithContext(ctx).Create(run).Error; err != nil {
		return fmt.Errorf("create run: %w", err)
	}
	return nil
}

func (r *adeRepository) UpdateRun(ctx context.Context, run *model.Run) error {
	if err := r.db.WithContext(ctx).Save(run).Error; err != nil {
		return fmt.Errorf("update run: %w", err)
	}
	return nil
}

func (r *adeRepository) GetRun(ctx context.Context, id uint64) (*model.Run, error) {
	var run model.Run
	if err := r.db.WithContext(ctx).First(&run, id).Error; err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}
	return &run, nil
}

func (r *adeRepository) ListRuns(ctx context.Context, userID uint64, limit int) ([]model.Run, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var list []model.Run
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).Order("id desc").Limit(limit).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	return list, nil
}

func (r *adeRepository) ListRunsByTask(ctx context.Context, taskID uint64) ([]model.Run, error) {
	var list []model.Run
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).Order("id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list task runs: %w", err)
	}
	return list, nil
}

func (r *adeRepository) AppendRunEvent(ctx context.Context, e *model.RunEvent) error {
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("append run event: %w", err)
	}
	return nil
}

func (r *adeRepository) ListRunEvents(ctx context.Context, runID uint64, afterSeq int) ([]model.RunEvent, error) {
	var list []model.RunEvent
	if err := r.db.WithContext(ctx).
		Where("run_id = ? AND seq > ?", runID, afterSeq).
		Order("seq asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list run events: %w", err)
	}
	return list, nil
}

func (r *adeRepository) CreateHost(ctx context.Context, h *model.DeployHost) error {
	if err := r.db.WithContext(ctx).Create(h).Error; err != nil {
		return fmt.Errorf("create host: %w", err)
	}
	return nil
}

func (r *adeRepository) UpdateHost(ctx context.Context, h *model.DeployHost) error {
	if err := r.db.WithContext(ctx).Save(h).Error; err != nil {
		return fmt.Errorf("update host: %w", err)
	}
	return nil
}

func (r *adeRepository) DeleteHost(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.DeployHost{}, id).Error
}

func (r *adeRepository) GetHost(ctx context.Context, id uint64) (*model.DeployHost, error) {
	var h model.DeployHost
	if err := r.db.WithContext(ctx).First(&h, id).Error; err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}
	return &h, nil
}

func (r *adeRepository) ListHosts(ctx context.Context, userID uint64) ([]model.DeployHost, error) {
	var list []model.DeployHost
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).Order("id desc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list hosts: %w", err)
	}
	return list, nil
}

func (r *adeRepository) CreateDeployTask(ctx context.Context, t *model.DeployTask) error {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		return fmt.Errorf("create deploy task: %w", err)
	}
	return nil
}

func (r *adeRepository) UpdateDeployTask(ctx context.Context, t *model.DeployTask) error {
	if err := r.db.WithContext(ctx).Save(t).Error; err != nil {
		return fmt.Errorf("update deploy task: %w", err)
	}
	return nil
}

func (r *adeRepository) DeleteDeployTask(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.DeployTask{}, id).Error
}

func (r *adeRepository) GetDeployTask(ctx context.Context, id uint64) (*model.DeployTask, error) {
	var t model.DeployTask
	if err := r.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, fmt.Errorf("get deploy task: %w", err)
	}
	return &t, nil
}

func (r *adeRepository) ListDeployTasks(ctx context.Context, userID, projectID uint64) ([]model.DeployTask, error) {
	var list []model.DeployTask
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND project_id = ?", userID, projectID).
		Order("id desc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list deploy tasks: %w", err)
	}
	return list, nil
}

func (r *adeRepository) CreateRelease(ctx context.Context, rel *model.Release) error {
	if err := r.db.WithContext(ctx).Create(rel).Error; err != nil {
		return fmt.Errorf("create release: %w", err)
	}
	return nil
}

func (r *adeRepository) ListReleases(ctx context.Context, userID uint64, limit int) ([]model.Release, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var list []model.Release
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).Order("id desc").Limit(limit).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list releases: %w", err)
	}
	return list, nil
}

func (r *adeRepository) LatestReleases(ctx context.Context, hostID uint64, limit int) ([]model.Release, error) {
	if limit <= 0 {
		limit = 5
	}
	var list []model.Release
	if err := r.db.WithContext(ctx).
		Where("host_id = ?", hostID).Order("id desc").Limit(limit).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("latest releases: %w", err)
	}
	return list, nil
}

func (r *adeRepository) AddCommandAudit(ctx context.Context, a *model.CommandAudit) error {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		return fmt.Errorf("add command audit: %w", err)
	}
	return nil
}

func (r *adeRepository) ListCommandAudits(ctx context.Context, userID uint64, hostID, taskID uint64, limit int) ([]model.CommandAudit, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if hostID > 0 {
		q = q.Where("host_id = ?", hostID)
	}
	if taskID > 0 {
		q = q.Where("task_id = ?", taskID)
	}
	var list []model.CommandAudit
	if err := q.Order("id desc").Limit(limit).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list command audits: %w", err)
	}
	return list, nil
}

func (r *adeRepository) CreateScene(ctx context.Context, sc *model.Scene) error {
	if err := r.db.WithContext(ctx).Create(sc).Error; err != nil {
		return fmt.Errorf("create scene: %w", err)
	}
	return nil
}

func (r *adeRepository) UpdateScene(ctx context.Context, sc *model.Scene) error {
	if err := r.db.WithContext(ctx).Save(sc).Error; err != nil {
		return fmt.Errorf("update scene: %w", err)
	}
	return nil
}

func (r *adeRepository) DeleteScene(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&model.Scene{}, id).Error; err != nil {
		return fmt.Errorf("delete scene: %w", err)
	}
	return nil
}

func (r *adeRepository) GetScene(ctx context.Context, id uint64) (*model.Scene, error) {
	var sc model.Scene
	if err := r.db.WithContext(ctx).First(&sc, id).Error; err != nil {
		return nil, fmt.Errorf("get scene: %w", err)
	}
	return &sc, nil
}

func (r *adeRepository) ListVisibleScenes(ctx context.Context, userID uint64) ([]model.Scene, error) {
	var list []model.Scene
	err := r.db.WithContext(ctx).
		Where("user_id = 0 OR user_id = ?", userID).
		Order("user_id asc, id asc").
		Find(&list).Error
	if err != nil {
		return nil, fmt.Errorf("list scenes: %w", err)
	}
	return list, nil
}

func (r *adeRepository) ListAdminScenes(ctx context.Context) ([]model.Scene, error) {
	var list []model.Scene
	if err := r.db.WithContext(ctx).Where("user_id = 0").Order("id desc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list admin scenes: %w", err)
	}
	return list, nil
}
