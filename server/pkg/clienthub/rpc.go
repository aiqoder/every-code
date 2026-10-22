package clienthub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"chengzi-agent-code-server/pkg/clienthub/protocol"
)

// 网页遥控台允许下发的方法。控制台以对话为主，仅保留只读状态与会话绑定。
// 验收、合并和 diff 仍走这里。worktree 由智能体用 shell 创建，计划用 read_context 读取。
var rpcMethods = map[string]struct{}{
	"workspace.list":      {},
	"link.status":         {},
	"conversation.bind":   {},
	"conversation.unbind": {},
	"conversation.agents": {},
	"ade.verify.run":      {},
	"ade.git.merge":       {},
	"ade.git.diff":        {},
}

// AllowedRPC 判断方法是否在遥控白名单内。
func AllowedRPC(method string) bool {
	_, ok := rpcMethods[strings.TrimSpace(method)]
	return ok
}

func rpcUpdatesPresence(method string) bool {
	switch method {
	case "workspace.list", "link.status":
		return true
	default:
		return false
	}
}

// InvokeRPC 向指定在线客户端下发白名单调用并等待结果。客户端不在线时立即失败。
func (h *Hub) InvokeRPC(ctx context.Context, userID uint64, clientID, method string, params json.RawMessage) (json.RawMessage, error) {
	method = strings.TrimSpace(method)
	if !AllowedRPC(method) {
		return nil, fmt.Errorf("不支持的方法")
	}
	c, ok := h.Get(userID, clientID)
	if !ok || !c.connected() {
		return nil, fmt.Errorf("客户端未连接")
	}
	if len(params) == 0 {
		params = json.RawMessage(`{}`)
	}
	id := uuid.NewString()
	req, err := protocol.NewMessage(protocol.TypeRpc, id, protocol.RpcPayload{
		Method: method,
		Params: params,
	})
	if err != nil {
		return nil, err
	}
	ch := make(chan *protocol.Message, 1)
	c.pendMu.Lock()
	c.pending[id] = ch
	c.pendMu.Unlock()
	if err := c.Send(req); err != nil {
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
		return nil, fmt.Errorf("下发调用失败: %w", err)
	}
	msg, err := c.collect(ctx, id, ch, h.invokeTimeout)
	if err != nil {
		return nil, err
	}
	if msg.Type != protocol.TypeRpcResult {
		return nil, fmt.Errorf("unexpected reply type: %s", msg.Type)
	}
	var p protocol.RpcResultPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return nil, fmt.Errorf("invalid rpc result: %w", err)
	}
	if p.Error != "" {
		return nil, fmt.Errorf("%s", p.Error)
	}
	if rpcUpdatesPresence(method) {
		applyPresence(h, userID, clientID, p.Result)
	}
	if len(p.Result) == 0 {
		return json.RawMessage(`null`), nil
	}
	return p.Result, nil
}

func applyPresence(h *Hub, userID uint64, clientID string, raw json.RawMessage) {
	if len(raw) == 0 || string(raw) == "null" {
		return
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return
	}
	var name, workspace string
	updateWS := false
	if v, ok := obj["clientName"]; ok {
		_ = json.Unmarshal(v, &name)
	}
	if v, ok := obj["workspace"]; ok {
		_ = json.Unmarshal(v, &workspace)
		updateWS = true
	}
	_ = h.UpdatePresence(userID, clientID, name, workspace, "", updateWS, false)
}
