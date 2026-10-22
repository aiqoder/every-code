package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"chengzi-agent-code-server/agents"
	"chengzi-agent-code-server/internal/agentcatalog"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	servertool "chengzi-agent-code-server/internal/tool"
	"chengzi-agent-code-server/internal/types"
	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/agentloop/askgate"
	"chengzi-agent-code-server/pkg/agentloop/builtin"
	"chengzi-agent-code-server/pkg/agentloop/httptool"
	"chengzi-agent-code-server/pkg/catalogstore"
	"chengzi-agent-code-server/pkg/clienthub/protocol"
	"chengzi-agent-code-server/pkg/llmchat"
	"chengzi-agent-code-server/pkg/skills"
	"chengzi-agent-code-server/pkg/strutil"
	"chengzi-agent-code-server/pkg/tokencount"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChatLogic) Chat(conversationID, userID uint64, content, agentID, clientID string, attachments []types.ChatAttachment, onEvent func(agentloop.Event)) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("消息内容不能为空")
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return fmt.Errorf("请先选择智能体")
	}
	if onEvent == nil {
		onEvent = func(agentloop.Event) {}
	}
	files, err := decodeAttachments(attachments)
	if err != nil {
		return err
	}
	c, err := l.svcCtx.ConversationRepo.GetByID(l.ctx, conversationID)
	if err != nil {
		return fmt.Errorf("会话不存在")
	}
	if c.UserID != userID {
		return fmt.Errorf("无权访问该会话")
	}
	if err := ensureAttachmentCaps(c.Model, files); err != nil {
		return err
	}

	clientID = l.chatClientID(userID, clientID, c.ClientID)

	// 上一条若仍是用户消息（例如停止后未产生助手回复），覆盖该条而非再插一条。
	var userMsg *model.Message
	if n := len(c.Messages); n > 0 && c.Messages[n-1].Role == string(llmchat.RoleUser) {
		userMsg = &c.Messages[n-1]
	} else {
		userMsg = &model.Message{
			ConversationID: conversationID,
			Role:           string(llmchat.RoleUser),
			Content:        content,
		}
		if err := l.svcCtx.ConversationRepo.AddMessage(l.ctx, userMsg); err != nil {
			return err
		}
		c.Messages = append(c.Messages, *userMsg)
		userMsg = &c.Messages[len(c.Messages)-1]
	}
	meta, err := l.writeAttachments(conversationID, userMsg.ID, files)
	if err != nil {
		return err
	}
	if err := l.svcCtx.ConversationRepo.UpdateMessageBody(l.ctx, userMsg.ID, content, meta); err != nil {
		return err
	}
	userMsg.Content = content
	userMsg.Metadata = meta

	history, err := l.buildHistory(c.Messages)
	if err != nil {
		return err
	}

	if c.Title == "新对话" || c.Title == "" {
		titleSrc := skills.StripSkillTokens(content)
		if titleSrc == "" {
			titleSrc = content
		}
		_ = l.svcCtx.ConversationRepo.UpdateTitle(l.ctx, conversationID, truncateRunes(titleSrc, 40))
	}

	return l.runAgentLoop(conversationID, userID, clientID, agentID, content, history, onEvent)
}

