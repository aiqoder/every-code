package conversation

import (
	"encoding/json"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/pkg/llmchat"
	"chengzi-agent-code-server/pkg/tokencount"
)

// resolveConversationUsage 优先用最近一次 loop 落库的快照；
// 旧会话没有快照时，按已存消息估算对话占用，避免点开历史时用量条空白。
func resolveConversationUsage(c *model.Conversation) map[string]any {
	if c == nil {
		return nil
	}
	if raw := parseStoredUsage(c.ContextUsage); raw != nil {
		return raw
	}
	limit := int64(0)
	if c.Model != nil && c.Model.InputContextLimit > 0 {
		limit = c.Model.InputContextLimit
	}
	return estimateUsageFromMessages(c.Messages, limit)
}

func parseStoredUsage(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil || len(m) == 0 {
		return nil
	}
	if _, hasUsed := m["used"]; !hasUsed {
		if _, hasLimit := m["limit"]; !hasLimit {
			return nil
		}
	}
	return m
}

func estimateUsageFromMessages(msgs []model.Message, limit int64) map[string]any {
	hist := make([]llmchat.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == string(llmchat.RoleSystem) {
			continue
		}
		msg := llmchat.Message{
			Role:       llmchat.Role(m.Role),
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			Name:       m.ToolName,
		}
		if m.Role == string(llmchat.RoleAssistant) && m.Metadata != "" {
			var calls []llmchat.ToolCall
			if err := json.Unmarshal([]byte(m.Metadata), &calls); err == nil {
				msg.ToolCalls = calls
			}
		}
		hist = append(hist, msg)
	}
	if len(hist) == 0 {
		return nil
	}
	used := tokencount.CountMessages(hist)
	remaining := int64(0)
	ratio := 0.0
	if limit > 0 {
		remaining = limit - int64(used)
		if remaining < 0 {
			remaining = 0
		}
		ratio = float64(used) / float64(limit)
		if ratio > 1 {
			ratio = 1
		}
	}
	return map[string]any{
		"limit":     limit,
		"used":      used,
		"remaining": remaining,
		"ratio":     ratio,
		"buckets": []map[string]any{
			{"key": "system", "label": "系统提示词", "tokens": 0},
			{"key": "tools", "label": "工具", "tokens": 0},
			{"key": "skills", "label": "技能", "tokens": 0},
			{"key": "messages", "label": "对话消息", "tokens": used},
		},
	}
}
