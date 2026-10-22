package custom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"

	"chengzi-agent-code-server/internal/logic/conversation"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/clienthub"
	"chengzi-agent-code-server/pkg/clienthub/protocol"
	"chengzi-agent-code-server/pkg/response"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ChatHandler POST /api/conversations/:id/chat SSE 流式对话。
func ChatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathID(r)
		if err != nil {
			response.WriteFail(w, "无效的会话 ID", err.Error())
			return
		}
		var req types.ChatReq
		if err := httpx.Parse(r, &req); err != nil {
			response.WriteFail(w, "参数错误", err.Error())
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			response.WriteFail(w, "当前连接不支持流式响应", "")
			return
		}
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		writeEvent := func(ev agentloop.Event) {
			payload, _ := json.Marshal(ev)
			_, _ = io.WriteString(w, "event: "+string(ev.Type)+"\n")
			_, _ = io.WriteString(w, "data: "+string(payload)+"\n\n")
			flusher.Flush()
		}

		ctx := r.Context()
		err = conversation.NewChatLogic(ctx, svcCtx).Chat(id, middleware.UserID(ctx), req.Content, req.AgentId, req.ClientId, req.Attachments, writeEvent)
		if err != nil && !errors.Is(err, context.Canceled) {
			writeEvent(agentloop.Event{Type: agentloop.EventError, Error: err.Error()})
		}
	}
}

// WebSocketHandler GET /api/clients/ws WebSocket 客户端接入。
func WebSocketHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserID(r.Context())
		if userID == 0 {
			response.WriteFail(w, "未登录", "missing user id")
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade websocket: %v", err)
			return
		}

		client := clienthub.NewRemoteClient(userID, "", "", conn)
		idle := svcCtx.Config.ClientAPI.HeartbeatInterval * 2
		if idle <= 0 {
			idle = 60 * time.Second
		}
		if err := runClient(r.Context(), svcCtx.ClientHub, client, idle); err != nil {
			log.Printf("client session ended: %v", err)
		}
	}
}

func runClient(ctx context.Context, hub *clienthub.Hub, client *clienthub.RemoteClient, readIdle time.Duration) error {
	defer func() {
		hub.Unregister(client.UserID, client.ClientID)
		_ = client.Conn.Close()
	}()

	var reg protocol.Message
	if err := client.Conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	if err := client.Conn.ReadJSON(&reg); err != nil {
		return err
	}
	if reg.Type != protocol.TypeRegister {
		_ = sendError(client, "", "first message must be register")
		return nil
	}
	var p protocol.RegisterPayload
	if err := json.Unmarshal(reg.Payload, &p); err != nil {
		_ = sendError(client, "", "invalid register payload")
		return nil
	}
	if p.ClientID == "" {
		_ = sendError(client, "", "clientId is required")
		return nil
	}
	client.ClientID = p.ClientID
	client.Name = p.Name
	client.Workspace = p.Workspace
	if client.Name == "" {
		client.Name = client.ClientID
	}
	// 控制帧 Pong 在 ReadJSON 阻塞期间到达，必须在这里续期，否则页面定时器被挂起后会被误判离线。
	client.Conn.SetPongHandler(func(string) error {
		if readIdle <= 0 {
			return nil
		}
		return client.Conn.SetReadDeadline(time.Now().Add(readIdle))
	})
	if err := hub.Register(client.UserID, client); err != nil {
		_ = sendError(client, "", err.Error())
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if readIdle > 0 {
			if err := client.Conn.SetReadDeadline(time.Now().Add(readIdle)); err != nil {
				return err
			}
		}
		var msg protocol.Message
		if err := client.Conn.ReadJSON(&msg); err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return fmt.Errorf("client idle timeout")
			}
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				return err
			}
			return nil
		}
		switch msg.Type {
		case protocol.TypeStatus:
			var st protocol.StatusPayload
			if err := json.Unmarshal(msg.Payload, &st); err != nil {
				_ = sendError(client, msg.ID, "invalid status payload")
				continue
			}
			_ = hub.UpdatePresence(client.UserID, client.ClientID, st.Name, st.Workspace, st.PublicURL, true, st.HasPublicURL)
		case protocol.TypeToolResult, protocol.TypeSkillContent, protocol.TypeRpcResult:
			hub.DispatchResult(&msg)
		case protocol.TypePing:
			_ = sendPong(client, msg.ID)
		case protocol.TypePong:
			// 客户端对应用层 ping 的应答，只用来刷新读超时。
		case protocol.TypeError:
			log.Printf("client %s/%d error: %s", client.ClientID, client.UserID, string(msg.Payload))
		default:
			_ = sendError(client, msg.ID, "unknown message type")
		}
	}
}

func parsePathID(r *http.Request) (uint64, error) {
	vars := pathvar.Vars(r)
	raw := vars["id"]
	if raw == "" {
		return 0, strconv.ErrRange
	}
	return strconv.ParseUint(raw, 10, 64)
}

func sendError(client *clienthub.RemoteClient, id, message string) error {
	msg, err := protocol.NewMessage(protocol.TypeError, id, protocol.ErrorPayload{Message: message})
	if err != nil {
		return err
	}
	return client.Send(msg)
}

func sendPong(client *clienthub.RemoteClient, id string) error {
	msg, err := protocol.NewMessage(protocol.TypePong, id, nil)
	if err != nil {
		return err
	}
	return client.Send(msg)
}
