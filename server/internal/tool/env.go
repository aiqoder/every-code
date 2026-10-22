package tool

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/agentloop"
	"chengzi-agent-code-server/pkg/timex"
)

type envItem struct {
	Tech      string `json:"tech"`
	Label     string `json:"label"`
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Command   string `json:"command"`
}

type saveEnvCheckTool struct {
	repo  repository.AdeRepository
	scope Scope
}

func (t *saveEnvCheckTool) Name() string { return SaveEnvCheck }

func (t *saveEnvCheckTool) Description() string {
	return "把本机开发环境检查结果写回当前项目。每项含 tech、label、installed、version、command。只保存，不安装。"
}

func (t *saveEnvCheckTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tech":      map[string]any{"type": "string"},
						"label":     map[string]any{"type": "string"},
						"installed": map[string]any{"type": "boolean"},
						"version":   map[string]any{"type": "string"},
						"command":   map[string]any{"type": "string"},
					},
					"required":             []string{"label", "installed"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"items"},
		"additionalProperties": false,
	}
}

func (t *saveEnvCheckTool) Execute(ctx context.Context, argsJSON string, _ *agentloop.Runtime) (string, error) {
	var args struct {
		Items []envItem `json:"items"`
	}
	if err := decode(argsJSON, &args); err != nil {
		return "", err
	}
	items := make([]envItem, 0, len(args.Items))
	for _, item := range args.Items {
		item.Tech = strings.TrimSpace(item.Tech)
		item.Label = strings.TrimSpace(item.Label)
		item.Version = strings.TrimSpace(item.Version)
		item.Command = strings.TrimSpace(item.Command)
		if item.Label == "" {
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return "", errf("没有可保存的检测结果")
	}
	p, err := t.scope.project(ctx, t.repo)
	if err != nil {
		return "", err
	}
	allOK := true
	for _, item := range items {
		if !item.Installed {
			allOK = false
		}
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	p.EnvCheck = string(raw)
	p.EnvReady = allOK
	p.UpdatedAt = timex.LocalTime(time.Now())
	if err := t.repo.UpdateProject(ctx, p); err != nil {
		return "", err
	}
	return okJSON(map[string]any{"projectId": p.ID, "envReady": p.EnvReady, "count": len(items)})
}
