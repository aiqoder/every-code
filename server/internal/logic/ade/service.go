package ade

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"chengzi-agent-code-server/internal/logic/common"
	convlogic "chengzi-agent-code-server/internal/logic/conversation"
	"chengzi-agent-code-server/internal/logic/llmprovider"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/clienthub"
	"chengzi-agent-code-server/pkg/timex"
)

// Service ADE 领域服务：拆解、执行、验收、复核、调度的编排都在这里。
type Service struct {
	svcCtx *svc.ServiceContext
	repo   repository.AdeRepository
	bus    *EventBus

	mu      sync.Mutex
	running map[uint64]struct{} // taskId 正在跑的集合
	queued  map[uint64]struct{} // 已点开始、桌面不在线时等下一轮再拉起
	seq     map[uint64]int      // 每任务事件序号
	skills  *SkillStore

	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewService 构造 ADE 服务。skillRoot 为系统技能目录。
func NewService(svcCtx *svc.ServiceContext, repo repository.AdeRepository, skillRoot string) *Service {
	return &Service{
		svcCtx:  svcCtx,
		repo:    repo,
		bus:     NewEventBus(),
		running: make(map[uint64]struct{}),
		queued:  make(map[uint64]struct{}),
		seq:     make(map[uint64]int),
		skills:  NewSkillStore(skillRoot),
		stopCh:  make(chan struct{}),
	}
}

// Skills 返回系统技能仓库。
func (s *Service) Skills() *SkillStore { return s.skills }

// SystemSkillIDs 返回系统技能 id 集合，供能力清单按绑定裁剪。
func (s *Service) SystemSkillIDs() map[string]struct{} {
	out := map[string]struct{}{}
	if s == nil || s.skills == nil {
		return out
	}
	list, err := s.skills.List()
	if err != nil {
		return out
	}
	for _, meta := range list {
		id := strings.TrimSpace(meta.ID)
		if id == "" {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}

// Bus 暴露事件总线给 SSE 订阅。
func (s *Service) Bus() *EventBus { return s.bus }

// Repo 暴露仓储。
func (s *Service) Repo() repository.AdeRepository { return s.repo }

// ---------------------------------------------------------------------------
// 项目
// ---------------------------------------------------------------------------

// CreateProject 校验必填项后落库；技术栈可空。
func (s *Service) CreateProject(ctx context.Context, userID uint64, name, description, localDir, scenario string, technologies []string, modelID, sceneID uint64) (*model.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("项目名称不能为空")
	}
	localDir = strings.TrimSpace(localDir)
	if localDir == "" {
		return nil, fmt.Errorf("本地目录不能为空")
	}
	description = strings.TrimSpace(description)
	if sceneID > 0 {
		sc, err := s.sceneForUser(ctx, userID, sceneID)
		if err != nil {
			return nil, err
		}
		description = sc.Description
	}
	techs := NormalizeTechs(technologies)
	p := &model.Project{
		UserID:       userID,
		Name:         name,
		Description:  description,
		LocalDir:     localDir,
		Scenario:     strings.TrimSpace(scenario),
		Technologies: strings.Join(techs, ","),
		ModelID:      modelID,
		SceneID:      sceneID,
	}
	if err := s.repo.CreateProject(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// UpdateProjectInput 可改字段；指针/切片区分「未传」与「清空」。
type UpdateProjectInput struct {
	Name         *string
	Description  *string
	LocalDir     *string
	Scenario     *string
	Technologies []string
	TechsSet     bool
	ModelID      *uint64
}

// UpdateProject 更新项目元数据；技术栈若传入则整表替换。
func (s *Service) UpdateProject(ctx context.Context, p *model.Project, in UpdateProjectInput) (*model.Project, error) {
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, fmt.Errorf("项目名称不能为空")
		}
		p.Name = name
	}
	if in.Description != nil {
		p.Description = strings.TrimSpace(*in.Description)
	}
	if in.LocalDir != nil {
		dir := strings.TrimSpace(*in.LocalDir)
		if dir == "" {
			return nil, fmt.Errorf("本地目录不能为空")
		}
		p.LocalDir = dir
	}
	if in.Scenario != nil {
		p.Scenario = strings.TrimSpace(*in.Scenario)
	}
	if in.TechsSet {
		techs := NormalizeTechs(in.Technologies)
		p.Technologies = strings.Join(techs, ",")
	}
	if in.ModelID != nil {
		p.ModelID = *in.ModelID
	}
	p.UpdatedAt = timex.LocalTime(time.Now())
	if err := s.repo.UpdateProject(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ProjectVO 项目对外视图。
type ProjectVO struct {
	*model.Project
	Technologies []string   `json:"technologies"`
	EnvCheck     []EnvCheck `json:"envCheck"`
	TaskCount    int        `json:"taskCount"`
	RunningTask  uint64     `json:"runningTask"`
}

// EnvCheck 单项环境检测结果。
type EnvCheck struct {
	Tech      string `json:"tech"`
	Label     string `json:"label"`
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Command   string `json:"command"`
}

// ToProjectVO 组装项目视图。
func (s *Service) ToProjectVO(ctx context.Context, p *model.Project) *ProjectVO {
	vo := &ProjectVO{
		Project:      p,
		Technologies: SplitTechs(p.Technologies),
		EnvCheck:     []EnvCheck{},
	}
	if strings.TrimSpace(p.EnvCheck) != "" {
		var items []EnvCheck
		if json.Unmarshal([]byte(p.EnvCheck), &items) == nil && items != nil {
			vo.EnvCheck = items
		}
	}
	tasks, err := s.repo.ListTasksByProject(ctx, p.ID)
	if err == nil {
		vo.TaskCount = len(tasks)
		for _, t := range tasks {
			if s.isRunning(t.ID) {
				vo.RunningTask = t.ID
				break
			}
		}
	}
	return vo
}

// SaveEnvCheck 保存本机环境检测结果，全齐备时置 EnvReady。
func (s *Service) SaveEnvCheck(ctx context.Context, p *model.Project, items []EnvCheck) error {
	allOK := len(items) > 0
	for _, item := range items {
		if !item.Installed {
			allOK = false
		}
	}
	raw, _ := json.Marshal(items)
	p.EnvCheck = string(raw)
	p.EnvReady = allOK
	p.UpdatedAt = timex.LocalTime(time.Now())
	return s.repo.UpdateProject(ctx, p)
}

// ---------------------------------------------------------------------------
// 任务
// ---------------------------------------------------------------------------

// TaskVO 任务对外视图，含待办。
type TaskVO struct {
	*model.Task
	ProjectName string       `json:"projectName"`
	Todos       []model.Todo `json:"todos"`
	DoneCount   int          `json:"doneCount"`
	TotalCount  int          `json:"totalCount"`
}

// ToTaskVO 组装任务视图。
func (s *Service) ToTaskVO(ctx context.Context, t *model.Task, projectName string) *TaskVO {
	vo := &TaskVO{Task: t, ProjectName: projectName, Todos: []model.Todo{}}
	todos, err := s.repo.ListTodos(ctx, t.ID)
	if err == nil {
		vo.Todos = todos
		vo.TotalCount = len(todos)
		for _, td := range todos {
			if td.Status == model.TodoStatusDone {
				vo.DoneCount++
			}
		}
	}
	return vo
}

// TaskTodoInput 对话里定好的一条待办。
type TaskTodoInput struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// TaskPlanInput 对话里定好的计划。创建时一并落库，不再后台拆解。
type TaskPlanInput struct {
	Name     string
	Overview string
	Plan     string
	Todos    []TaskTodoInput
}

// CreateTask 保存对话里已经定好的计划。
func (s *Service) CreateTask(ctx context.Context, userID uint64, projectID uint64, description string, plan TaskPlanInput) (*model.Task, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return nil, fmt.Errorf("请填写任务描述")
	}
	name := strings.TrimSpace(plan.Name)
	todos := cleanPlanTodos(plan.Todos)
	if len(todos) > 0 && name == "" {
		return nil, fmt.Errorf("请填写任务名称")
	}
	p, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("项目不存在")
	}
	if p.UserID != userID {
		return nil, fmt.Errorf("无权访问该项目")
	}
	// 用户原文只进 RawRequest；Description 留给智能体润色后写入，建任务时不把原文填进去。
	t := &model.Task{
		UserID:      userID,
		ProjectID:   p.ID,
		Name:        name,
		Overview:    strings.TrimSpace(plan.Overview),
		RawRequest:  description,
		Description: strings.TrimSpace(plan.Plan),
	}
	if err := s.repo.CreateTask(ctx, t); err != nil {
		return nil, err
	}
	for i, item := range todos {
		td := &model.Todo{
			TaskID: t.ID,
			Seq:    i + 1,
			Title:  item.Title,
			Detail: item.Detail,
			Status: model.TodoStatusTodo,
			Source: model.TodoSourceAgent,
		}
		if err := s.repo.CreateTodo(ctx, td); err != nil {
			return nil, err
		}
	}
	return t, nil
}

// ApplyTaskPlan 把智能体整理好的名称、概览、润色需求和待办写回已经存在的任务。
func (s *Service) ApplyTaskPlan(ctx context.Context, t *model.Task, plan TaskPlanInput) error {
	existing, err := s.repo.ListTodos(ctx, t.ID)
	if err != nil {
		return err
	}
	if !model.TaskPlanOpen(existing) {
		return fmt.Errorf("任务已经开始，不能改计划")
	}
	name := strings.TrimSpace(plan.Name)
	if name == "" {
		return fmt.Errorf("请填写任务名称")
	}
	todos := cleanPlanTodos(plan.Todos)
	if len(todos) == 0 {
		return fmt.Errorf("计划至少要有一条待办")
	}
	polished := strings.TrimSpace(plan.Plan)
	if polished == "" {
		polished = strings.TrimSpace(t.RawRequest)
	}
	if polished == "" {
		return fmt.Errorf("请填写任务描述")
	}
	t.Name = name
	t.Overview = strings.TrimSpace(plan.Overview)
	t.Description = polished
	rows := make([]*model.Todo, 0, len(todos))
	for _, item := range todos {
		rows = append(rows, &model.Todo{
			Title:  item.Title,
			Detail: item.Detail,
		})
	}
	return s.repo.SaveTaskPlan(ctx, t, rows)
}

func cleanPlanTodos(items []TaskTodoInput) []TaskTodoInput {
	todos := make([]TaskTodoInput, 0, len(items))
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		item.Title = title
		item.Detail = strings.TrimSpace(item.Detail)
		todos = append(todos, item)
	}
	return todos
}

// UpdateDescription 只保存需求原文，不改润色后的 Description，也不在后台重新拆解。
func (s *Service) UpdateDescription(ctx context.Context, t *model.Task, description string) error {
	description = strings.TrimSpace(description)
	if description == "" {
		return fmt.Errorf("需求描述不能为空")
	}
	t.RawRequest = description
	return s.repo.UpdateTask(ctx, t)
}

// AddUserTodo 用户可以追加待办。
func (s *Service) AddUserTodo(ctx context.Context, taskID uint64, title, detail string) (*model.Todo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("待办标题不能为空")
	}
	existing, _ := s.repo.ListTodos(ctx, taskID)
	seq := len(existing) + 1
	td := &model.Todo{
		TaskID: taskID,
		Seq:    seq,
		Title:  title,
		Detail: strings.TrimSpace(detail),
		Status: model.TodoStatusTodo,
		Source: model.TodoSourceUser,
	}
	if err := s.repo.CreateTodo(ctx, td); err != nil {
		return nil, err
	}
	return td, nil
}