// runAgentLoop 是 Chat 的核心：取会话与客户端 catalog、按智能体套用人设、构造工具与系统提示、
// 调用 agentloop.Run、把本轮新增的 assistant / tool 消息落库。
// history 必须已经包含触发本轮的全部上文（含本轮新 user 消息）。
func (l *ChatLogic) runAgentLoop(
	conversationID, userID uint64,
	clientID, agentID, userContent string,
	history []llmchat.Message,
	onEvent func(agentloop.Event),
) error {
	c, err := l.svcCtx.ConversationRepo.GetByID(l.ctx, conversationID)
	if err != nil {
		return fmt.Errorf("会话不存在")
	}
	if c.Model == nil || c.Model.Provider == nil {
		return fmt.Errorf("会话模型或供应商未配置")
	}

	provider := c.Model.Provider
	protocol, baseURL, err := llmchat.ResolveProtocol(provider.OpenAICompatibleBaseURL, provider.AnthropicBaseURL)
	if err != nil {
		return err
	}
	client, err := llmchat.New(llmchat.Config{
		Protocol: protocol,
		APIKey:   provider.APIKey,
		BaseURL:  baseURL,
	})
	if err != nil {
		return err
	}
	client = wrapTokenUsageRecorder(client, l.svcCtx.TokenUsageRepo, tokenUsageMeta{
		UserID:         userID,
		ConversationID: conversationID,
		ModelID:        c.Model.ID,
		ModelName:      c.Model.Name,
		ProviderID:     c.Model.ProviderID,
		InputPrice:     c.Model.InputPrice,
		OutputPrice:    c.Model.OutputPrice,
		CachePrice:     c.Model.CachePrice,
	})

	clientCatalog, err := l.loadClientCatalog(userID, clientID)
	if err != nil {
		return err
	}
	catalog, err := agentcatalog.Materialize(l.ctx, l.svcCtx, agentID, clientCatalog)
	if err != nil {
		return err
	}
	agent := catalog.ToAgentMeta()
	executorID, executorClientID := userID, clientID
	// 工具落脚目录：会话绑定优先，其次能力清单上报的工作区。
	toolWorkspace := strings.TrimSpace(c.Workspace)
	if toolWorkspace == "" {
		toolWorkspace = strings.TrimSpace(catalog.Workspace)
	}
	kit, err := l.buildChatToolkit(executorID, executorClientID, toolWorkspace, catalog, c)
	if err != nil {
		return err
	}
	systemPrompt, err := l.resolveSystemPrompt(c, catalog, executorID, executorClientID, kit, userContent)
	if err != nil {
		return err
	}

	var lastUsageJSON string
	emit := func(ev agentloop.Event) {
		if ev.Type == agentloop.EventContextUsage || ev.Type == agentloop.EventContextCompacted {
			if b, err := json.Marshal(ev.Data); err == nil && len(ev.Data) > 0 {
				lastUsageJSON = string(b)
			}
		}
		if onEvent != nil {
			onEvent(ev)
		}
	}
	res, runErr := agentloop.Run(l.ctx, agentloop.Config{
		Chat:               client,
		Model:              c.Model.Name,
		Messages:           history,
		Tools:              kit.Tools,
		MaxSteps:           agentMaxSteps(agent),
		SystemPrompt:       systemPrompt,
		OnEvent:            emit,
		InputContextLimit:  c.Model.InputContextLimit,
		OutputContextLimit: c.Model.OutputContextLimit,
		Reasoning:          c.Model.Reasoning,
		ReasoningEffort:    firstReasoningEffort(c.Model.Reasoning, c.Model.ReasoningOptions),
		Ask:                l.askOpener(conversationID, userID),
		AskWait:            l.askWaiter(),
	})

	// 取消后 ctx 已失效，落库需用未取消的 context，以便保留已流出的部分回复。
	saveCtx := l.ctx
	if errors.Is(runErr, context.Canceled) {
		saveCtx = context.WithoutCancel(l.ctx)
	}

	if res != nil {
		if err := l.persistRunMessages(saveCtx, conversationID, history, systemPrompt, res); err != nil {
			if runErr == nil {
				return err
			}
		}
	}
	usageJSON := lastUsageJSON
	if usageJSON == "" && res != nil && (res.Usage.Used > 0 || res.Usage.Limit > 0) {
		if b, err := json.Marshal(res.Usage); err == nil {
			usageJSON = string(b)
		}
	}
	if usageJSON != "" {
		_ = l.svcCtx.ConversationRepo.UpdateContextUsage(saveCtx, conversationID, usageJSON)
	}
	return runErr
}

