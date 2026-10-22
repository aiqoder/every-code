package agentloop

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"

	"chengzi-agent-code-server/pkg/llmchat"
	"chengzi-agent-code-server/pkg/tokencount"
)

const (
	defaultOutputReserve    int64   = 8192
	defaultCompactRatio     float64 = 0.85
	defaultKeepRecentTurns          = 4
	defaultMaxToolResultTok         = 8000
	minSummarizeTokens              = 800
	minOutputTokens         int64   = 256
	summarizerMaxTokens     int64   = 2048
)

// ContextPolicy 上下文窗口策略。InputLimit=0 时不做超窗压缩，仍会截断过大的工具结果。
type ContextPolicy struct {
	InputLimit       int64
	OutputReserve    int64
	CompactThreshold float64
	KeepRecentTurns  int
	MaxToolResultTok int
}

func (p ContextPolicy) withDefaults() ContextPolicy {
	if p.CompactThreshold <= 0 || p.CompactThreshold >= 1 {
		p.CompactThreshold = defaultCompactRatio
	}
	if p.KeepRecentTurns <= 0 {
		p.KeepRecentTurns = defaultKeepRecentTurns
	}
	if p.MaxToolResultTok <= 0 {
		p.MaxToolResultTok = defaultMaxToolResultTok
	}
	return p
}

func (p ContextPolicy) outputReserve() int64 {
	p = p.withDefaults()
	reserve := p.OutputReserve
	if reserve <= 0 {
		reserve = defaultOutputReserve
	}
	if p.InputLimit > 0 && reserve >= p.InputLimit {
		reserve = p.InputLimit / 8
		if reserve < minOutputTokens {
			reserve = minOutputTokens
		}
	}
	return reserve
}

func (p ContextPolicy) budget() int64 {
	p = p.withDefaults()
	if p.InputLimit <= 0 {
		return 0
	}
	b := p.InputLimit - p.outputReserve()
	if b < minOutputTokens {
		return p.InputLimit
	}
	return b
}

func (p ContextPolicy) maxToolResultTokens() int {
	p = p.withDefaults()
	max := p.MaxToolResultTok
	if p.InputLimit > 0 {
		capAt := int(p.InputLimit / 10)
		if capAt < 1024 {
			capAt = 1024
		}
		if max > capAt {
			max = capAt
		}
	}
	return max
}

func (p ContextPolicy) thresholdTokens() int {
	b := p.budget()
	if b <= 0 {
		return 0
	}
	return int(float64(b) * p.withDefaults().CompactThreshold)
}

// UsageBucket 用量分桶（系统 / 工具 / 技能 / 对话）。
type UsageBucket struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Tokens int64  `json:"tokens"`
}

// UsageSnapshot 一次模型调用前的上下文占用。
type UsageSnapshot struct {
	Limit     int64         `json:"limit"`
	Used      int64         `json:"used"`
	Remaining int64         `json:"remaining"`
	Ratio     float64       `json:"ratio"`
	Buckets   []UsageBucket `json:"buckets,omitempty"`
}

func (u UsageSnapshot) toData(compacted bool, actions []string) map[string]any {
	data := map[string]any{
		"limit":     u.Limit,
		"used":      u.Used,
		"remaining": u.Remaining,
		"ratio":     u.Ratio,
		"compacted": compacted,
	}
	if len(u.Buckets) > 0 {
		data["buckets"] = u.Buckets
	}
	if len(actions) > 0 {
		data["actions"] = actions
	}
	return data
}

// CompactOutcome 压缩后的请求消息与占用。
type CompactOutcome struct {
	Messages  []llmchat.Message
	Usage     UsageSnapshot
	MaxOutput int64
	Compacted bool
	Actions   []string
}

type summarizer interface {
	Summarize(ctx context.Context, transcript string) (string, error)
}

type contextManager struct {
	policy    ContextPolicy
	toolsTok  int
	summarize summarizer

	cachedFP      uint64
	cachedSummary string
}

func newContextManager(policy ContextPolicy, tools []llmchat.ToolDef, sum summarizer) *contextManager {
	policy = policy.withDefaults()
	return &contextManager{
		policy:    policy,
		toolsTok:  EstimateTools(tools),
		summarize: sum,
	}
}

