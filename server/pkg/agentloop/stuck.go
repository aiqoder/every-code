package agentloop

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"chengzi-agent-code-server/pkg/llmchat"
)

// 卡住熔断默认阈值：连续 10 次相同调用 / 连续 10 轮无进展空轮次。
// Config 里 0 表示用默认值，<0 表示禁用对应熔断。
const (
	defaultStuckRepeatLimit = 10
	defaultStuckEmptyLimit  = 10
)

// stuckWrapUpPrompt 卡住熔断后收尾调用注入的系统提示。
const stuckWrapUpPrompt = "本轮任务因循环卡住已中止：同一工具被反复调用，或连续多轮没有新进展，无法再调用任何工具。" +
	"请基于目前已经获得的信息直接给出阶段性总结：已完成什么、卡住在哪里、用户接下来可以怎么做。"

// resolveStuckLimit 将配置值规范为实际阈值：0→默认，<0→0（禁用）。
func resolveStuckLimit(v, fallback int) int {
	if v == 0 {
		return fallback
	}
	if v < 0 {
		return 0
	}
	return v
}

// stuckTracker 检测两类空转：
//   - 同一工具 + 同一参数连续调用（跨步、步内都计数）；
//   - 连续「无可见正文且工具组合不变」的空轮次。
//
// 空轮次不把「无正文 + 换了工具」算卡住：很多模型会连续静默调工具，那是正常推进。
type stuckTracker struct {
	repeatLimit int
	emptyLimit  int

	lastFP      string
	repeatCount int

	lastEmptyKey string
	emptyStreak  int
}

func newStuckTracker(repeatLimit, emptyLimit int) *stuckTracker {
	return &stuckTracker{repeatLimit: repeatLimit, emptyLimit: emptyLimit}
}

// noteEmptyTurn 记录本步是否为空轮次。hasReasoning 为 true 时视为有可见产出，不计入空轮。
// 达到阈值时返回面向用户的熔断说明，否则返回空串。
func (t *stuckTracker) noteEmptyTurn(content string, calls []llmchat.ToolCall, hasReasoning bool) string {
	if t.emptyLimit <= 0 {
		return ""
	}
	empty := strings.TrimSpace(content) == "" && !hasReasoning && len(calls) > 0
	if !empty {
		t.emptyStreak = 0
		t.lastEmptyKey = ""
		return ""
	}
	key := stepFingerprint(calls)
	if key != "" && key == t.lastEmptyKey {
		t.emptyStreak++
	} else {
		t.emptyStreak = 1
		t.lastEmptyKey = key
	}
	if t.emptyStreak >= t.emptyLimit {
		return fmt.Sprintf("连续 %d 轮无正文且工具组合无变化", t.emptyStreak)
	}
	return ""
}

// noteCall 记录一次工具调用。达到重复阈值时返回 true（本次即为第 N 次，不应再执行）。
func (t *stuckTracker) noteCall(call llmchat.ToolCall) bool {
	if t.repeatLimit <= 0 {
		return false
	}
	fp := toolFingerprint(call.Name, call.Arguments)
	if fp == t.lastFP {
		t.repeatCount++
	} else {
		t.lastFP = fp
		t.repeatCount = 1
	}
	return t.repeatCount >= t.repeatLimit
}

func (t *stuckTracker) repeatNotice(call llmchat.ToolCall) string {
	name := strings.TrimSpace(call.Name)
	if name == "" {
		name = "unknown"
	}
	return fmt.Sprintf("同一工具被反复调用（%s × %d）", name, t.repeatCount)
}

// toolFingerprint 工具名 + 规范化参数，用于重复调用比对。
func toolFingerprint(name, args string) string {
	return strings.TrimSpace(name) + "\n" + canonicalizeArgs(args)
}

// stepFingerprint 本步全部工具调用的稳定指纹（排序后拼接），用于空轮次「组合是否变化」。
func stepFingerprint(calls []llmchat.ToolCall) string {
	if len(calls) == 0 {
		return ""
	}
	fps := make([]string, len(calls))
	for i, c := range calls {
		fps[i] = toolFingerprint(c.Name, c.Arguments)
	}
	sort.Strings(fps)
	return strings.Join(fps, "\x1e")
}

// canonicalizeArgs 把工具参数收成可比较的 JSON。
// 空串与 "{}" 视为相同；合法 JSON 会重排 object 键；非法 JSON 保留原文（坏参数修复轮也能对上）。
func canonicalizeArgs(args string) string {
	s := strings.TrimSpace(args)
	if s == "" || s == "{}" {
		return "{}"
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return s
	}
	return string(b)
}

func stuckToolResult(reason string) string {
	return fmt.Sprintf(`{"error":%q}`, reason+"。已中止后续工具执行。")
}