// buildHistory 把落库消息转成 llmchat 历史，跳过 system 消息。
// 用户消息上的图片和 PDF 会从附件目录读回，放进发给模型的内容块。
func (l *ChatLogic) buildHistory(msgs []model.Message) ([]llmchat.Message, error) {
	history := make([]llmchat.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == string(llmchat.RoleSystem) {
			continue
		}
		msgContent := m.Content
		if m.Role == string(llmchat.RoleUser) {
			msgContent = userContentForLLM(m.Content)
		}
		msg := llmchat.Message{
			Role:       llmchat.Role(m.Role),
			Content:    msgContent,
			ToolCallID: m.ToolCallID,
			Name:       m.ToolName,
		}
		if m.Role == string(llmchat.RoleAssistant) && m.Metadata != "" {
			var calls []llmchat.ToolCall
			if err := json.Unmarshal([]byte(m.Metadata), &calls); err == nil {
				msg.ToolCalls = calls
			}
		}
		if m.Role == string(llmchat.RoleUser) {
			parts, extra, err := l.partsFromMetadata(m.Metadata)
			if err != nil {
				return nil, err
			}
			msg.Parts = parts
			if extra != "" {
				if msg.Content != "" {
					msg.Content += "\n\n"
				}
				msg.Content += extra
			}
		}
		history = append(history, msg)
	}
	return history, nil
}

func (l *ChatLogic) askOpener(conversationID, userID uint64) func(context.Context, string, string) (string, error) {
	host := newChatHost(l.svcCtx.AskGate, conversationID, userID)
	if host == nil {
		return nil
	}
	return func(ctx context.Context, toolCallID, specJSON string) (string, error) {
		var spec askgate.Spec
		if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
			return "", err
		}
		return host.AskOpen(ctx, toolCallID, spec)
	}
}

func (l *ChatLogic) askWaiter() func(context.Context, string) (string, error) {
	gate := l.svcCtx.AskGate
	if gate == nil {
		return nil
	}
	return func(ctx context.Context, questionID string) (string, error) {
		return gate.Wait(ctx, questionID)
	}
}

// AnswerQuestion 提交用户对 AskQuestion 的选择。
// 只做校验并投递给正在 Wait 的 AskQuestion 工具；答案由同一轮 Run 回填为 tool 消息。
func (l *ChatLogic) AnswerQuestion(conversationID, userID uint64, questionID string, answers []askgate.Answer) error {
	if l.svcCtx.AskGate == nil {
		return fmt.Errorf("提问通道未就绪")
	}
	c, err := l.svcCtx.ConversationRepo.GetByID(l.ctx, conversationID)
	if err != nil || c.UserID != userID {
		return fmt.Errorf("无权访问该会话")
	}
	_, _, err = l.svcCtx.AskGate.Submit(conversationID, userID, strings.TrimSpace(questionID), answers)
	return err
}

func (l *ChatLogic) persistRunMessages(
	ctx context.Context,
	conversationID uint64,
	history []llmchat.Message,
	systemPrompt string,
	res *agentloop.Result,
) error {
	// res.Messages = [system?] + history(含本轮 user) + 本轮新增 assistant/tool
	newStart := 0
	if systemPrompt != "" {
		newStart = 1
	}
	newStart += len(history)
	if newStart > len(res.Messages) {
		newStart = len(res.Messages)
	}

	wroteAssistant := false
	for _, m := range res.Messages[newStart:] {
		if m.Role != llmchat.RoleAssistant && m.Role != llmchat.RoleTool {
			continue
		}
		msg := &model.Message{
			ConversationID: conversationID,
			Role:           string(m.Role),
			Content:        m.Content,
			ToolCallID:     m.ToolCallID,
			ToolName:       m.Name,
		}
		if m.Role == llmchat.RoleAssistant && len(m.ToolCalls) > 0 {
			msg.Metadata = llmchat.MustJSON(m.ToolCalls)
		}
		if err := l.svcCtx.ConversationRepo.AddMessage(ctx, msg); err != nil {
			return err
		}
		if m.Role == llmchat.RoleAssistant {
			wroteAssistant = true
		}
	}
	if !wroteAssistant && res.FinalContent != "" {
		if err := l.svcCtx.ConversationRepo.AddMessage(ctx, &model.Message{
			ConversationID: conversationID,
			Role:           string(llmchat.RoleAssistant),
			Content:        res.FinalContent,
		}); err != nil {
			return err
		}
	}
	return nil
}

