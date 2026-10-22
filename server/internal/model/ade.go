package model

import "chengzi-agent-code-server/pkg/timex"

// 待办来源。
const (
	TodoSourceAgent = "agent"
	TodoSourceUser  = "user"
)

// 待办状态。
const (
	TodoStatusTodo  = "todo"
	TodoStatusDoing = "doing"
	TodoStatusDone  = "done"
)

// 执行阶段，用来把后台会话挂到对应工位。
const (
	RunKindPlanner   = "planner"
	RunKindCoder     = "coder"
	RunKindReviewer  = "reviewer"
	RunKindInstaller = "installer"
	RunKindDeploy    = "deploy"
)

// Project 项目是容器。
type Project struct {
	ID           uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID       uint64          `gorm:"not null;index;comment:所属用户" json:"userId"`
	Name         string          `gorm:"size:128;not null;comment:项目名称" json:"name"`
	Description  string          `gorm:"type:text;not null;default:'';comment:项目介绍" json:"description"`
	LocalDir     string          `gorm:"size:1024;not null;default:'';comment:本机仓库目录" json:"localDir"`
	Scenario     string          `gorm:"size:64;not null;default:'';comment:场景" json:"scenario"`
	Technologies string          `gorm:"size:512;not null;default:'';comment:技术栈，逗号分隔" json:"-"`
	ModelID      uint64          `gorm:"not null;default:0;comment:默认模型，0 表示未选" json:"modelId"`
	SceneID      uint64          `gorm:"not null;default:0;comment:所选场景，0 表示未选" json:"sceneId"`
	EnvCheck     string          `gorm:"type:text;not null;comment:环境检测结果 JSON" json:"-"`
	EnvReady     bool            `gorm:"not null;default:false;comment:环境是否齐备" json:"envReady"`
	ClientID     string          `gorm:"size:128;not null;default:'';comment:绑定的桌面端" json:"clientId"`
	CreatedAt    timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt    timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (Project) TableName() string    { return "ade_projects" }
func (Project) TableComment() string { return "项目容器" }

// Scene 新建项目时可选的场景。UserID 为 0 表示管理员维护的公共场景。
type Scene struct {
	ID          uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID      uint64          `gorm:"not null;default:0;index;comment:所属用户，0 为公共场景" json:"userId"`
	Title       string          `gorm:"size:256;not null;comment:场景标题" json:"title"`
	Description string          `gorm:"type:text;not null;default:'';comment:场景说明" json:"description"`
	CreatedAt   timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt   timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (Scene) TableName() string    { return "ade_scenes" }
func (Scene) TableComment() string { return "新建项目时可选的场景" }

// Task 任务是一条计划。
type Task struct {
	ID          uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID      uint64          `gorm:"not null;index;comment:所属用户" json:"userId"`
	ProjectID   uint64          `gorm:"not null;index;comment:所属项目" json:"projectId"`
	Name        string          `gorm:"size:256;not null;default:'';comment:任务名称" json:"name"`
	Overview    string          `gorm:"type:text;not null;comment:计划概览" json:"overview"`
	Description string          `gorm:"type:text;not null;comment:润色后的需求说明" json:"description"`
	RawRequest  string          `gorm:"type:text;not null;comment:用户原始需求" json:"rawRequest"`
	CreatedAt   timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt   timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (Task) TableName() string    { return "ade_tasks" }
func (Task) TableComment() string { return "开发任务" }

// TaskPlanOpen 待办都还没开始时，计划可以改。
func TaskPlanOpen(todos []Todo) bool {
	for _, td := range todos {
		if td.Status != TodoStatusTodo {
			return false
		}
	}
	return true
}

// Todo 有序步骤。Agent 生成的不可改不可删。
type Todo struct {
	ID        uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	TaskID    uint64          `gorm:"not null;index;comment:所属任务" json:"taskId"`
	Seq       int             `gorm:"not null;default:0;comment:顺序" json:"seq"`
	Title     string          `gorm:"size:256;not null;comment:标题" json:"title"`
	Detail    string          `gorm:"type:text;not null;comment:步骤说明" json:"detail"`
	Status    string          `gorm:"size:16;not null;default:'todo';comment:待办状态。todo 待办，doing 进行中，done 完成" json:"status"`
	Source    string          `gorm:"size:16;not null;default:'agent';comment:来源。agent 智能体生成且不可改，user 用户追加" json:"source"`
	CreatedAt timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (Todo) TableName() string    { return "ade_todos" }
func (Todo) TableComment() string { return "任务待办" }

// DeployHost 部署主机档案。私钥不落库，只在本机。
type DeployHost struct {
	ID          uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID      uint64          `gorm:"not null;index;comment:所属用户" json:"userId"`
	Name        string          `gorm:"size:128;not null;comment:档案名称" json:"name"`
	Host        string          `gorm:"size:256;not null;comment:主机地址" json:"host"`
	Port        int             `gorm:"not null;default:22;comment:SSH 端口" json:"port"`
	Username    string          `gorm:"size:128;not null;comment:登录用户" json:"username"`
	Fingerprint string          `gorm:"size:256;not null;default:'';comment:主机指纹，核对后回写" json:"fingerprint"`
	RemoteDir   string          `gorm:"size:512;not null;default:'';comment:远程部署目录" json:"remoteDir"`
	Playbook    string          `gorm:"type:text;not null;comment:发布剧本" json:"playbook"`
	CreatedAt   timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt   timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (DeployHost) TableName() string    { return "ade_deploy_hosts" }
func (DeployHost) TableComment() string { return "部署主机档案，私钥不入库" }

// DeployTask 某个项目下的部署任务。执行时把说明发给部署会话。
type DeployTask struct {
	ID        uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID    uint64          `gorm:"not null;index;comment:所属用户" json:"userId"`
	ProjectID uint64          `gorm:"not null;index;comment:所属项目" json:"projectId"`
	Title     string          `gorm:"size:200;not null;comment:标题" json:"title"`
	Prompt    string          `gorm:"type:text;not null;comment:发给部署会话的说明" json:"prompt"`
	CreatedAt timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt timex.LocalTime `gorm:"comment:更新时间" json:"updatedAt"`
}

func (DeployTask) TableName() string    { return "ade_deploy_tasks" }
func (DeployTask) TableComment() string { return "项目下的部署任务" }

// Release 发布记录。
type Release struct {
	ID        uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID    uint64          `gorm:"not null;index;comment:所属用户" json:"userId"`
	TaskID    uint64          `gorm:"not null;index;comment:所属任务" json:"taskId"`
	HostID    uint64          `gorm:"not null;index;comment:部署主机" json:"hostId"`
	Version   string          `gorm:"size:64;not null;comment:发布版本" json:"version"`
	Status    string          `gorm:"size:32;not null;comment:发布结果" json:"status"`
	Detail    string          `gorm:"type:text;not null;comment:发布详情" json:"detail"`
	Previous  string          `gorm:"size:64;not null;default:'';comment:回滚指向的上一版本" json:"previous"`
	CreatedAt timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
}

func (Release) TableName() string    { return "ade_releases" }
func (Release) TableComment() string { return "发布记录" }

// CommandAudit 远程命令审计，回写服务端。
type CommandAudit struct {
	ID        uint64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	UserID    uint64          `gorm:"not null;index;comment:所属用户" json:"userId"`
	TaskID    uint64          `gorm:"not null;default:0;index;comment:所属任务，0 表示没有" json:"taskId"`
	HostID    uint64          `gorm:"not null;default:0;index;comment:部署主机，0 表示没有" json:"hostId"`
	Command   string          `gorm:"type:text;not null;comment:远程命令" json:"command"`
	ExitCode  int             `gorm:"not null;default:0;comment:退出码" json:"exitCode"`
	Output    string          `gorm:"type:text;not null;comment:命令输出" json:"output"`
	CreatedAt timex.LocalTime `gorm:"comment:创建时间" json:"createdAt"`
}

func (CommandAudit) TableName() string    { return "ade_command_audits" }
func (CommandAudit) TableComment() string { return "远程命令审计" }
