package agentcatalog

import (
	"context"
	"strings"
	"testing"

	"chengzi-agent-code-server/pkg/clienthub/protocol"
)

func TestMaterializeUsesServerPersona(t *testing.T) {
	view, err := Materialize(context.Background(), nil, "ade-coder", protocol.CatalogPayload{
		Name:         "客户端名称",
		SystemPrompt: "客户端提示词",
		MaxSteps:     3,
		Workspace:    "/work/app",
		Tools: []protocol.ToolMeta{
			{Name: "read"},
			{Name: "shell_exec"},
			{Name: "write"},
			{Name: "page_state"},
		},
		PageChannel: "create",
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Name != "开发" || view.MaxSteps != 60 {
		t.Fatalf("persona: %+v", view)
	}
	if !strings.Contains(view.SystemPrompt, "/work/app") || strings.Contains(view.SystemPrompt, "客户端提示词") {
		t.Fatalf("prompt: %q", view.SystemPrompt)
	}
	if len(view.Tools) != 3 || view.Tools[0].Name != "read" || view.Tools[1].Name != "shell_exec" || view.Tools[2].Name != "write" {
		t.Fatalf("tools: %+v", view.Tools)
	}
}

func TestMaterializeDropsPageTools(t *testing.T) {
	view, err := Materialize(context.Background(), nil, "ade-advisor", protocol.CatalogPayload{
		Workspace: "/work/app",
		Tools: []protocol.ToolMeta{
			{Name: "read"},
			{Name: "write"},
			{Name: "page_state"},
			{Name: "page_save_plan"},
		},
		PageChannel: "create",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Tools) != 1 || view.Tools[0].Name != "read" {
		t.Fatalf("tools: %+v", view.Tools)
	}
	if strings.Contains(view.SystemPrompt, "page_") {
		t.Fatalf("prompt: %q", view.SystemPrompt)
	}
}

func TestMaterializePlannerIgnoresPageTools(t *testing.T) {
	view, err := Materialize(context.Background(), nil, "ade-planner", protocol.CatalogPayload{
		Workspace: "/work/app",
		Tools: []protocol.ToolMeta{
			{Name: "read"},
			{Name: "write"},
			{Name: "page_state"},
		},
		PageChannel: "create",
		Skills: []protocol.SkillMeta{
			{ID: "local-one", Name: "本机"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Name != "拆解器" || view.MaxSteps != 12 {
		t.Fatalf("persona: %+v", view)
	}
	if len(view.Tools) != 1 || view.Tools[0].Name != "read" {
		t.Fatalf("tools: %+v", view.Tools)
	}
	if len(view.Skills) != 1 || view.Skills[0].ID != "local-one" {
		t.Fatalf("skills: %+v", view.Skills)
	}
}

func TestMaterializeSkipsBuiltinFromClient(t *testing.T) {
	view, err := Materialize(context.Background(), nil, "ade-advisor", protocol.CatalogPayload{
		Workspace: "/work/app",
		Tools: []protocol.ToolMeta{
			{Name: "read"},
			{Name: "AskQuestion"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Tools) != 1 || view.Tools[0].Name != "read" {
		t.Fatalf("tools: %+v", view.Tools)
	}
}

func TestMaterializeRejectsUnknownAgent(t *testing.T) {
	_, err := Materialize(context.Background(), nil, "no-such", protocol.CatalogPayload{OsInfo: "darwin-arm64"})
	if err == nil {
		t.Fatal("expected error")
	}
}
