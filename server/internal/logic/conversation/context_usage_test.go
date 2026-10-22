package conversation

import (
	"testing"

	"chengzi-agent-code-server/internal/model"
)

func TestParseStoredUsage(t *testing.T) {
	if parseStoredUsage("") != nil {
		t.Fatal("empty should be nil")
	}
	if parseStoredUsage("{") != nil {
		t.Fatal("invalid json should be nil")
	}
	got := parseStoredUsage(`{"used":12,"limit":100}`)
	if got == nil || got["used"] != float64(12) {
		t.Fatalf("unexpected: %#v", got)
	}
}

func TestEstimateUsageFromMessages(t *testing.T) {
	if estimateUsageFromMessages(nil, 1000) != nil {
		t.Fatal("empty messages should be nil")
	}
	got := estimateUsageFromMessages([]model.Message{
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "您好，需要什么帮助？"},
	}, 1000)
	if got == nil {
		t.Fatal("expected estimate")
	}
	used, _ := got["used"].(int)
	if used <= 0 {
		t.Fatalf("used should be > 0, got %#v", got["used"])
	}
	if got["limit"] != int64(1000) {
		t.Fatalf("limit: %#v", got["limit"])
	}
}