// UpdateUserTodo 只允许改用户自己加的待办。
func (s *Service) UpdateUserTodo(ctx context.Context, td *model.Todo, title, detail, status string) error {
	if td.Source != model.TodoSourceUser {
		return fmt.Errorf("Agent 生成的待办不可修改")
	}
	if v := strings.TrimSpace(title); v != "" {
		td.Title = v
	}
	td.Detail = strings.TrimSpace(detail)
	if status != "" {
		td.Status = status
	}
	td.UpdatedAt = timex.LocalTime(time.Now())
	return s.repo.UpdateTodo(ctx, td)
}

// DeleteUserTodo 只允许删用户自己加的待办。
func (s *Service) DeleteUserTodo(ctx context.Context, td *model.Todo) error {
	if td.Source != model.TodoSourceUser {
		return fmt.Errorf("Agent 生成的待办不可删除")
	}
	return s.repo.DeleteTodo(ctx, td.ID)
}

// DeleteTask 删除整条任务。正在执行时拒绝，避免后台还在往已删除的任务写结果。
func (s *Service) DeleteTask(ctx context.Context, t *model.Task) error {
	if s.isRunning(t.ID) {
		return fmt.Errorf("任务正在执行，不能删除")
	}
	s.forgetQueued(t.ID)
	return s.repo.DeleteTask(ctx, t.ID)
}

