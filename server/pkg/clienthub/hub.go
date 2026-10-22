package clienthub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"chengzi-agent-code-server/pkg/catalogstore"
	"chengzi-agent-code-server/pkg/clienthub/protocol"
)

// ToolMeta 与协议层工具描述一致，供聊天层构造远程工具。
type ToolMeta = protocol.ToolMeta

// RemoteClient 一条已鉴权的本地客户端连接。
type RemoteClient struct {
	UserID    uint64
	ClientID  string
	Name      string
	Workspace string
	PublicURL string
	Conn      *websocket.Conn

	writeMu   sync.Mutex
	pendMu    sync.Mutex
	pending   map[string]chan *protocol.Message
	closed    chan struct{}
	closeOnce sync.Once
}

// NewRemoteClient 构造远程客户端（尚未 Register）。
func NewRemoteClient(userID uint64, clientID, name string, conn *websocket.Conn) *RemoteClient {
	return &RemoteClient{
		UserID:   userID,
		ClientID: clientID,
		Name:     name,
		Conn:     conn,
		pending:  make(map[string]chan *protocol.Message),
		closed:   make(chan struct{}),
	}
}

// Send 线程安全写出一帧。
func (c *RemoteClient) Send(msg *protocol.Message) error {
	if c == nil || c.Conn == nil {
		return fmt.Errorf("client not connected")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.Conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	return c.Conn.WriteJSON(msg)
}

// Ping 发送协议层 Ping。浏览器会自动回 Pong，不依赖页面里的定时器。
func (c *RemoteClient) Ping() error {
	if c == nil || c.Conn == nil {
		return fmt.Errorf("client not connected")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	deadline := time.Now().Add(10 * time.Second)
	_ = c.Conn.SetWriteDeadline(deadline)
	return c.Conn.WriteControl(websocket.PingMessage, []byte("hb"), deadline)
}

func (c *RemoteClient) markClosed() {
	c.closeOnce.Do(func() {
		close(c.closed)
		c.failAllPending(fmt.Errorf("client disconnected"))
	})
}

// connected 判断 WebSocket 连接是否仍可用（Hub 登记不等于实时可达）。
func (c *RemoteClient) connected() bool {
	if c == nil || c.Conn == nil {
		return false
	}
	select {
	case <-c.closed:
		return false
	default:
		return true
	}
}

func (c *RemoteClient) failAllPending(err error) {
	c.pendMu.Lock()
	defer c.pendMu.Unlock()
	for id, ch := range c.pending {
		select {
		case ch <- errorMessage(id, err.Error()):
		default:
		}
		delete(c.pending, id)
	}
}

func errorMessage(id, message string) *protocol.Message {
	msg, err := protocol.NewMessage(protocol.TypeError, id, protocol.ErrorPayload{Message: message})
	if err != nil {
		return &protocol.Message{Type: protocol.TypeError, ID: id}
	}
	return msg
}

// ClientSummary 在线客户端摘要（给前端列表）。
type ClientSummary struct {
	ClientID  string `json:"clientId"`
	Name      string `json:"name"`
	Workspace string `json:"workspace"`
	PublicURL string `json:"publicURL"`
	Online    bool   `json:"online"`
}

const (
	PresenceOnline  = "online"
	PresenceOffline = "offline"
	PresenceCatalog = "catalog"
)

// PresenceEvent 在线状态变化，供 SSE 推给已登录用户。
type PresenceEvent struct {
	Type     string         `json:"type"`
	ClientID string         `json:"clientId,omitempty"`
	AgentID  string         `json:"agentId,omitempty"`
	Client   *ClientSummary `json:"client,omitempty"`
}

// Hub 按用户隔离的在线客户端中心。
type Hub struct {
	mu            sync.RWMutex
	clients       map[uint64]map[string]*RemoteClient
	subs          map[uint64][]chan PresenceEvent
	catalogStore  *catalogstore.Store
	invokeTimeout time.Duration
	heartbeat     time.Duration
}

// NewHub 构造 Hub。
func NewHub(invokeTimeout, heartbeatInterval time.Duration) *Hub {
	if invokeTimeout <= 0 {
		invokeTimeout = 60 * time.Second
	}
	return &Hub{
		clients:       make(map[uint64]map[string]*RemoteClient),
		subs:          make(map[uint64][]chan PresenceEvent),
		invokeTimeout: invokeTimeout,
		heartbeat:     heartbeatInterval,
	}
}

// SetCatalogStore 注入 Redis catalog 存储。能力清单只写 Redis，不留在连接对象上。
func (h *Hub) SetCatalogStore(store *catalogstore.Store) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.catalogStore = store
}

// CatalogStore 返回 catalog 存储，未配置时为 nil。
func (h *Hub) CatalogStore() *catalogstore.Store {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.catalogStore
}

// Register 登记连接；同用户同 clientId 重复接入时替换旧连接。
func (h *Hub) Register(userID uint64, client *RemoteClient) error {
	if client == nil || client.ClientID == "" {
		return fmt.Errorf("clientId is required")
	}
	client.UserID = userID
	h.mu.Lock()
	defer h.mu.Unlock()
	byUser, ok := h.clients[userID]
	if !ok {
		byUser = make(map[string]*RemoteClient)
		h.clients[userID] = byUser
	}
	if old, exists := byUser[client.ClientID]; exists && old != client {
		old.markClosed()
		_ = old.Conn.Close()
	}
	byUser[client.ClientID] = client
	h.publishLocked(userID, client.presence(PresenceOnline))
	if h.heartbeat > 0 {
		go h.pingLoop(client)
	}
	return nil
}

// Unregister 移除连接。
func (h *Hub) Unregister(userID uint64, clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	byUser, ok := h.clients[userID]
	if !ok {
		return
	}
	if c, ok := byUser[clientID]; ok {
		c.markClosed()
		delete(byUser, clientID)
		h.publishLocked(userID, PresenceEvent{Type: PresenceOffline, ClientID: clientID})
		if h.catalogStore != nil {
			_ = h.catalogStore.DeleteClient(context.Background(), userID, clientID)
		}
	}
	if len(byUser) == 0 {
		delete(h.clients, userID)
	}
}

// UpdateCatalog 把一台客户端的能力清单写入 Redis。连接对象只表示在线，不保存清单副本。
func (h *Hub) UpdateCatalog(userID uint64, clientID string, catalog protocol.CatalogPayload) error {
	if catalog.Tools == nil {
		catalog.Tools = []protocol.ToolMeta{}
	}
	if catalog.Skills == nil {
		catalog.Skills = []protocol.SkillMeta{}
	}
	h.mu.Lock()
	c, ok := h.clientLocked(userID, clientID)
	store := h.catalogStore
	if ok {
		h.publishLocked(userID, c.presence(PresenceCatalog))
	}
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("client not found")
	}
	if store == nil {
		return fmt.Errorf("catalog store is nil")
	}
	return store.Save(context.Background(), userID, clientID, catalog)
}