func (m *contextManager) snapshot(msgs []llmchat.Message) UsageSnapshot {
	b := tokencount.SplitPrompt(msgs, m.toolsTok)
	used := int64(b.Used)
	limit := m.policy.InputLimit
	var remaining int64
	var ratio float64
	if limit > 0 {
		remaining = limit - used
		if remaining < 0 {
			remaining = 0
		}
		ratio = float64(used) / float64(limit)
		if ratio > 1 {
			ratio = 1
		}
	}
	return UsageSnapshot{
		Limit:     limit,
		Used:      used,
		Remaining: remaining,
		Ratio:     ratio,
		Buckets: []UsageBucket{
			{Key: "system", Label: "系统提示词", Tokens: int64(b.System)},
			{Key: "tools", Label: "工具", Tokens: int64(b.Tools)},
			{Key: "skills", Label: "技能", Tokens: int64(b.Skills)},
			{Key: "messages", Label: "对话消息", Tokens: int64(b.Messages)},
		},
	}
}

func (m *contextManager) maxOutputFor(used int64) int64 {
	// OutputContextLimit=0 表示不限制：不写死 MaxTokens，避免把整页 HTML/长代码截成空回复。
	if m.policy.OutputReserve <= 0 {
		return 0
	}
	reserve := m.policy.outputReserve()
	if m.policy.InputLimit > 0 {
		remain := m.policy.InputLimit - used
		if remain < reserve {
			reserve = remain
		}
	}
	if reserve < minOutputTokens {
		reserve = minOutputTokens
	}
	return reserve
}

// Prepare 在不改动原始历史的前提下，生成送给模型的消息副本。
// 分层：工具结果截断 → 旧轮摘要 → 按 user-turn 裁剪 → 硬截断。
func (m *contextManager) Prepare(ctx context.Context, messages []llmchat.Message) (CompactOutcome, error) {
	out := cloneMessages(messages)
	actions := make([]string, 0, 3)

	maxTool := m.policy.maxToolResultTokens()
	if truncateToolResults(out, maxTool) {
		actions = append(actions, "truncate_tools")
	}

	used := EstimatePromptTokens(out, m.toolsTok)
	threshold := m.policy.thresholdTokens()
	budget := int(m.policy.budget())

	needWindowCompact := m.policy.InputLimit > 0 && threshold > 0 && used > threshold
	if needWindowCompact {
		summarized, ok, err := m.trySummarize(ctx, out)
		if err != nil {
			return CompactOutcome{}, err
		}
		if ok {
			out = summarized
			actions = append(actions, "summarize")
			used = EstimatePromptTokens(out, m.toolsTok)
		}
		if budget > 0 && used > budget {
			out = pruneTurns(out, budget, m.toolsTok, m.policy.KeepRecentTurns)
			actions = append(actions, "prune")
			used = EstimatePromptTokens(out, m.toolsTok)
		}
		if budget > 0 && used > budget {
			out = shrinkContents(out, budget, m.toolsTok, maxTool)
			actions = append(actions, "shrink")
		}
	}

	usage := m.snapshot(out)
	return CompactOutcome{
		Messages:  out,
		Usage:     usage,
		MaxOutput: m.maxOutputFor(usage.Used),
		Compacted: len(actions) > 0,
		Actions:   actions,
	}, nil
}

