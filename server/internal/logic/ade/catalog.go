package ade

import (
	"sort"
	"strings"
)

// TechOption 一项可选技术。
type TechOption struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind"` // frontend | backend | tool
	Probe   string `json:"probe"` // 探测命令
	Skill   string `json:"skill"` // 绑定的系统技能 id
	Summary string `json:"summary"`
}

// ScenarioOption 预设场景，点一下勾好一组技术。
type ScenarioOption struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Summary      string   `json:"summary"`
	Technologies []string `json:"technologies"`
}

// TechnologyCatalog 技术清单（内置，桌面只展示）。
func TechnologyCatalog() []TechOption {
	return []TechOption{
		{ID: "go", Label: "Go", Kind: "backend", Probe: "go version", Skill: "go-service", Summary: "高性能后端与命令行服务"},
		{ID: "php", Label: "PHP", Kind: "backend", Probe: "php -v", Skill: "php-service", Summary: "传统 Web 后端，部署简单"},
		{ID: "java", Label: "Java", Kind: "backend", Probe: "java -version", Skill: "java-service", Summary: "企业级后端与服务端应用"},
		{ID: "node", Label: "Node.js", Kind: "backend", Probe: "node -v", Skill: "node-service", Summary: "脚本化服务与前端构建链"},
		{ID: "vue", Label: "Vue", Kind: "frontend", Probe: "node -v", Skill: "vue-app", Summary: "Vue 3 单页应用"},
		{ID: "html", Label: "原生 HTML", Kind: "frontend", Probe: "", Skill: "html-app", Summary: "零构建的静态页面"},
		{ID: "miniprogram", Label: "微信开发者工具", Kind: "tool", Probe: "wechat-devtools", Skill: "wechat-miniprogram", Summary: "微信小程序工程与预览"},
	}
}

// ScenarioCatalog 预设场景。
func ScenarioCatalog() []ScenarioOption {
	return []ScenarioOption{
		{ID: "miniprogram", Label: "小程序", Summary: "微信小程序工程，Node 构建，开发者工具预览", Technologies: []string{"node", "miniprogram"}},
		{ID: "go-vue", Label: "Go + Vue", Summary: "常用前后端分离组合", Technologies: []string{"go", "vue"}},
		{ID: "php-vue", Label: "PHP + Vue", Summary: "传统后端配现代化前端", Technologies: []string{"php", "vue"}},
		{ID: "java-vue", Label: "Java + Vue", Summary: "企业级后端配前端", Technologies: []string{"java", "vue"}},
		{ID: "php-html", Label: "PHP + 原生 HTML", Summary: "轻量传统网页", Technologies: []string{"php", "html"}},
		{ID: "go-html", Label: "Go + 原生 HTML", Summary: "单二进制服务带静态页", Technologies: []string{"go", "html"}},
		{ID: "java-html", Label: "Java + 原生 HTML", Summary: "服务端渲染的简单站点", Technologies: []string{"java", "html"}},
		{ID: "frontend-only", Label: "只做前端", Summary: "纯静态或纯前端工程", Technologies: []string{"vue"}},
	}
}

// NormalizeTechs 去重、按清单顺序排序、丢弃未知项。
func NormalizeTechs(in []string) []string {
	catalog := TechnologyCatalog()
	known := make(map[string]int, len(catalog))
	for i, tech := range catalog {
		known[tech.ID] = i
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		id := strings.ToLower(strings.TrimSpace(raw))
		if id == "" {
			continue
		}
		if _, ok := known[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.SliceStable(out, func(i, j int) bool { return known[out[i]] < known[out[j]] })
	return out
}

// SplitTechs 拆分落库的逗号字符串。
func SplitTechs(raw string) []string {
	parts := strings.Split(raw, ",")
	return NormalizeTechs(parts)
}

// TechLabels 把技术 id 转成展示名，用于系统提示词。
func TechLabels(ids []string) []string {
	catalog := TechnologyCatalog()
	label := make(map[string]string, len(catalog))
	for _, t := range catalog {
		label[t.ID] = t.Label
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if v, ok := label[id]; ok {
			out = append(out, v)
		}
	}
	return out
}

// ProbeCommands 返回本项目勾选技术的固定探测命令。
func ProbeCommands(ids []string) []EnvCheck {
	catalog := TechnologyCatalog()
	byID := make(map[string]TechOption, len(catalog))
	for _, t := range catalog {
		byID[t.ID] = t
	}
	out := make([]EnvCheck, 0, len(ids))
	seen := make(map[string]struct{})
	for _, id := range ids {
		t, ok := byID[id]
		if !ok {
			continue
		}
		if t.Probe == "" {
			out = append(out, EnvCheck{Tech: t.ID, Label: t.Label, Installed: true, Version: "浏览器", Command: ""})
			continue
		}
		if _, dup := seen[t.Probe]; dup {
			continue
		}
		seen[t.Probe] = struct{}{}
		out = append(out, EnvCheck{Tech: t.ID, Label: t.Label, Command: t.Probe})
	}
	return out
}