// UpdatePresence 更新显示名、工作目录与公网地址。
// name 为空时保留原名称；updateWorkspace 为真时写入 workspace（允许空）；
// updatePublic 为真时写入 publicURL（允许空，表示关闭外网）。
func (h *Hub) UpdatePresence(userID uint64, clientID, name, workspace, publicURL string, updateWorkspace, updatePublic bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.clientLocked(userID, clientID)
	if !ok {
		return fmt.Errorf("client not found")
	}
	if strings.TrimSpace(name) != "" {
		c.Name = strings.TrimSpace(name)
	}
	if updateWorkspace {
		c.Workspace = strings.TrimSpace(workspace)
	}
	if updatePublic {
		c.PublicURL = strings.TrimSpace(publicURL)
	}
	h.publishLocked(userID, c.presence(PresenceCatalog))
	return nil
}

// Subscribe 订阅某用户的在线变化。cancel 后不再投递。
func (h *Hub) Subscribe(userID uint64) (<-chan PresenceEvent, func()) {
	ch := make(chan PresenceEvent, 32)
	h.mu.Lock()
	h.subs[userID] = append(h.subs[userID], ch)
	h.mu.Unlock()
	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			list := h.subs[userID]
			kept := make([]chan PresenceEvent, 0, len(list))
			for _, item := range list {
				if item != ch {
					kept = append(kept, item)
				}
			}
			if len(kept) == 0 {
				delete(h.subs, userID)
			} else {
				h.subs[userID] = kept
			}
		})
	}
	return ch, cancel
}

// Get 按用户与客户端 ID 查找。
func (h *Hub) Get(userID uint64, clientID string) (*RemoteClient, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	byUser, ok := h.clients[userID]
	if !ok {
		return nil, false
	}
	c, ok := byUser[clientID]
	return c, ok
}

