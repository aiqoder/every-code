package clienthub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/clienthub/protocol"
	"chengzi-agent-code-server/pkg/llmchat"
)

// RemoteTool 将已注册的本地工具包装为 agentloop.Tool，由服务端在对话中调用。
// workspace 是这条调用应落脚的工作目录，会随 tool_call 下发给客户端。
func (h *Hub) RemoteTool(userID uint64, clientID, workspace string, meta ToolMeta) agentloop.Tool {
	return &remoteTool{hub: h, userID: userID, clientID: clientID, workspace: workspace, meta: meta}
}

type remoteTool struct {
	hub       *Hub
	userID    uint64
	clientID  string
	workspace string
	meta      ToolMeta
}

func (t *remoteTool) Name() string { return t.meta.Name }

func (t *remoteTool) Description() string { return t.meta.Description }

func (t *remoteTool) Parameters() map[string]any {
	if t.meta.Parameters == nil {
		return llmchat.DefaultToolParameters()
	}
	return t.meta.Parameters
}

func (t *remoteTool) Execute(ctx context.Context, argsJSON string, rt *agentloop.Runtime) (string, error) {
	return t.hub.InvokeTool(ctx, t.userID, t.clientID, workspaceOf(t.workspace, rt), t.meta.Name, argsJSON)
}

// workspaceOf 优先用本轮里刚记下的 worktree，否则用会话一开始的目录。
func workspaceOf(fallback string, rt *agentloop.Runtime) string {
	if rt != nil && rt.Shared != nil {
		if w, ok := rt.Shared[agentloop.SharedWorkspace].(string); ok {
			if w = strings.TrimSpace(w); w != "" {
				return w
			}
		}
	}
	return fallback
}

// TimeoutFor 让桌面端自报的命令时长决定服务端等待上限。
// shell_exec 最长 30 分钟；其余工具沿用默认（2 分钟）。
func (t *remoteTool) TimeoutFor(argsJSON string) time.Duration {
	const maxShell = 30 * time.Minute
	const grace = 45 * time.Second
	if t.meta.Name != "shell_exec" {
		return 0
	}
	var args struct {
		TimeoutMs int64 `json:"timeout_ms"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	// 缺省与桌面端保持一致（5 分钟），否则服务端先超时、桌面还在跑。
	d := time.Duration(args.TimeoutMs) * time.Millisecond
	if d <= 0 {
		d = 5 * time.Minute
	}
	if d > maxShell {
		d = maxShell
	}
	return d + grace
}

// InvokeTool 向指定客户端下发工具调用并等待结果。
func (h *Hub) InvokeTool(ctx context.Context, userID uint64, clientID, workspace, name, argsJSON string) (string, error) {
	c, ok := h.Get(userID, clientID)
	if !ok || !c.connected() {
		return "", fmt.Errorf("客户端未连接，无法调用工具: %s", clientID)
	}
	id := uuid.NewString()
	req, err := protocol.NewMessage(protocol.TypeToolCall, id, protocol.ToolCallPayload{
		Name:      name,
		Arguments: argsJSON,
		Workspace: workspace,
	})
	if err != nil {
		return "", err
	}
	if err := c.Send(req); err != nil {
		return "", fmt.Errorf("下发工具调用失败: %w", err)
	}
	msg, err := c.waitResult(ctx, id, h.invokeTimeout)
	if err != nil {
		return "", err
	}
	if msg.Type != protocol.TypeToolResult {
		return "", fmt.Errorf("unexpected reply type: %s", msg.Type)
	}
	var p protocol.ToolResultPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return "", fmt.Errorf("invalid tool result: %w", err)
	}
	if p.Error != "" {
		return "", fmt.Errorf("%s", p.Error)
	}
	return p.Content, nil
}

// LoadSkill 向客户端拉取技能正文。
func (h *Hub) LoadSkill(ctx context.Context, userID uint64, clientID, skillID string) (string, error) {
	c, ok := h.Get(userID, clientID)
	if !ok || !c.connected() {
		return "", fmt.Errorf("客户端未连接，无法加载技能: %s", clientID)
	}
	id := uuid.NewString()
	req, err := protocol.NewMessage(protocol.TypeSkillLoad, id, protocol.SkillLoadPayload{SkillID: skillID})
	if err != nil {
		return "", err
	}
	if err := c.Send(req); err != nil {
		return "", fmt.Errorf("请求技能失败: %w", err)
	}
	msg, err := c.waitResult(ctx, id, h.invokeTimeout)
	if err != nil {
		return "", err
	}
	if msg.Type != protocol.TypeSkillContent {
		return "", fmt.Errorf("unexpected reply type: %s", msg.Type)
	}
	var p protocol.SkillContentPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return "", fmt.Errorf("invalid skill content: %w", err)
	}
	if p.Error != "" {
		return "", fmt.Errorf("%s", p.Error)
	}
	return p.Content, nil
}
