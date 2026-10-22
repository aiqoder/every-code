package builtin

import "strings"

// DefaultSystemPrompt 根 agent 基础系统提示（不含具体工具说明）。
func DefaultSystemPrompt() string {
	return strings.TrimSpace(`你是橙子 Agent 助手。你可以使用工具完成任务，并用自然语言向用户汇报结果，不要只输出工具调用。
- 若会话已绑定桌面客户端，优先用本机工具（如 read、edit、write、shell_exec、write_stdin）完成文件与命令类任务。
- shell_exec 默认只等待一段时间（yield）；若返回 session_id，用 write_stdin 继续输入或空 chars 轮询，勿对同一交互式 login 另开进程。`)
}

// SystemPromptForBuiltins 根据已绑定内置工具补充提示。
func SystemPromptForBuiltins(names []string) string {
	parts := make([]string, 0, 3)
	seen := map[string]struct{}{}
	for _, name := range names {
		name = CanonicalName(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		switch name {
		case ToolTodoWriter:
			parts = append(parts, "- 复杂多步骤任务先用 todowriter 列出计划并随进度更新。")
		case ToolTask:
			parts = append(parts, "- 范围清晰、可以并行的子任务用 Task 委派。prompt 必须自包含；explore 只读探索，generalPurpose 继承当前工具且不能再调用 Task。")
		case ToolAskQuestion:
			parts = append(parts, "- 需要用户在明确选项里做决定时用 AskQuestion，不要用它询问可以用工具查到的事实。用户有 5 分钟作答；若返回 timeout，请自行决策并继续。")
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}
