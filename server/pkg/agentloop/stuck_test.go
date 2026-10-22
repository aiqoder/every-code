package agentloop

import (
	"testing"

	"chengzi-agent-code-server/pkg/llmchat"
)

func TestCanonicalizeArgs(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "{}"},
		{"{}", "{}"},
		{"  {}  ", "{}"},
		{`{"b":2,"a":1}`, `{"a":1,"b":2}`},
		{"{invalid", "{invalid"},
	}
	for _, c := range cases {
		if got := canonicalizeArgs(c.in); got != c.want {
			t.Fatalf("canonicalizeArgs(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestToolFingerprintIgnoresArgKeyOrder(t *testing.T) {
	a := toolFingerprint("read", `{"path":"a","q":1}`)
	b := toolFingerprint("read", `{"q":1,"path":"a"}`)
	if a != b {
		t.Fatalf("fingerprints differ: %q vs %q", a, b)
	}
	if toolFingerprint("read", `{"path":"a"}`) == toolFingerprint("read", `{"path":"b"}`) {
		t.Fatal("different args should not match")
	}
}

func TestStuckTrackerRepeat(t *testing.T) {
	tr := newStuckTracker(3, 3)
	call := llmchat.ToolCall{Name: "read", Arguments: `{"path":"x"}`}
	if tr.noteCall(call) {
		t.Fatal("first call should not trip")
	}
	if tr.noteCall(call) {
		t.Fatal("second call should not trip")
	}
	if !tr.noteCall(call) {
		t.Fatal("third identical call should trip")
	}
	if got := tr.repeatNotice(call); got != "同一工具被反复调用（read × 3）" {
		t.Fatalf("notice=%q", got)
	}
}

func TestStuckTrackerRepeatResetsOnDifferentCall(t *testing.T) {
	tr := newStuckTracker(3, 3)
	a := llmchat.ToolCall{Name: "read", Arguments: `{"path":"a"}`}
	b := llmchat.ToolCall{Name: "read", Arguments: `{"path":"b"}`}
	tr.noteCall(a)
	tr.noteCall(a)
	if tr.noteCall(b) {
		t.Fatal("different args should reset streak")
	}
	if tr.noteCall(b) {
		t.Fatal("second b should not trip")
	}
	if !tr.noteCall(b) {
		t.Fatal("third b should trip")
	}
}

func TestStuckTrackerEmptySameSet(t *testing.T) {
	tr := newStuckTracker(3, 3)
	calls := []llmchat.ToolCall{
		{Name: "read", Arguments: `{"path":"a"}`},
		{Name: "grep", Arguments: `{"q":"x"}`},
	}
	// 顺序不同也算同一组合。
	rev := []llmchat.ToolCall{calls[1], calls[0]}
	if msg := tr.noteEmptyTurn("", calls, false); msg != "" {
		t.Fatalf("first empty: %q", msg)
	}
	if msg := tr.noteEmptyTurn("", rev, false); msg != "" {
		t.Fatalf("second empty: %q", msg)
	}
	if msg := tr.noteEmptyTurn("", calls, false); msg == "" {
		t.Fatal("third identical empty turn should trip")
	}
}

func TestStuckTrackerEmptyResetsOnNewToolsOrText(t *testing.T) {
	tr := newStuckTracker(3, 3)
	a := []llmchat.ToolCall{{Name: "read", Arguments: `{"path":"a"}`}}
	b := []llmchat.ToolCall{{Name: "read", Arguments: `{"path":"b"}`}}
	tr.noteEmptyTurn("", a, false)
	tr.noteEmptyTurn("", a, false)
	if msg := tr.noteEmptyTurn("", b, false); msg != "" {
		t.Fatalf("changed tools should reset: %q", msg)
	}
	tr.noteEmptyTurn("", b, false)
	if msg := tr.noteEmptyTurn("我来读一下", b, false); msg != "" {
		t.Fatalf("text should reset: %q", msg)
	}
	if msg := tr.noteEmptyTurn("", b, false); msg != "" {
		t.Fatalf("after text, first empty should not trip: %q", msg)
	}
}

func TestStuckTrackerEmptyIgnoresReasoningOnly(t *testing.T) {
	tr := newStuckTracker(3, 3)
	calls := []llmchat.ToolCall{{Name: "read", Arguments: `{}`}}
	for i := 0; i < 5; i++ {
		if msg := tr.noteEmptyTurn("", calls, true); msg != "" {
			t.Fatalf("reasoning-only turn %d should not count as empty: %q", i+1, msg)
		}
	}
}

func TestStuckTrackerDisabled(t *testing.T) {
	tr := newStuckTracker(0, 0)
	call := llmchat.ToolCall{Name: "read", Arguments: `{}`}
	for i := 0; i < 5; i++ {
		if tr.noteCall(call) {
			t.Fatal("disabled repeat should not trip")
		}
		if msg := tr.noteEmptyTurn("", []llmchat.ToolCall{call}, false); msg != "" {
			t.Fatal("disabled empty should not trip")
		}
	}
}

func TestResolveStuckLimit(t *testing.T) {
	if got := resolveStuckLimit(0, 10); got != 10 {
		t.Fatalf("0 → default, got %d", got)
	}
	if got := resolveStuckLimit(-1, 10); got != 0 {
		t.Fatalf("-1 → disabled, got %d", got)
	}
	if got := resolveStuckLimit(5, 10); got != 5 {
		t.Fatalf("5 → 5, got %d", got)
	}
}