func (m *contextManager) trySummarize(ctx context.Context, msgs []llmchat.Message) ([]llmchat.Message, bool, error) {
	if m.summarize == nil {
		return msgs, false, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	sys, rest := splitLeadingSystem(msgs)
	turns := groupTurns(rest)
	keep := m.policy.KeepRecentTurns
	if keep < 1 {
		keep = 1
	}
	if len(turns) <= keep {
		return msgs, false, nil
	}
	oldTurns := turns[:len(turns)-keep]
	recentTurns := turns[len(turns)-keep:]
	oldMsgs := concatTurns(oldTurns)
	if EstimateMessages(oldMsgs) < minSummarizeTokens {
		return msgs, false, nil
	}

	fp := fingerprintMessages(oldMsgs)
	summary := ""
	if fp == m.cachedFP && m.cachedSummary != "" {
		summary = m.cachedSummary
	} else {
		transcript := capTranscript(renderTranscript(oldMsgs), m.summarizerInputBudget())
		var err error
		summary, err = m.summarize.Summarize(ctx, transcript)
		if err != nil {
			if ctx.Err() != nil {
				return nil, false, err
			}
			return msgs, false, nil
		}
		summary = strings.TrimSpace(summary)
		if summary == "" {
			return msgs, false, nil
		}
		m.cachedFP = fp
		m.cachedSummary = summary
	}

	merged := mergeSummaryIntoSystem(sys, summary)
	out := append(merged, concatTurns(recentTurns)...)
	return out, true, nil
}

func (m *contextManager) summarizerInputBudget() int {
	b := int(m.policy.budget())
	if b <= 0 {
		return 32_000
	}
	capAt := b / 2
	if capAt > 32_000 {
		capAt = 32_000
	}
	if capAt < 2000 {
		capAt = 2000
	}
	return capAt
}

func mergeSummaryIntoSystem(sys []llmchat.Message, summary string) []llmchat.Message {
	block := formatSummary(summary)
	if len(sys) == 0 {
		return []llmchat.Message{{Role: llmchat.RoleSystem, Content: block}}
	}
	out := cloneMessages(sys)
	out[0].Content = strings.TrimSpace(out[0].Content) + "\n\n" + block
	return out
}

func formatSummary(summary string) string {
	return "【对话摘要】以下为较早轮次的压缩记忆，细节可能被省略。请以此为背景，优先采信后续完整消息。\n\n" + strings.TrimSpace(summary)
}

func pruneTurns(msgs []llmchat.Message, budget, toolsTok, minKeep int) []llmchat.Message {
	sys, rest := splitLeadingSystem(msgs)
	turns := groupTurns(rest)
	if len(turns) == 0 {
		return msgs
	}
	keep := minKeep
	if keep < 1 {
		keep = 1
	}
	if keep > len(turns) {
		keep = len(turns)
	}
	for keep >= 1 {
		candidate := append(cloneMessages(sys), concatTurns(turns[len(turns)-keep:])...)
		if EstimatePromptTokens(candidate, toolsTok) <= budget {
			return candidate
		}
		keep--
	}
	return append(cloneMessages(sys), concatTurns(turns[len(turns)-1:])...)
}

func shrinkContents(msgs []llmchat.Message, budget, toolsTok, maxTool int) []llmchat.Message {
	out := cloneMessages(msgs)
	for _, div := range []int{2, 4, 8} {
		capAt := maxTool / div
		if capAt < 256 {
			capAt = 256
		}
		truncateToolResults(out, capAt)
		if EstimatePromptTokens(out, toolsTok) <= budget {
			return out
		}
	}
	const bodyCap = 512
	for i := range out {
		if out[i].Role == llmchat.RoleSystem {
			continue
		}
		if EstimateTokens(out[i].Content) > bodyCap {
			out[i].Content = takePrefixByTokens(out[i].Content, bodyCap) + "\n…[已截断]"
		}
		for j := range out[i].ToolCalls {
			if EstimateTokens(out[i].ToolCalls[j].Arguments) > bodyCap {
				out[i].ToolCalls[j].Arguments = takePrefixByTokens(out[i].ToolCalls[j].Arguments, bodyCap)
			}
		}
	}
	return out
}

func truncateToolResults(msgs []llmchat.Message, maxTok int) bool {
	if maxTok <= 0 {
		return false
	}
	changed := false
	for i := range msgs {
		if msgs[i].Role != llmchat.RoleTool {
			continue
		}
		next, ok := truncateMiddle(msgs[i].Content, maxTok, true)
		if ok {
			msgs[i].Content = next
			changed = true
		}
	}
	return changed
}

func truncateMiddle(s string, maxTok int, toolResult bool) (string, bool) {
	if maxTok <= 0 || EstimateTokens(s) <= maxTok {
		return s, false
	}
	orig := EstimateTokens(s)
	marker := fmt.Sprintf("\n\n[内容已截断：原文约 %d tokens，已保留开头与结尾]\n\n", orig)
	if toolResult {
		marker = fmt.Sprintf("\n\n[工具结果已截断：原文约 %d tokens，已保留开头与结尾]\n\n", orig)
	}
	markerTok := EstimateTokens(marker)
	if maxTok <= markerTok+32 {
		return takePrefixByTokens(s, maxTok), true
	}
	remain := maxTok - markerTok
	headBudget := remain * 3 / 4
	if headBudget < 16 {
		headBudget = remain / 2
	}
	head := takePrefixByTokens(s, headBudget)
	tailBudget := remain - EstimateTokens(head)
	if tailBudget < 8 {
		return head + marker, true
	}
	tail := takeSuffixByTokens(s, tailBudget)
	if len(head)+len(tail) >= len(s) {
		return s, false
	}
	return head + marker + tail, true
}

// CapToolResult 截断单条工具结果，供 loop 回填模型与落库时使用。
func CapToolResult(result string, maxTok int) string {
	if maxTok <= 0 {
		maxTok = defaultMaxToolResultTok
	}
	out, _ := truncateMiddle(result, maxTok, true)
	return out
}

func splitLeadingSystem(msgs []llmchat.Message) (sys, rest []llmchat.Message) {
	i := 0
	for i < len(msgs) && msgs[i].Role == llmchat.RoleSystem {
		i++
	}
	return msgs[:i], msgs[i:]
}

func groupTurns(msgs []llmchat.Message) [][]llmchat.Message {
	turns := make([][]llmchat.Message, 0, 8)
	var cur []llmchat.Message
	flush := func() {
		if len(cur) == 0 {
			return
		}
		turns = append(turns, cur)
		cur = nil
	}
	for _, m := range msgs {
		if m.Role == llmchat.RoleUser && len(cur) > 0 {
			flush()
		}
		cur = append(cur, m)
	}
	flush()
	return turns
}

func concatTurns(turns [][]llmchat.Message) []llmchat.Message {
	n := 0
	for _, t := range turns {
		n += len(t)
	}
	out := make([]llmchat.Message, 0, n)
	for _, t := range turns {
		out = append(out, t...)
	}
	return out
}

func cloneMessages(in []llmchat.Message) []llmchat.Message {
	out := make([]llmchat.Message, len(in))
	for i, m := range in {
		out[i] = m
		if len(m.ToolCalls) > 0 {
			out[i].ToolCalls = append([]llmchat.ToolCall(nil), m.ToolCalls...)
		}
	}
	return out
}

func fingerprintMessages(msgs []llmchat.Message) uint64 {
	h := fnv.New64a()
	for _, m := range msgs {
		_, _ = h.Write([]byte(m.Role))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(m.Content))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(m.ToolCallID))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(m.Name))
		for _, tc := range m.ToolCalls {
			_, _ = h.Write([]byte(tc.ID))
			_, _ = h.Write([]byte(tc.Name))
			_, _ = h.Write([]byte(tc.Arguments))
		}
		_, _ = h.Write([]byte{1})
	}
	return h.Sum64()
}

