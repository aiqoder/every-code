package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	SourceClaude  = "claude"
	SourceProject = "project"
)

// Skill 可绑定到智能体的技能元信息。
type Skill struct {
	ID          string `json:"id"`          // 唯一键，如 claude:git-commit
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"` // claude | project
	Path        string `json:"-"` // 技能目录或 SKILL.md 绝对路径
}

// ListAvailable 扫描 Claude Code skills 与项目同级 .skills，返回可用技能列表。
func ListAvailable() ([]Skill, error) {
	roots, err := skillRoots()
	if err != nil {
		return nil, err
	}
	var out []Skill
	seen := map[string]struct{}{}
	for _, root := range roots {
		list, err := scanRoot(root.source, root.dir)
		if err != nil {
			return nil, err
		}
		for _, s := range list {
			if _, ok := seen[s.ID]; ok {
				continue
			}
			seen[s.ID] = struct{}{}
			out = append(out, s)
		}
	}
	if out == nil {
		out = []Skill{}
	}
	return out, nil
}

// LoadContent 按技能 ID 读取 SKILL.md 全文；不存在返回错误。
func LoadContent(id string) (string, error) {
	list, err := ListAvailable()
	if err != nil {
		return "", err
	}
	for _, s := range list {
		if s.ID == id {
			b, err := os.ReadFile(s.Path)
			if err != nil {
				return "", fmt.Errorf("read skill %s: %w", id, err)
			}
			return string(b), nil
		}
	}
	return "", fmt.Errorf("skill not found: %s", id)
}

// ResolveByIDs 按 ID 列表返回技能元信息（frontmatter 的 name/description）；跳过不存在的 ID。
func ResolveByIDs(ids []string) ([]Skill, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	list, err := ListAvailable()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Skill, len(list))
	for _, s := range list {
		byID[s.ID] = s
	}
	out := make([]Skill, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		s, ok := byID[id]
		if !ok {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// FormatAvailableLine 格式化技能/工具摘要行（仅 name + description）。
func FormatAvailableLine(name, description string) string {
	description = strings.TrimSpace(description)
	return fmt.Sprintf("- %s: %s", name, description)
}

// LoadContents 按 ID 列表读取内容；跳过不存在的 ID。
func LoadContents(ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	list, err := ListAvailable()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Skill, len(list))
	for _, s := range list {
		byID[s.ID] = s
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		s, ok := byID[id]
		if !ok {
			continue
		}
		b, err := os.ReadFile(s.Path)
		if err != nil {
			return nil, fmt.Errorf("read skill %s: %w", id, err)
		}
		out = append(out, string(b))
	}
	return out, nil
}

// ValidateIDs 校验技能 ID 均存在于可用列表中。
func ValidateIDs(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	list, err := ListAvailable()
	if err != nil {
		return err
	}
	allowed := make(map[string]struct{}, len(list))
	for _, s := range list {
		allowed[s.ID] = struct{}{}
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := allowed[id]; !ok {
			return fmt.Errorf("unknown skill: %s", id)
		}
	}
	return nil
}

type skillRoot struct {
	source string
	dir    string
}

func skillRoots() ([]skillRoot, error) {
	return claudeCodeRoots(""), nil
}

// ListClaudeCode 扫描 Claude Code 用户技能与工作目录项目技能。
// workspace 为空时只扫 ~/.claude/skills。
func ListClaudeCode(workspace string) ([]Skill, error) {
	roots := claudeCodeRoots(workspace)
	var out []Skill
	seen := map[string]struct{}{}
	for _, root := range roots {
		list, err := scanRoot(root.source, root.dir)
		if err != nil {
			return nil, err
		}
		for _, s := range list {
			if _, ok := seen[s.ID]; ok {
				continue
			}
			seen[s.ID] = struct{}{}
			out = append(out, s)
		}
	}
	if out == nil {
		out = []Skill{}
	}
	return out, nil
}

func claudeCodeRoots(workspace string) []skillRoot {
	var roots []skillRoot
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, skillRoot{
			source: SourceClaude,
			dir:    filepath.Join(home, ".claude", "skills"),
		})
	}
	ws := strings.TrimSpace(workspace)
	if ws != "" {
		roots = append(roots, skillRoot{
			source: SourceProject,
			dir:    filepath.Join(ws, ".claude", "skills"),
		})
	}
	return roots
}

func scanRoot(source, dir string) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skills dir %s: %w", dir, err)
	}
	out := make([]Skill, 0)
	for _, ent := range entries {
		if !ent.IsDir() || strings.HasPrefix(ent.Name(), ".") {
			continue
		}
		skillMD := filepath.Join(dir, ent.Name(), "SKILL.md")
		info, err := os.Stat(skillMD)
		if err != nil || info.IsDir() {
			continue
		}
		b, err := os.ReadFile(skillMD)
		if err != nil {
			continue
		}
		name, desc := ParseFrontmatter(string(b))
		if name == "" {
			name = ent.Name()
		}
		out = append(out, Skill{
			ID:          source + ":" + ent.Name(),
			Name:        name,
			Description: desc,
			Source:      source,
			Path:        skillMD,
		})
	}
	return out, nil
}

// ParseFrontmatter 解析 SKILL.md 头部 YAML frontmatter 的 name / description。
// 支持 description: > / | 这类 YAML 块标量；逐行切 key:value 时会把 ">" 当成整段描述。
func ParseFrontmatter(content string) (name, description string) {
	content = strings.TrimPrefix(content, "\ufeff")
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		return "", ""
	}
	rest := strings.TrimSpace(content)
	rest = strings.TrimPrefix(rest, "---")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", ""
	}
	lines := strings.Split(rest[:end], "\n")
	for i := 0; i < len(lines); {
		raw := lines[i]
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}
		key, val, ok := strings.Cut(trimmed, ":")
		if !ok {
			i++
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		indent := leadingIndent(raw)
		if isYAMLBlockScalar(val) {
			val, i = readYAMLBlockScalar(lines, i, indent)
		} else {
			val = strings.Trim(val, `"'`)
			i++
		}
		switch key {
		case "name":
			name = val
		case "description":
			description = val
		}
	}
	return name, description
}

func leadingIndent(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += 2
			continue
		}
		break
	}
	return n
}

func isYAMLBlockScalar(val string) bool {
	if val == "" {
		return false
	}
	if val[0] != '>' && val[0] != '|' {
		return false
	}
	rest := val[1:]
	for _, r := range rest {
		if r == '+' || r == '-' || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func readYAMLBlockScalar(lines []string, i, keyIndent int) (string, int) {
	i++
	var parts []string
	for i < len(lines) {
		next := lines[i]
		if strings.TrimSpace(next) == "" {
			parts = append(parts, "")
			i++
			continue
		}
		if leadingIndent(next) <= keyIndent {
			break
		}
		parts = append(parts, strings.TrimSpace(next))
		i++
	}
	text := strings.Join(parts, " ")
	return strings.Join(strings.Fields(text), " "), i
}
