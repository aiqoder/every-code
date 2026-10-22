package custom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/rest/pathvar"

	"chengzi-agent-code-server/internal/handler/hresp"
	"chengzi-agent-code-server/internal/logic/client"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/pkg/clienthub"
	"chengzi-agent-code-server/pkg/response"
)

// ClientRPCHandler POST /api/clients/:clientId/rpc 把白名单调用转到在线客户端。
func ClientRPCHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID := pathvar.Vars(r)["clientId"]
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}
		raw, err := client.NewRpcLogic(r.Context(), svcCtx).Invoke(clientID, req.Method, req.Params)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var data any
		if len(raw) == 0 || string(raw) == "null" {
			data = map[string]any{}
		} else if err := json.Unmarshal(raw, &data); err != nil {
			hresp.WriteFail(w, "客户端返回无法解析", err)
			return
		}
		hresp.WriteOK(w, data)
	}
}

// ClientEventsHandler GET /api/clients/events 推送当前用户客户端的上线、下线与状态变化。
func ClientEventsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserID(r.Context())
		if userID == 0 {
			response.WriteFail(w, "未登录", "missing user id")
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			response.WriteFail(w, "当前连接不支持流式响应", "")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		events, cancel := svcCtx.ClientHub.Subscribe(userID)
		defer cancel()
		for _, item := range svcCtx.ClientHub.List(userID) {
			summary := item
			if err := writePresence(w, flusher, clienthub.PresenceEvent{
				Type:     clienthub.PresenceOnline,
				ClientID: summary.ClientID,
				Client:   &summary,
			}); err != nil {
				return
			}
		}

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case ev := <-events:
				if err := writePresence(w, flusher, ev); err != nil {
					return
				}
			case <-ticker.C:
				if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

func writePresence(w io.Writer, flusher http.Flusher, ev clienthub.PresenceEvent) error {
	raw, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, raw); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