// List 返回某用户当前在线客户端。
func (h *Hub) List(userID uint64) []ClientSummary {
	h.mu.RLock()
	defer h.mu.RUnlock()
	byUser := h.clients[userID]
	out := make([]ClientSummary, 0, len(byUser))
	for _, c := range byUser {
		out = append(out, c.summary())
	}
	return out
}

// IsUserOnline 判断某用户是否有任一在线客户端（用于智能体列表展示创建人在线状态）。
func (h *Hub) IsUserOnline(userID uint64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	byUser, ok := h.clients[userID]
	if !ok || len(byUser) == 0 {
		return false
	}
	for _, c := range byUser {
		if c.connected() {
			return true
		}
	}
	return false
}

// DispatchResult 将客户端回包交给等待中的调用。
func (h *Hub) DispatchResult(msg *protocol.Message) {
	if msg == nil || msg.ID == "" {
		return
	}
	h.mu.RLock()
	var target *RemoteClient
	for _, byUser := range h.clients {
		for _, c := range byUser {
			c.pendMu.Lock()
			if _, ok := c.pending[msg.ID]; ok {
				target = c
				c.pendMu.Unlock()
				break
			}
			c.pendMu.Unlock()
		}
		if target != nil {
			break
		}
	}
	h.mu.RUnlock()
	if target == nil {
		return
	}
	target.pendMu.Lock()
	ch, ok := target.pending[msg.ID]
	if ok {
		delete(target.pending, msg.ID)
	}
	target.pendMu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- msg:
	default:
	}
}

// CloseAll 断开全部客户端。
func (h *Hub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for userID, byUser := range h.clients {
		for id, c := range byUser {
			c.markClosed()
			_ = c.Conn.Close()
			delete(byUser, id)
		}
		delete(h.clients, userID)
	}
}

func (h *Hub) pingLoop(c *RemoteClient) {
	ticker := time.NewTicker(h.heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-c.closed:
			return
		case <-ticker.C:
			cur, ok := h.Get(c.UserID, c.ClientID)
			if !ok || cur != c {
				return
			}
			if err := c.Ping(); err != nil {
				// 写失败后连接已不可用，关掉它让读循环退出并注销，避免一直占着「在线」。
				_ = c.Conn.Close()
				return
			}
			h.mu.RLock()
			store := h.catalogStore
			h.mu.RUnlock()
			if store != nil {
				_ = store.TouchClient(context.Background(), c.UserID, c.ClientID)
			}
		}
	}
}

func (c *RemoteClient) waitResult(ctx context.Context, id string, timeout time.Duration) (*protocol.Message, error) {
	ch := make(chan *protocol.Message, 1)
	c.pendMu.Lock()
	c.pending[id] = ch
	c.pendMu.Unlock()
	// 调用方若给了更宽的截止时间（如长命令的 shell_exec），以它为准。
	if d, ok := ctx.Deadline(); ok {
		if left := time.Until(d) + time.Second; left > timeout {
			timeout = left
		}
	}
	return c.collect(ctx, id, ch, timeout)
}

func (c *RemoteClient) collect(ctx context.Context, id string, ch chan *protocol.Message, timeout time.Duration) (*protocol.Message, error) {
	defer func() {
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closed:
		return nil, fmt.Errorf("client disconnected")
	case <-timer.C:
		return nil, fmt.Errorf("invoke timeout")
	case msg := <-ch:
		if msg == nil {
			return nil, fmt.Errorf("empty result")
		}
		if msg.Type == protocol.TypeError {
			var p protocol.ErrorPayload
			if err := json.Unmarshal(msg.Payload, &p); err == nil && p.Message != "" {
				return nil, fmt.Errorf("%s", p.Message)
			}
			return nil, fmt.Errorf("client error")
		}
		return msg, nil
	}
}

func (c *RemoteClient) summary() ClientSummary {
	return ClientSummary{
		ClientID:  c.ClientID,
		Name:      c.Name,
		Workspace: c.Workspace,
		PublicURL: c.PublicURL,
		Online:    c.connected(),
	}
}

func (c *RemoteClient) presence(typ string) PresenceEvent {
	s := c.summary()
	return PresenceEvent{Type: typ, ClientID: c.ClientID, Client: &s}
}

func (h *Hub) clientLocked(userID uint64, clientID string) (*RemoteClient, bool) {
	byUser, ok := h.clients[userID]
	if !ok {
		return nil, false
	}
	c, ok := byUser[clientID]
	return c, ok
}

func (h *Hub) publishLocked(userID uint64, ev PresenceEvent) {
	for _, ch := range h.subs[userID] {
		select {
		case ch <- ev:
		default:
		}
	}
}