type chatToolkit struct {
	Tools     []agentloop.Tool
	ToolLines []string
}

// chatClientID 选用本轮执行工具的客户端：请求 clientId 优先，其次会话绑定，否则取在线客户端。
func (l *ChatLogic) chatClientID(userID uint64, reqClientID, stored string) string {
	reqClientID = strings.TrimSpace(reqClientID)
	stored = strings.TrimSpace(stored)
	if reqClientID != "" {
		return reqClientID
	}
	if stored != "" {
		if _, ok := l.svcCtx.ClientHub.Get(userID, stored); ok {
			return stored
		}
	}
	for _, summary := range l.svcCtx.ClientHub.List(userID) {
		if strings.TrimSpace(summary.ClientID) != "" {
			return summary.ClientID
		}
	}
	return stored
}

// loadClientCatalog 读取 Redis 中 (user, client) 的能力清单。
func (l *ChatLogic) loadClientCatalog(userID uint64, clientID string) (protocol.CatalogPayload, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return protocol.CatalogPayload{}, fmt.Errorf("请指定客户端 clientId")
	}
	store := l.svcCtx.ClientHub.CatalogStore()
	if store == nil {
		return protocol.CatalogPayload{}, fmt.Errorf("客户端未上报能力清单")
	}
	cat, err := store.Load(l.ctx, userID, clientID)
	if errors.Is(err, catalogstore.ErrNotFound) {
		return protocol.CatalogPayload{}, fmt.Errorf("客户端未上报能力清单")
	}
	if err != nil {
		return protocol.CatalogPayload{}, fmt.Errorf("读取客户端能力清单失败: %w", err)
	}
	return cat, nil
}

func normalizeOsInfo(raw string) (string, error) {
	return catalogstore.NormalizeOsInfo(raw)
}

func (l *ChatLogic) resolveSystemPrompt(c *model.Conversation, catalog protocol.CatalogPayload, executorID uint64, executorClientID string, kit *chatToolkit, userContent string) (string, error) {
	parts := make([]string, 0, 6)
	if osInfo := strings.TrimSpace(catalog.OsInfo); osInfo != "" {
		if normalized, err := normalizeOsInfo(osInfo); err == nil {
			parts = append(parts, fmt.Sprintf("运行环境：%s", normalized))
		} else {
			parts = append(parts, fmt.Sprintf("运行环境：%s", osInfo))
		}
	}
	// 会话工作目录优先：ADE 的任务会话会绑定到某个 worktree，
	// 不能退回客户端全局工作区，否则工具会改错目录。
	ws := strings.TrimSpace(c.Workspace)
	if ws == "" {
		ws = strings.TrimSpace(catalog.Workspace)
	}
	if ws != "" {
		parts = append(parts, fmt.Sprintf("当前工作目录：%s", ws))
	}
	if sp := strings.TrimSpace(catalog.SystemPrompt); sp != "" {
		parts = append(parts, sp)
	}
	bound := catalogSkillsToBound(catalog.Skills)
	if len(bound) > 0 {
		parts = append(parts, tokencount.BoundSkillsHeading+"\n\n"+skills.FormatBoundSkills(bound))
	}

	invoked, err := l.buildInvokedSkillsPrompt(executorID, executorClientID, catalog, userContent)
	if err != nil {
		return "", err
	}
	if invoked != "" {
		parts = append(parts, invoked)
	}
	if kit != nil && len(kit.ToolLines) > 0 {
		parts = append(parts, fmt.Sprintf("%s\n%s", tokencount.AvailableToolsPrefix, strings.Join(kit.ToolLines, "\n")))
	}
	return strings.Join(parts, "\n\n"), nil
}