// StartTask 人点「开始执行」。还有未完成待办时交给调度器。
func (s *Service) StartTask(ctx context.Context, t *model.Task) error {
	todos, err := s.repo.ListTodos(ctx, t.ID)
	if err != nil {
		return err
	}
	if len(todos) == 0 {
		return fmt.Errorf("任务尚未拆解完成")
	}
	if !hasOpenTodo(todos) {
		return fmt.Errorf("待办已完成")
	}
	if s.isRunning(t.ID) {
		return fmt.Errorf("任务正在执行")
	}
	if busyID, err := s.projectActiveDevelopID(ctx, t.ProjectID, t.ID); err != nil {
		return err
	} else if busyID > 0 {
		return fmt.Errorf("同项目任务 #%d 正在开发，同一时间只能跑一条", busyID)
	}
	s.markQueued(t.ID)
	s.dispatch(t.ID)
	return nil
}

func hasOpenTodo(todos []model.Todo) bool {
	for _, td := range todos {
		if td.Status != model.TodoStatusDone {
			return true
		}
	}
	return false
}

// projectActiveDevelopID 同项目是否已有正在执行的任务（不含 excludeID）。
func (s *Service) projectActiveDevelopID(ctx context.Context, projectID, excludeID uint64) (uint64, error) {
	if projectID == 0 {
		return 0, nil
	}
	tasks, err := s.repo.ListTasksByProject(ctx, projectID)
	if err != nil {
		return 0, err
	}
	for _, other := range tasks {
		if other.ID == excludeID {
			continue
		}
		if s.isRunning(other.ID) {
			return other.ID, nil
		}
	}
	return 0, nil
}

