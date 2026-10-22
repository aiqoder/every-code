package agents

import (
	"strings"
	"testing"
)

func TestAllStations(t *testing.T) {
	got := All()
	want := []string{"ade-advisor", "ade-coder", "ade-deploy"}
	if len(got) != len(want) {
		t.Fatalf("count: %d", len(got))
	}
	for i, agent := range got {
		if agent.ID != want[i] {
			t.Fatalf("agent %d: %+v", i, agent)
		}
		if agent.Name == "" || agent.MaxSteps < 1 || len(agent.Tools) == 0 {
			t.Fatalf("incomplete: %+v", agent)
		}
		if _, ok := ByID(agent.ID); !ok {
			t.Fatalf("missing id %s", agent.ID)
		}
	}
	create, ok := ByID("ade-advisor")
	if !ok || create.SystemPrompt == "" {
		t.Fatal("empty prompt")
	}
}

func TestPipelineAgents(t *testing.T) {
	got := Pipeline()
	want := []string{"ade-planner", "ade-reviewer"}
	if len(got) != len(want) {
		t.Fatalf("count: %d", len(got))
	}
	for i, agent := range got {
		if agent.ID != want[i] || agent.SystemPrompt == "" || agent.MaxSteps < 1 || len(agent.Tools) == 0 {
			t.Fatalf("pipeline %d: %+v", i, agent)
		}
		if _, ok := ByID(agent.ID); !ok {
			t.Fatalf("missing id %s", agent.ID)
		}
	}
}

func TestBuiltinToolsFollowWhitelist(t *testing.T) {
	for _, id := range []string{"ade-advisor", "ade-planner", "ade-deploy"} {
		agent, ok := ByID(id)
		if !ok || !hasTool(agent, "AskQuestion") {
			t.Fatalf("%s should allow AskQuestion", id)
		}
	}
	for _, id := range []string{"ade-coder", "ade-reviewer", EnvCheckID} {
		agent, ok := ByID(id)
		if !ok || hasTool(agent, "AskQuestion") {
			t.Fatalf("%s should not allow AskQuestion", id)
		}
	}
	coder, ok := ByID("ade-coder")
	if !ok || !hasTool(coder, "todowriter") || !hasTool(coder, "Task") {
		t.Fatal("execute should allow todowriter and Task")
	}
	agent, err := parse("---\nid: x\nname: y\nmaxSteps: 1\ntools: read, AskQuestion\n---\n{{workspace}}\n")
	if err != nil || !hasTool(agent, "AskQuestion") {
		t.Fatalf("parse builtin: %+v err=%v", agent, err)
	}
	if _, err := parse("---\nid: x\nname: y\nmaxSteps: 1\ntools: nope\n---\n{{workspace}}\n"); err == nil {
		t.Fatal("unknown tool should fail")
	}
}

func hasTool(agent Agent, name string) bool {
	for _, tool := range agent.Tools {
		if tool == name {
			return true
		}
	}
	return false
}

func TestEnvCheckAgentIsSeparate(t *testing.T) {
	agent, ok := ByID(EnvCheckID)
	if !ok {
		t.Fatal("missing env check agent")
	}
	for _, row := range All() {
		if row.ID == EnvCheckID {
			t.Fatal("env check should not be a station")
		}
	}
	for _, row := range Pipeline() {
		if row.ID == EnvCheckID {
			t.Fatal("env check should not be a pipeline agent")
		}
	}
	tools := map[string]struct{}{}
	for _, name := range agent.Tools {
		tools[name] = struct{}{}
	}
	if _, ok := tools["shell_exec"]; !ok {
		t.Fatal("env check needs shell_exec")
	}
	create, ok := ByID("ade-advisor")
	if !ok {
		t.Fatal("missing planner")
	}
	for _, name := range create.Tools {
		if name == "shell_exec" {
			t.Fatal("task planning should not have shell_exec")
		}
	}
	if strings.Contains(create.SystemPrompt, "save_env_check") || strings.Contains(create.SystemPrompt, "shell_exec") {
		t.Fatal("task planning prompt should not mention env check tools")
	}
}
