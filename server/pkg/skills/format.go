package skills

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const (
	boundSkillsPathIntro = "脚本与资源位于对应技能目录下。请用绝对路径执行 scripts/ 中的脚本（例如 node <技能目录>/scripts/xxx.js）。需要完整说明时使用 /skill:<id>。"
	noDirGroupTitle      = "无本地目录（请用 /skill:<id> 加载正文）"
)

// Dir 返回技能目录绝对路径。Path 可能是目录本身，或 SKILL.md 文件路径。
func (s Skill) Dir() string {
	p := strings.TrimSpace(s.Path)
	if p == "" {
		return ""
	}
	if strings.EqualFold(filepath.Base(p), "SKILL.md") {
		return filepath.Dir(p)
	}
	return p
}

// FormatBoundSkills 按技能目录的父路径分组，生成系统提示词中的绑定技能段（不含标题）。
func FormatBoundSkills(list []Skill) string {
	if len(list) == 0 {
		return ""
	}
	groups := make(map[string][]Skill)
	for _, s := range list {
		key := ""
		if dir := s.Dir(); dir != "" {
			key = filepath.Dir(dir)
		}
		groups[key] = append(groups[key], s)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] == "" {
			return false
		}
		if keys[j] == "" {
			return true
		}
		return keys[i] < keys[j]
	})

	var b strings.Builder
	b.WriteString(boundSkillsPathIntro)
	for _, key := range keys {
		items := groups[key]
		sort.Slice(items, func(i, j int) bool {
			if items[i].Name != items[j].Name {
				return items[i].Name < items[j].Name
			}
			return items[i].ID < items[j].ID
		})
		title := key
		if title == "" {
			title = noDirGroupTitle
		}
		b.WriteString("\n\n### ")
		b.WriteString(title)
		for _, s := range items {
			b.WriteByte('\n')
			b.WriteString(formatBoundSkillLine(s))
		}
	}
	return b.String()
}

func formatBoundSkillLine(s Skill) string {
	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = s.ID
	}
	head := fmt.Sprintf("- %s（%s）", name, s.ID)
	if desc := strings.TrimSpace(s.Description); desc != "" {
		head += "：" + desc
	}
	if dir := s.Dir(); dir != "" {
		return head + "\n  目录：" + dir
	}
	return head
}