// RetryTask 继续执行还有未完成待办的任务。
func (s *Service) RetryTask(ctx context.Context, t *model.Task) error {
	return s.StartTask(ctx, t)
}

func (s *Service) isRunning(taskID uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.running[taskID]
	return ok
}

func (s *Service) markQueued(taskID uint64) {
	s.mu.Lock()
	s.queued[taskID] = struct{}{}
	s.mu.Unlock()
}

func (s *Service) forgetQueued(taskID uint64) {
	s.mu.Lock()
	delete(s.queued, taskID)
	s.mu.Unlock()
}

func (s *Service) queuedIDs() []uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]uint64, 0, len(s.queued))
	for id := range s.queued {
		ids = append(ids, id)
	}
	return ids
}

// dispatch 启动一个后台协程跑任务，避免重复进入。同项目已有本进程在跑的任务则跳过。
func (s *Service) dispatch(taskID uint64) {
	ctx := context.Background()
	t, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return
	}
	if t.ProjectID > 0 {
		others, _ := s.repo.ListTasksByProject(ctx, t.ProjectID)
		for _, other := range others {
			if other.ID == t.ID {
				continue
			}
			if s.isRunning(other.ID) {
				return
			}
		}
	}

	s.mu.Lock()
	if _, ok := s.running[taskID]; ok {
		s.mu.Unlock()
		return
	}
	s.running[taskID] = struct{}{}
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.running, taskID)
			s.mu.Unlock()
		}()
		s.runTask(context.Background(), taskID)
	}()
}

// ---------------------------------------------------------------------------
// 执行上下文与事件
// ---------------------------------------------------------------------------

// agentRun 一轮执行的内存上下文，不落库。
type agentRun struct {
	TaskID         uint64
	Kind           string
	AgentID        string
	ConversationID uint64
}

func (s *Service) nextSeq(taskID uint64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq[taskID]++
	return s.seq[taskID]
}

func (s *Service) startRun(t *model.Task, kind, agentID string) *agentRun {
	s.bus.Begin(t.ID)
	return &agentRun{TaskID: t.ID, Kind: kind, AgentID: agentID}
}

