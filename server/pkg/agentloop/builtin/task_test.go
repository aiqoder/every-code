package builtin

import (
	"context"
	"strings"
	"testing"

	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/llmchat"
)

type namedTool struct{ name string }

func (t namedTool) Name() string               { return t.name }
func (t namedTool) Description() string        { return t.name }
func (t namedTool) Parameters() map[string]any { return nil }
func (t namedTool) Execute(context.Context, string, *agentloop.Runtime) (string, error) {
	return "", nil
}

func TestChildToolsExploreIsReadOnly(t *testing.T) {
	parent := []agentloop.Tool{
		namedTool{name: "read"},
		namedTool{name: "grep"},
		namedTool{name: "find"},
		namedTool{name: "write"},
		namedTool{name: "shell_exec"},
		namedTool{name: ToolTask},
	}
	got := childTools(parent, subagentExplore)
	if len(got) != 3 {
		t.Fatalf("explore tools=%d want 3", len(got))
	}
	for _, tool := range got {
		if _, ok := exploreToolNames[tool.Name()]; !ok {
			t.Fatalf("unexpected tool %s", tool.Name())
		}
	}
}

func TestChildToolsGeneralDropsTask(t *testing.T) {
	parent := []agentloop.Tool{
		namedTool{name: "read"},
		namedTool{name: "write"},
		namedTool{name: ToolTask},
	}
	got := childTools(parent, subagentGeneral)
	if len(got) != 2 {
		t.Fatalf("tools=%d want 2", len(got))
	}
	for _, tool := range got {
		if tool.Name() == ToolTask {
			t.Fatalf("task leaked: %s", tool.Name())
		}
	}
}

func TestTaskRejectsEmptyPrompt(t *testing.T) {
	_, err := NewTaskTool().Execute(context.Background(), `{"description":"看代码","prompt":"  "}`, &agentloop.Runtime{})
	if err == nil || !strings.Contains(err.Error(), "prompt") {
		t.Fatalf("err=%v", err)
	}
}

func TestTaskRejectsUnknownType(t *testing.T) {
	_, err := NewTaskTool().Execute(context.Background(), `{"description":"看代码","prompt":"找入口","subagent_type":"cloud"}`, &agentloop.Runtime{MaxDepth: 2})
	if err == nil || !strings.Contains(err.Error(), "generalPurpose") {
		t.Fatalf("err=%v", err)
	}
}

type textChat struct{}

func (textChat) Protocol() llmchat.Protocol { return llmchat.ProtocolOpenAICompatible }

func (textChat) ChatStream(_ context.Context, _ *llmchat.ChatRequest, onEvent llmchat.EventHandler) (*llmchat.AssistantTurn, error) {
	if onEvent != nil {
		_ = onEvent(llmchat.StreamEvent{Type: llmchat.StreamEventTextDelta, Text: "找到 main.go"})
	}
	return &llmchat.AssistantTurn{Content: "找到 main.go"}, nil
}

func TestTaskReturnsChildResult(t *testing.T) {
	var started, finished bool
	rt := &agentloop.Runtime{
		Depth:    0,
		MaxDepth: 2,
		Chat:     textChat{},
		Model:    "test",
		Tools:    []agentloop.Tool{namedTool{name: "read"}, namedTool{name: "write"}, NewTaskTool()},
		Emit: func(ev agentloop.Event) {
			switch ev.Type {
			case agentloop.EventSubagentStarted:
				started = true
				if ev.Data["agentId"] == "" || ev.Data["subagentType"] != subagentExplore {
					t.Fatalf("start data=%v", ev.Data)
				}
			case agentloop.EventSubagentFinished:
				finished = true
			}
		},
	}
	out, err := NewTaskTool().Execute(context.Background(), `{"description":"找入口","prompt":"main 在哪","subagent_type":"explore"}`, rt)
	if err != nil {
		t.Fatal(err)
	}
	if !started || !finished {
		t.Fatalf("started=%v finished=%v", started, finished)
	}
	if !strings.Contains(out, "找到 main.go") || !strings.Contains(out, `"subagentType":"explore"`) {
		t.Fatalf("out=%s", out)
	}
}
