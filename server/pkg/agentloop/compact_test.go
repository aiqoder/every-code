package agentloop

import (
	"context"
	"strings"
	"testing"

	"chengzi-agent-code-server/pkg/llmchat"
)

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Fatalf("empty: %d", got)
	}
	if got := EstimateTokens("你好"); got != 2 {
		t.Fatalf("cjk: %d", got)
	}
	if got := EstimateTokens("abcd"); got != 1 {
		t.Fatalf("ascii: %d", got)
	}
	if got := EstimateTokens("你a"); got != 2 {
		t.Fatalf("mixed: %d", got)
	}
}

func TestCapToolResultKeepsHeadAndTail(t *testing.T) {
	body := strings.Repeat("测", 5000)
	out := CapToolResult(body, 200)
	if EstimateTokens(out) > 220 {
		t.Fatalf("still too large: %d", EstimateTokens(out))
	}
	if !strings.Contains(out, "工具结果已截断") {
		t.Fatalf("missing marker: %s", out[:min(80, len(out))])
	}
	if !strings.HasPrefix(out, "测") {
		t.Fatal("lost head")
	}
	if !strings.HasSuffix(out, "测") {
		t.Fatal("lost tail")
	}
}

func TestPruneKeepsToolCallPairs(t *testing.T) {
	msgs := []llmchat.Message{
		{Role: llmchat.RoleSystem, Content: "sys"},
		{Role: llmchat.RoleUser, Content: strings.Repeat("旧", 400)},
		{Role: llmchat.RoleAssistant, Content: "call", ToolCalls: []llmchat.ToolCall{{ID: "c1", Name: "read", Arguments: "{}"}}},
		{Role: llmchat.RoleTool, Content: strings.Repeat("x", 400), ToolCallID: "c1", Name: "read"},
		{Role: llmchat.RoleUser, Content: "最近问题"},
		{Role: llmchat.RoleAssistant, Content: "答"},
	}
	out := pruneTurns(msgs, 80, 0, 1)
	if len(out) < 2 {
		t.Fatalf("too few: %+v", out)
	}
	ids := map[string]bool{}
	for _, m := range out {
		for _, tc := range m.ToolCalls {
			ids[tc.ID] = true
		}
	}
	for _, m := range out {
		if m.Role != llmchat.RoleTool {
			continue
		}
		if !ids[m.ToolCallID] {
			t.Fatalf("orphan tool result %s", m.ToolCallID)
		}
	}
	foundRecent := false
	for _, m := range out {
		if m.Role == llmchat.RoleUser && m.Content == "最近问题" {
			foundRecent = true
		}
	}
	if !foundRecent {
		t.Fatalf("dropped recent user: %+v", rolesOf(out))
	}
}

func TestPrepareSummarizesOldTurns(t *testing.T) {
	sum := &stubSummarizer{text: "用户要修登录；已改 auth.go。"}
	mgr := newContextManager(ContextPolicy{
		InputLimit:       2500,
		OutputReserve:    400,
		CompactThreshold: 0.5,
		KeepRecentTurns:  2,
		MaxToolResultTok: 8000,
	}, nil, sum)

	msgs := []llmchat.Message{
		{Role: llmchat.RoleSystem, Content: "你是助手"},
	}
	for i := 0; i < 6; i++ {
		msgs = append(msgs,
			llmchat.Message{Role: llmchat.RoleUser, Content: strings.Repeat("问", 300)},
			llmchat.Message{Role: llmchat.RoleAssistant, Content: strings.Repeat("答", 300)},
		)
	}

	out, err := mgr.Prepare(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Compacted {
		t.Fatalf("expected compact, used=%d limit=%d actions=%v", out.Usage.Used, out.Usage.Limit, out.Actions)
	}
	joined := ""
	for _, m := range out.Messages {
		joined += m.Content
	}
	if !strings.Contains(joined, "【对话摘要】") || !strings.Contains(joined, "auth.go") {
		t.Fatalf("missing summary: %s", joined[:min(200, len(joined))])
	}
	if sum.calls != 1 {
		t.Fatalf("summarize calls=%d", sum.calls)
	}

	// 同前缀再 Prepare 应复用缓存
	msgs = append(msgs, llmchat.Message{Role: llmchat.RoleAssistant, Content: "续"})
	out2, err := mgr.Prepare(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}
	if sum.calls != 1 {
		t.Fatalf("cache missed, calls=%d compacted=%v", sum.calls, out2.Actions)
	}
}

func TestMaxOutputUnlimitedWhenOutputReserveZero(t *testing.T) {
	mgr := newContextManager(ContextPolicy{
		InputLimit:    200_000,
		OutputReserve: 0,
	}, nil, nil)
	if got := mgr.maxOutputFor(5000); got != 0 {
		t.Fatalf("unlimited output should not set MaxTokens, got %d", got)
	}
	mgr2 := newContextManager(ContextPolicy{
		InputLimit:    200_000,
		OutputReserve: 4096,
	}, nil, nil)
	if got := mgr2.maxOutputFor(5000); got != 4096 {
		t.Fatalf("explicit output cap: got %d", got)
	}
}

func TestPrepareBelowThresholdDoesNotSummarize(t *testing.T) {
	sum := &stubSummarizer{text: "不该出现"}
	mgr := newContextManager(ContextPolicy{
		InputLimit:       50_000,
		OutputReserve:    1000,
		KeepRecentTurns:  4,
		MaxToolResultTok: 8000,
	}, nil, sum)
	msgs := []llmchat.Message{
		{Role: llmchat.RoleSystem, Content: "sys"},
		{Role: llmchat.RoleUser, Content: "hi"},
		{Role: llmchat.RoleAssistant, Content: "hello"},
	}
	out, err := mgr.Prepare(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}
	if sum.calls != 0 {
		t.Fatal("summarized small prompt")
	}
	if out.Usage.Used <= 0 {
		t.Fatal("expected token usage")
	}
}

type stubSummarizer struct {
	text  string
	calls int
}

func (s *stubSummarizer) Summarize(context.Context, string) (string, error) {
	s.calls++
	return s.text, nil
}

func rolesOf(msgs []llmchat.Message) []llmchat.Role {
	out := make([]llmchat.Role, len(msgs))
	for i, m := range msgs {
		out[i] = m.Role
	}
	return out
}
