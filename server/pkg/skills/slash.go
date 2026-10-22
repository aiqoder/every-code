package skills

import (
	"regexp"
	"strings"
)

// 消息中的显式技能指令：/skill:<id>（id 可含冒号）
var skillTokenRE = regexp.MustCompile(`/skill:([^\s]+)`)

// ExtractSkillIDs 按出现顺序去重提取 /skill:<id>。
func ExtractSkillIDs(content string) []string {
	matches := skillTokenRE.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		id := strings.TrimSpace(m[1])
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// StripSkillTokens 去掉技能指令 token，保留用户任务正文（按行清理空白）。
func StripSkillTokens(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		t := skillTokenRE.ReplaceAllString(line, " ")
		t = strings.Join(strings.Fields(t), " ")
		out = append(out, t)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// ReplaceUserContentForLLM 将 user 消息中的 skill token 去掉后交给模型。
func ReplaceUserContentForLLM(content string) string {
	return StripSkillTokens(content)
}
