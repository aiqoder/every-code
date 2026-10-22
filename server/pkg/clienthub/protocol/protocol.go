// Package protocol 定义本地客户端与 cmd/server 之间的 WebSocket 消息契约。
package protocol

import (
	"encoding/json"
	"fmt"
)

const (
	TypeRegister     = "register"
	TypeToolCall     = "tool_call"
	TypeToolResult   = "tool_result"
	TypeSkillLoad    = "skill_load"
	TypeSkillContent = "skill_content"
	TypePing         = "ping"
	TypePong         = "pong"
	TypeError        = "error"
	TypeRpc          = "rpc"
	TypeRpcResult    = "rpc_result"
	TypeStatus       = "status"
)

// Message 帧：type + 可选关联 id + JSON payload。
type Message struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// NewMessage 构造消息；payload 为 nil 时省略内容。
func NewMessage(typ, id string, payload any) (*Message, error) {
	msg := &Message{Type: typ, ID: id}
	if payload == nil {
		return msg, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal %s payload: %w", typ, err)
	}
	msg.Payload = raw
	return msg, nil
}

// RegisterPayload 客户端上线声明。
type RegisterPayload struct {
	ClientID  string `json:"clientId"`
	Name      string `json:"name"`
	Workspace string `json:"workspace,omitempty"`
}

// StatusPayload 客户端上报显示名、工作目录，以及可选的公网访问地址。
type StatusPayload struct {
	Name      string `json:"name"`
	Workspace string `json:"workspace"`
	// PublicURL 穿透后的访问地址。仅当 HasPublicURL 为真时写入（允许空，表示关闭外网）。
	PublicURL    string `json:"publicURL,omitempty"`
	HasPublicURL bool   `json:"hasPublicURL,omitempty"`
}

// RpcPayload 服务端下发的白名单调用。
type RpcPayload struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// RpcResultPayload 客户端回传的调用结果。
type RpcResultPayload struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// ToolMeta 供模型调用的工具描述。
type ToolMeta struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,optional"`
}

// SkillMeta 技能目录项（正文按需拉取）。
// source/path 由服务端补默认值，客户端可以不上报，所以标成 optional。
type SkillMeta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source,optional"`
	Path        string `json:"path,optional"`
}

// AgentMeta 智能体展示投影（由单 agent catalog 派生）。
type AgentMeta struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SystemPrompt string `json:"systemPrompt"`
	Avatar       string `json:"avatar,omitempty"`
	MaxSteps     int    `json:"maxSteps"`
	Version      string `json:"version"`
}

// CatalogPayload 一台客户端的能力清单（按 user-client 注册）。
// 人设字段只在服务端按智能体套用后的运行视图里有值，客户端上报时留空。
type CatalogPayload struct {
	AgentID      string      `json:"agentId,omitempty"`
	Name         string      `json:"name,omitempty"`
	Description  string      `json:"description,omitempty"`
	SystemPrompt string      `json:"systemPrompt,omitempty"`
	Avatar       string      `json:"avatar,omitempty"`
	MaxSteps     int         `json:"maxSteps,omitempty"`
	Version      string      `json:"version,omitempty"`
	Tools        []ToolMeta  `json:"tools"`
	Skills       []SkillMeta `json:"skills"`
	PageChannel  string      `json:"pageChannel,omitempty"`
	OsInfo       string      `json:"osInfo"`
	Workspace    string      `json:"workspace"`
}

// ToAgentMeta 投影为列表/对话使用的智能体元数据。
func (c CatalogPayload) ToAgentMeta() AgentMeta {
	return AgentMeta{
		ID:           c.AgentID,
		Name:         c.Name,
		Description:  c.Description,
		SystemPrompt: c.SystemPrompt,
		Avatar:       c.Avatar,
		MaxSteps:     c.MaxSteps,
		Version:      c.Version,
	}
}

// ToolCallPayload 服务端下发的工具调用。
// Workspace 是这条调用应落脚的工作目录（如某个任务 worktree），
// 客户端优先用它解析相对路径，为空时退回自己登记的工作区。
type ToolCallPayload struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Workspace string `json:"workspace,optional"`
}

// ToolResultPayload 本地执行结果。
type ToolResultPayload struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// SkillLoadPayload 服务端请求技能正文。
type SkillLoadPayload struct {
	SkillID string `json:"skillId"`
}

// SkillContentPayload 技能正文回传。
type SkillContentPayload struct {
	SkillID string `json:"skillId"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// ErrorPayload 协议错误。
type ErrorPayload struct {
	Message string `json:"message"`
}
