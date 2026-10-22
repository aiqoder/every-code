package tokencount

import (
	"testing"

	"chengzi-agent-code-server/pkg/llmchat"
)

func TestCountGolden(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"", 0},
		{"hello", 1},
		{"abcd", 1},
		{"你好", 2},
		{"你a", 2},
		{"function", 1},
		{"Available tools:", 3},
		{"当前工作目录：/tmp", 8},
	}
	for _, tc := range cases {
		if got := Count(tc.text); got != tc.want {
			t.Errorf("Count(%q)=%d want %d", tc.text, got, tc.want)
		}
	}
}

func TestCountMessageAndTools(t *testing.T) {
	msg := llmchat.Message{
		Role:    llmchat.RoleUser,
		Content: "你好",
	}
	// 3 + role(1) + content(2) = 6
	if got := CountMessage(msg); got != 6 {
		t.Fatalf("CountMessage=%d", got)
	}
	if got := CountMessages([]llmchat.Message{msg}); got != 9 {
		t.Fatalf("CountMessages=%d want 9 (6+priming3)", got)
	}

	tools := []llmchat.ToolDef{{
		Name:        "read",
		Description: "读取文件",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
		},
	}}
	toolsTok := CountTools(tools)
	if toolsTok != 20 {
		t.Fatalf("CountTools=%d want 20", toolsTok)
	}

	asst := llmchat.Message{
		Role:    llmchat.RoleAssistant,
		Content: "ok",
		ToolCalls: []llmchat.ToolCall{{
			ID:        "c1",
			Name:      "read",
			Arguments: `{"path":"a.go"}`,
		}},
	}
	// 3 + assistant(1) + ok(1) + 3 + c1(2) + read(1) + args(6) = 17
	if got := CountMessage(asst); got != 17 {
		t.Fatalf("CountMessage(tool_call)=%d want 17", got)
	}
	prompt := CountPrompt([]llmchat.Message{
		{Role: llmchat.RoleSystem, Content: "你是助手"},
		{Role: llmchat.RoleUser, Content: "你好"},
	}, toolsTok)
	if prompt != 38 {
		t.Fatalf("CountPrompt=%d want 38", prompt)
	}
}

func TestPrefixSuffix(t *testing.T) {
	s := "你好世界abcd"
	pre := Prefix(s, 2)
	if Count(pre) > 2 {
		t.Fatalf("prefix tokens=%d text=%q", Count(pre), pre)
	}
	suf := Suffix(s, 2)
	if Count(suf) > 2 {
		t.Fatalf("suffix tokens=%d text=%q", Count(suf), suf)
	}
	if Prefix(s, 100) != s || Suffix(s, 100) != s {
		t.Fatal("short text should be unchanged")
	}
}

func TestSplitPromptBuckets(t *testing.T) {
	sys := "你是助手\n\n" + BoundSkillsHeading + "\n\n- git: 提交\n\n" + AvailableToolsPrefix + "\n- read: 读"
	msgs := []llmchat.Message{
		{Role: llmchat.RoleSystem, Content: sys},
		{Role: llmchat.RoleUser, Content: "你好"},
	}
	b := SplitPrompt(msgs, 20)
	if b.Used != CountPrompt(msgs, 20) {
		t.Fatalf("used=%d prompt=%d", b.Used, CountPrompt(msgs, 20))
	}
	if b.System+b.Tools+b.Skills+b.Messages != b.Used {
		t.Fatalf("buckets %d+%d+%d+%d != %d", b.System, b.Tools, b.Skills, b.Messages, b.Used)
	}
	if b.Tools != 20 {
		t.Fatalf("tools=%d", b.Tools)
	}
	if b.Skills <= 0 {
		t.Fatal("expected skills bucket")
	}
	if b.System <= 0 || b.Messages <= 0 {
		t.Fatalf("system=%d messages=%d", b.System, b.Messages)
	}

	plain := SplitPrompt([]llmchat.Message{
		{Role: llmchat.RoleSystem, Content: "你是助手"},
		{Role: llmchat.RoleUser, Content: "hi"},
	}, 0)
	if plain.Skills != 0 || plain.Tools != 0 {
		t.Fatalf("plain skills=%d tools=%d", plain.Skills, plain.Tools)
	}
}