func renderTranscript(msgs []llmchat.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		switch m.Role {
		case llmchat.RoleUser:
			fmt.Fprintf(&b, "user: %s\n", m.Content)
		case llmchat.RoleAssistant:
			fmt.Fprintf(&b, "assistant: %s\n", m.Content)
			for _, tc := range m.ToolCalls {
				args := tc.Arguments
				if EstimateTokens(args) > 400 {
					args = takePrefixByTokens(args, 400) + "…"
				}
				fmt.Fprintf(&b, "assistant tool_call %s %s %s\n", tc.Name, tc.ID, args)
			}
		case llmchat.RoleTool:
			fmt.Fprintf(&b, "tool %s: %s\n", m.Name, m.Content)
		case llmchat.RoleSystem:
			fmt.Fprintf(&b, "system: %s\n", m.Content)
		}
	}
	return b.String()
}

func capTranscript(s string, maxTok int) string {
	if EstimateTokens(s) <= maxTok {
		return s
	}
	out, _ := truncateMiddle(s, maxTok, false)
	return out
}

func emitContextEvents(emit func(Event), outcome CompactOutcome) {
	data := outcome.Usage.toData(outcome.Compacted, outcome.Actions)
	emit(Event{Type: EventContextUsage, Data: data})
	if outcome.Compacted {
		emit(Event{Type: EventContextCompacted, Data: data})
	}
}