func catalogSkillsToBound(metas []protocol.SkillMeta) []skills.Skill {
	out := make([]skills.Skill, 0, len(metas))
	for _, sk := range metas {
		id := strings.TrimSpace(sk.ID)
		if id == "" {
			continue
		}
		source := sk.Source
		if source == "" {
			source = "client"
		}
		out = append(out, skills.Skill{
			ID:          id,
			Name:        sk.Name,
			Description: sk.Description,
			Source:      source,
			Path:        sk.Path,
		})
	}
	return out
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n]) + "…"
}

func firstReasoningEffort(enabled bool, optionsCSV string) string {
	if !enabled {
		return ""
	}
	opts := strutil.SplitCSV(optionsCSV)
	if len(opts) > 0 {
		return opts[0]
	}
	return "medium"
}

// defaultChatMaxSteps 智能体未配置 MaxSteps 时的服务端默认上限。
const defaultChatMaxSteps = 99

// agentMaxSteps 解析本轮实际使用的最大步数：智能体配置优先，未配置（<=0）用默认值。
func agentMaxSteps(agent protocol.AgentMeta) int {
	if agent.MaxSteps > 0 {
		return agent.MaxSteps
	}
	return defaultChatMaxSteps
}

func userContentForLLM(content string) string {
	stripped := skills.ReplaceUserContentForLLM(content)
	if stripped != "" {
		return stripped
	}
	if len(skills.ExtractSkillIDs(content)) > 0 {
		return "请按本轮调用的技能执行。"
	}
	return content
}

// buildInvokedSkillsPrompt 仅加载本轮 /skill:<id> 显式调用的技能（须属于该 agent catalog）。
func (l *ChatLogic) buildInvokedSkillsPrompt(executorID uint64, executorClientID string, catalog protocol.CatalogPayload, userContent string) (string, error) {
	ids := skills.ExtractSkillIDs(userContent)
	if len(ids) == 0 {
		return "", nil
	}
	if executorClientID == "" {
		return "", fmt.Errorf("智能体创建人客户端未连接，无法加载技能")
	}
	if _, ok := l.svcCtx.ClientHub.Get(executorID, executorClientID); !ok {
		return "", fmt.Errorf("智能体创建人客户端未连接: %s", executorClientID)
	}

	nameByID := make(map[string]string, len(catalog.Skills))
	allowed := make(map[string]struct{}, len(catalog.Skills))
	for _, sk := range catalog.Skills {
		nameByID[sk.ID] = sk.Name
		allowed[sk.ID] = struct{}{}
	}

	skillParts := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := allowed[id]; !ok {
			return "", fmt.Errorf("技能 %s 不在当前智能体 catalog 中", id)
		}
		content, err := l.svcCtx.ClientHub.LoadSkill(l.ctx, executorID, executorClientID, id)
		if err != nil {
			return "", fmt.Errorf("加载技能 %s 失败: %w", id, err)
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		name := nameByID[id]
		if name == "" {
			name = id
		}
		skillParts = append(skillParts, fmt.Sprintf("## Skill: %s\n\n%s", name, content))
	}
	if len(skillParts) == 0 {
		return "", nil
	}
	return tokencount.InvokedSkillsHeading + "\n\n" + strings.Join(skillParts, "\n\n---\n\n"), nil
}

