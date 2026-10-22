package tokencount

import (
	"strings"

	"chengzi-agent-code-server/pkg/llmchat"
)

// 系统提示里的技能段标题，须与 conversation.chat_logic 拼法一致。
const (
	BoundSkillsHeading   = "## 已绑定技能"
	InvokedSkillsHeading = "## 本轮调用的技能"
	AvailableToolsPrefix = "Available tools:"
)

// PromptBuckets 一次 prompt 的分桶占用，四项之和等于 Used。
type PromptBuckets struct {
	System   int
	Tools    int
	Skills   int
	Messages int
	Used     int
}

// SplitPrompt 把即将送给模型的 messages + 工具定义拆成系统 / 工具 / 技能 / 对话。
func SplitPrompt(msgs []llmchat.Message, toolsTok int) PromptBuckets {
	used := CountPrompt(msgs, toolsTok)
	sysMsgs, _ := splitLeadingSystem(msgs)
	sysText := joinSystem(sysMsgs)
	sysBody, skillBody := peelSkillSections(sysText)

	system, skills := 0, 0
	if sysText != "" {
		system = CountMessage(llmchat.Message{Role: llmchat.RoleSystem, Content: sysBody})
		if skillBody != "" {
			skills = CountMessage(llmchat.Message{Role: llmchat.RoleSystem, Content: sysText}) - system
			if skills < 0 {
				skills = 0
			}
		}
	}
	messages := used - system - skills - toolsTok
	if messages < 0 {
		messages = 0
	}
	return PromptBuckets{
		System:   system,
		Tools:    toolsTok,
		Skills:   skills,
		Messages: messages,
		Used:     used,
	}
}

func splitLeadingSystem(msgs []llmchat.Message) (sys, rest []llmchat.Message) {
	i := 0
	for i < len(msgs) && msgs[i].Role == llmchat.RoleSystem {
		i++
	}
	return msgs[:i], msgs[i:]
}

func joinSystem(msgs []llmchat.Message) string {
	if len(msgs) == 0 {
		return ""
	}
	if len(msgs) == 1 {
		return msgs[0].Content
	}
	parts := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if t := strings.TrimSpace(m.Content); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n")
}

func peelSkillSections(s string) (rest, skills string) {
	start := firstHeading(s, BoundSkillsHeading, InvokedSkillsHeading)
	if start < 0 {
		return strings.TrimSpace(s), ""
	}
	end := len(s)
	if i := headingIndex(s, AvailableToolsPrefix); i >= start {
		end = i
	}
	skills = strings.TrimSpace(s[start:end])
	head := strings.TrimSpace(s[:start])
	tail := strings.TrimSpace(s[end:])
	if head != "" && tail != "" {
		return head + "\n\n" + tail, skills
	}
	if tail != "" {
		return tail, skills
	}
	return head, skills
}

func firstHeading(s string, headings ...string) int {
	start := -1
	for _, h := range headings {
		if i := headingIndex(s, h); i >= 0 && (start < 0 || i < start) {
			start = i
		}
	}
	return start
}

func headingIndex(s, heading string) int {
	if strings.HasPrefix(s, heading) {
		return 0
	}
	needle := "\n" + heading
	i := strings.Index(s, needle)
	if i < 0 {
		return -1
	}
	return i + 1
}