// emit 广播一条任务级事件，并留在内存里供重连回放。
func (s *Service) emit(run *agentRun, ev Event) {
	if run != nil {
		if ev.TaskID == 0 {
			ev.TaskID = run.TaskID
		}
		ev.Kind = run.Kind
		if ev.ConversationID == 0 {
			ev.ConversationID = run.ConversationID
		}
	}
	ev.Seq = s.nextSeq(ev.TaskID)
	s.bus.Publish(ev)
}

// logLine 面向用户的运行日志（不是模型事件）。
func (s *Service) logLine(t *model.Task, run *agentRun, text string) {
	s.emit(run, Event{TaskID: t.ID, Type: "log", Text: text})
}

// ---------------------------------------------------------------------------
// 模型与对话
// ---------------------------------------------------------------------------

// resolveModel 取项目绑定模型，否则取用户可用的第一个模型。
func (s *Service) resolveModel(ctx context.Context, p *model.Project) (*model.LlmModel, error) {
	if p.ModelID > 0 {
		m, err := s.svcCtx.LlmModelRepo.GetByID(ctx, p.ModelID)
		if err == nil && llmprovider.CanAccessModel(m, p.UserID) {
			return m, nil
		}
	}
	list, _, err := s.svcCtx.LlmModelRepo.ListForUser(ctx, p.UserID, 0, 0, 50)
	if err != nil {
		return nil, fmt.Errorf("读取模型失败: %w", err)
	}
	for i := range list {
		if llmprovider.CanAccessModel(&list[i], p.UserID) {
			return &list[i], nil
		}
	}
	return nil, fmt.Errorf("尚未配置可用模型，请先在管理后台添加模型供应商")
}

// channelForRunKind 把运行类型映射到工位通道，便于会话挂到对应任务。
func channelForRunKind(kind string) string {
	switch kind {
	case model.RunKindDeploy, model.RunKindInstaller:
		return model.ChannelDeploy
	case model.RunKindPlanner:
		return model.ChannelCreate
	default:
		return model.ChannelExecute
	}
}

// runAgent 建会话、调 Agent 循环、返回最终助手文本。
// onEvent 收到任务级事件；workspace 为该轮工作目录。
func (s *Service) runAgent(ctx context.Context, t *model.Task, p *model.Project, run *agentRun, agentID, workspace, content string) (string, error) {
	m, err := s.resolveModel(ctx, p)
	if err != nil {
		return "", err
	}
	clientID := strings.TrimSpace(p.ClientID)
	if clientID == "" {
		return "", fmt.Errorf("没有可下发的客户端")
	}
	if _, online := s.svcCtx.ClientHub.Get(t.UserID, clientID); !online {
		return "", fmt.Errorf("绑定的桌面不在线")
	}

	channel := model.ChannelExecute
	if run != nil {
		channel = channelForRunKind(run.Kind)
	}
	conv := &model.Conversation{
		UserID:    t.UserID,
		Title:     truncate(fmt.Sprintf("%s · %s", agentID, t.Name), 60),
		ModelID:   m.ID,
		ClientID:  clientID,
		Workspace: workspace,
		Link: &model.ConversationLink{
			ProjectID: p.ID,
			Channel:   channel,
			TaskID:    t.ID,
		},
	}
	if err := s.svcCtx.ConversationRepo.Create(ctx, conv); err != nil {
		return "", fmt.Errorf("创建会话失败: %w", err)
	}
	if run != nil {
		run.ConversationID = conv.ID
	}

	runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 45*time.Minute)
	defer cancel()

	logic := convlogic.NewChatLogic(runCtx, s.svcCtx)
	err = logic.Chat(conv.ID, t.UserID, content, agentID, clientID, nil, func(ev agentloop.Event) {
		s.emit(run, fromLoop(ev))
	})
	if err != nil {
		return "", err
	}

	c, err := s.svcCtx.ConversationRepo.GetByID(runCtx, conv.ID)
	if err != nil {
		return "", err
	}
	for i := len(c.Messages) - 1; i >= 0; i-- {
		if c.Messages[i].Role == "assistant" && strings.TrimSpace(c.Messages[i].Content) != "" {
			return c.Messages[i].Content, nil
		}
	}
	return "", fmt.Errorf("智能体没有返回内容")
}

func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

// 让 ade 包引用 convlogic 与 clienthub，避免各文件重复导入。
var (
	_ = common.NormalizePage
	_ = clienthub.AllowedRPC
)
