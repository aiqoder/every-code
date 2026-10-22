package ade

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
	convlogic "chengzi-agent-code-server/internal/logic/conversation"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/pkg/agentloop"
)

// EnvChatHandler POST /api/ade/projects/:id/env-chat
// 服务端创建会话、组装检测提示词，并用用户默认模型开始 SSE 对话。
func EnvChatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := getService(svcCtx)
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		projectID, err := pathID(r, "id")
		if err != nil {
			hresp.WriteFail(w, err.Error(), err)
			return
		}
		var req struct {
			ClientID string `json:"clientId"`
			ModelID  uint64 `json:"modelId"`
			AgentID  string `json:"agentId"`
		}
		if err := decode(r, &req); err != nil {
			hresp.WriteFail(w, "参数错误", err)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			hresp.WriteFail(w, "当前连接不支持流式响应", nil)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		writeRaw := func(eventName string, payload any) {
			body, _ := json.Marshal(payload)
			_, _ = io.WriteString(w, "event: "+eventName+"\n")
			_, _ = io.WriteString(w, "data: "+string(body)+"\n\n")
			flusher.Flush()
		}
		writeEvent := func(ev agentloop.Event) {
			writeRaw(string(ev.Type), ev)
		}
		fail := func(msg string) {
			writeEvent(agentloop.Event{Type: agentloop.EventError, Error: msg})
		}

		prepared, err := s.PrepareEnvChat(r.Context(), userID(r), projectID, req.ModelID, req.ClientID, req.AgentID)
		if err != nil {
			fail(err.Error())
			return
		}
		writeRaw("conversation", map[string]any{
			"type":           "conversation",
			"conversationId": prepared.ConversationID,
			"modelId":        prepared.ModelID,
			"title":          "检测开发环境",
		})

		err = convlogic.NewChatLogic(r.Context(), svcCtx).Chat(
			prepared.ConversationID,
			userID(r),
			prepared.Prompt,
			prepared.AgentID,
			prepared.ClientID,
			nil,
			writeEvent,
		)
		if err != nil && !errors.Is(err, context.Canceled) {
			fail(err.Error())
		}
	}
}