// buildChatToolkit 构建 agent loop 工具集。
// 客户端工具来自本机能力清单；内置工具只挂智能体 tools 白名单里点名的；HTTP 管理员工具全量挂上。
// 工位再按通道挂上直接改库的服务端工具。
func (l *ChatLogic) buildChatToolkit(executorID uint64, executorClientID, workspace string, catalog protocol.CatalogPayload, conv *model.Conversation) (*chatToolkit, error) {
	records, err := l.svcCtx.ToolRepo.ListAdminForChat(l.ctx)
	if err != nil {
		return nil, err
	}

	catalogTools := catalog.Tools
	clientOnline := false
	if executorClientID != "" {
		if _, ok := l.svcCtx.ClientHub.Get(executorID, executorClientID); ok {
			clientOnline = true
		}
	}
	dataTools := servertool.For(l.svcCtx.AdeRepo, servertool.Scope{
		UserID:         executorID,
		ClientID:       executorClientID,
		ProjectID:      linkProjectID(conv),
		TaskID:         linkTaskID(conv),
		Channel:        linkChannel(conv),
		ConversationID: linkConversationID(conv),
		EnvCheck:       catalog.AgentID == agents.EnvCheckID,
	})
	if len(records) == 0 && len(catalogTools) == 0 && len(dataTools) == 0 {
		return &chatToolkit{}, nil
	}

	tools := make([]agentloop.Tool, 0, len(records)+len(catalogTools)+len(dataTools))
	toolLines := make([]string, 0, len(records)+len(catalogTools)+len(dataTools))
	seen := make(map[string]struct{}, len(catalogTools)+len(dataTools))
	for _, item := range dataTools {
		name := strings.TrimSpace(item.Name())
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
		tools = append(tools, item)
		toolLines = append(toolLines, formatAvailableToolLine(name, item.Description()))
	}
	if clientOnline {
		for _, meta := range catalogTools {
			name := strings.TrimSpace(meta.Name)
			if name == "" {
				continue
			}
			seen[name] = struct{}{}
			tools = append(tools, l.svcCtx.ClientHub.RemoteTool(executorID, executorClientID, workspace, meta))
			toolLines = append(toolLines, formatAvailableToolLine(name, meta.Description))
		}
	}
	for _, rec := range records {
		name := builtin.CanonicalName(rec.Name)
		if _, ok := seen[name]; ok {
			continue
		}
		switch rec.Type {
		case model.ToolTypeBuiltin:
			if !agentAllowsTool(catalog.AgentID, name) {
				continue
			}
			built := builtin.Toolkit([]string{name})
			if len(built) == 0 {
				continue
			}
			seen[name] = struct{}{}
			tools = append(tools, built[0])
			toolLines = append(toolLines, formatAvailableToolLine(name, rec.Description))
		case model.ToolTypeHTTP:
			t, err := httptool.FromModel(rec.Name, rec.DisplayName, rec.Description, rec.Parameters, rec.Extra)
			if err != nil {
				return nil, fmt.Errorf("加载工具 %s 失败: %w", rec.Name, err)
			}
			tools = append(tools, t)
			toolLines = append(toolLines, formatAvailableToolLine(rec.Name, rec.Description))
		default:
			l.Infof("skip unsupported tool type: %s (%s)", rec.Name, rec.Type)
		}
	}
	return &chatToolkit{
		Tools:     tools,
		ToolLines: toolLines,
	}, nil
}

func agentAllowsTool(agentID, toolName string) bool {
	agent, ok := agents.ByID(agentID)
	if !ok {
		return false
	}
	toolName = strings.TrimSpace(toolName)
	for _, name := range agent.Tools {
		if name == toolName {
			return true
		}
	}
	return false
}

func formatAvailableToolLine(name, description string) string {
	return skills.FormatAvailableLine(name, description)
}

func linkProjectID(c *model.Conversation) uint64 {
	if c == nil || c.Link == nil {
		return 0
	}
	return c.Link.ProjectID
}

func linkTaskID(c *model.Conversation) uint64 {
	if c == nil || c.Link == nil {
		return 0
	}
	return c.Link.TaskID
}

func linkChannel(c *model.Conversation) string {
	if c == nil || c.Link == nil {
		return ""
	}
	return c.Link.Channel
}

func linkConversationID(c *model.Conversation) uint64 {
	if c == nil {
		return 0
	}
	return c.ID
}
