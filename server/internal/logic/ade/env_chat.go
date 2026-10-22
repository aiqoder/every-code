package ade

import (
	"context"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/logic/llmprovider"
	"chengzi-agent-code-server/internal/model"
)

// EnvChatPrepared 检测开发环境前已建好的会话。提示词和模型都在服务端决定。
type EnvChatPrepared struct {
	ConversationID uint64
	ModelID        uint64
	AgentID        string
	ClientID       string
	Prompt         string
}

// PrepareEnvChat 为项目创建一条创建工位会话，并用用户默认模型。
// modelID 为 0 时退回项目绑定模型，再退回该用户第一个可用模型。
func (s *Service) PrepareEnvChat(ctx context.Context, userID, projectID, modelID uint64, clientID, agentID string) (*EnvChatPrepared, error) {
	p, err := s.repo.GetProject(ctx, projectID)
	if err != nil || p == nil {
		return nil, fmt.Errorf("项目不存在")
	}
	if p.UserID != userID {
		return nil, fmt.Errorf("无权访问该项目")
	}
	dir := strings.TrimSpace(p.LocalDir)
	if dir == "" {
		return nil, fmt.Errorf("这个项目没有本地目录")
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, fmt.Errorf("请指定客户端")
	}
	if _, ok := s.svcCtx.ClientHub.Get(userID, clientID); !ok {
		return nil, fmt.Errorf("客户端不存在或未在线")
	}
	m, err := s.resolveEnvModel(ctx, userID, modelID, p)
	if err != nil {
		return nil, err
	}
	agent, err := AgentByID(agentID)
	if err != nil {
		return nil, err
	}

	sceneTitle, sceneDesc := "", ""
	if p.SceneID > 0 {
		if sc, sceneErr := s.sceneForUser(ctx, userID, p.SceneID); sceneErr == nil && sc != nil {
			sceneTitle = sc.Title
			sceneDesc = sc.Description
		}
	}
	conv := &model.Conversation{
		UserID:    userID,
		Title:     "检测开发环境",
		ModelID:   m.ID,
		ClientID:  clientID,
		Workspace: dir,
		Link: &model.ConversationLink{
			ProjectID: p.ID,
			Channel:   model.ChannelCreate,
		},
	}
	if err := s.svcCtx.ConversationRepo.Create(ctx, conv); err != nil {
		return nil, fmt.Errorf("创建会话失败")
	}
	return &EnvChatPrepared{
		ConversationID: conv.ID,
		ModelID:        m.ID,
		AgentID:        agent.ID,
		ClientID:       clientID,
		Prompt:         EnvCheckPrompt(p.Name, dir, sceneTitle, sceneDesc, p.Description),
	}, nil
}

// resolveEnvModel 优先使用调用方传入的用户默认模型。
func (s *Service) resolveEnvModel(ctx context.Context, userID, modelID uint64, p *model.Project) (*model.LlmModel, error) {
	if modelID == 0 {
		return s.resolveModel(ctx, p)
	}
	m, err := s.svcCtx.LlmModelRepo.GetByID(ctx, modelID)
	if err != nil || !llmprovider.CanAccessModel(m, userID) {
		return nil, fmt.Errorf("默认模型不存在或无权使用")
	}
	return m, nil
}

// EnvCheckPrompt 组装检测开发环境的用户消息。只描述要查什么，不包含安装步骤。
func EnvCheckPrompt(name, localDir, sceneTitle, sceneDesc, description string) string {
	if strings.TrimSpace(sceneTitle) == "" {
		sceneTitle = "未选择场景"
	}
	lines := []string{
		"请检查这个项目在这台电脑上需要哪些开发环境和软件，并逐项确认是否已经装好。",
		"只检查，不要安装，不要修改仓库。",
		"【项目】" + strings.TrimSpace(name),
		"【目录】" + strings.TrimSpace(localDir),
		"【场景】" + strings.TrimSpace(sceneTitle),
	}
	if text := strings.TrimSpace(sceneDesc); text != "" {
		lines = append(lines, "【场景说明】\n"+text)
	}
	if text := strings.TrimSpace(description); text != "" {
		lines = append(lines, "【项目介绍】\n"+text)
	}
	lines = append(lines,
		"先阅读仓库里能说明技术栈的文件，例如 go.mod、package.json、composer.json、pom.xml、Dockerfile、README。再结合场景，列出真正要装在这台电脑上的语言、运行时、命令行工具和开发软件。纯浏览器就能打开的页面不要算成缺失。",
		"然后用 shell 逐项检查是否已安装，并记下版本。",
		"检查完调用 save_env_check 写回结果。回复末尾再单独输出一个 JSON 对象：",
		`{"items":[{"tech":"简短id","label":"名称","installed":true,"version":"版本或空","command":"检查用的命令"}]}`,
	)
	return strings.Join(lines, "\n")
}
