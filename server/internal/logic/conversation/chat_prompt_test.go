package conversation

import "testing"

func TestAgentAllowsBuiltinByWhitelist(t *testing.T) {
	if agentAllowsTool("ade-coder", "AskQuestion") {
		t.Fatal("execute should not get AskQuestion")
	}
	if !agentAllowsTool("ade-advisor", "AskQuestion") {
		t.Fatal("create should get AskQuestion")
	}
	if !agentAllowsTool("ade-coder", "todowriter") || !agentAllowsTool("ade-coder", "Task") {
		t.Fatal("execute should allow todowriter and Task")
	}
	if agentAllowsTool("missing", "AskQuestion") {
		t.Fatal("unknown agent should not get builtins")
	}
}

func TestNormalizeOsInfo(t *testing.T) {
	got, err := normalizeOsInfo(" Windows-ARM64 ")
	if err != nil || got != "windows-arm64" {
		t.Fatalf("got %q err=%v", got, err)
	}
	if _, err := normalizeOsInfo(""); err == nil {
		t.Fatal("empty should fail")
	}
	if _, err := normalizeOsInfo("windows"); err == nil {
		t.Fatal("missing arch should fail")
	}
	if _, err := normalizeOsInfo("windows_arm64"); err == nil {
		t.Fatal("underscore should fail")
	}
}
