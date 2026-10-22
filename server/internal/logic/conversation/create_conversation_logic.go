package conversation

import (
	"context"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/logic/llmprovider"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateConversationLogic {
	return &CreateConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateConversationLogic) CreateConversation(req *types.CreateConversationReq) (*common.ConversationVO, error) {
	m, err := l.svcCtx.LlmModelRepo.GetByID(l.ctx, req.ModelId)
	if err != nil {
		return nil, fmt.Errorf("模型不存在")
	}
	userID := middleware.UserID(l.ctx)
	if !llmprovider.CanAccessModel(m, userID) {
		return nil, fmt.Errorf("模型不存在或无权使用")
	}
	clientID := strings.TrimSpace(req.ClientId)
	if clientID != "" {
		if _, ok := l.svcCtx.ClientHub.Get(middleware.UserID(l.ctx), clientID); !ok {
			return nil, fmt.Errorf("客户端不存在或未在线")
		}
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "新对话"
	}
	channel, err := normalizeChannel(req.Channel)
	if err != nil {
		return nil, err
	}
	c := &model.Conversation{
		UserID:    userID,
		Title:     title,
		ModelID:   req.ModelId,
		ClientID:  clientID,
		Workspace: strings.TrimSpace(req.Workspace),
	}
	projectID := req.ProjectId
	var task *model.Task
	if req.TaskId > 0 {
		var err error
		task, err = l.svcCtx.AdeRepo.GetTask(l.ctx, req.TaskId)
		if err != nil || task.UserID != userID {
			return nil, fmt.Errorf("任务不存在")
		}
		if projectID > 0 && task.ProjectID != projectID {
			return nil, fmt.Errorf("任务不属于当前项目")
		}
		if projectID == 0 {
			projectID = task.ProjectID
		}
	}
	if projectID > 0 || channel != "" || req.TaskId > 0 {
		c.Link = &model.ConversationLink{
			ProjectID: projectID,
			Channel:   channel,
			TaskID:    req.TaskId,
		}
	}
	if err := l.svcCtx.ConversationRepo.Create(l.ctx, c); err != nil {
		return nil, err
	}
	// clientId 只写在会话表；挂到任务时再回填，供后续执行下发。
	if task != nil && clientID != "" && strings.TrimSpace(task.ClientID) == "" {
		task.ClientID = clientID
		_ = l.svcCtx.AdeRepo.UpdateTask(l.ctx, task)
	}
	return common.ToConversationVO(c, m), nil
}
