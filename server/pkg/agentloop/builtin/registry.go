package builtin

import (
	"fmt"
	"strings"

	"chengzi-agent-code-server/pkg/agentloop"
)

// KnownBuiltinNames 可绑定的内置工具名称。
var KnownBuiltinNames = []string{ToolTodoWriter, ToolTask, ToolAskQuestion}

var registry = map[string]func() agentloop.Tool{
	ToolTodoWriter:  NewTodoWriterTool,
	ToolTask:        NewTaskTool,
	ToolAskQuestion: NewAskQuestionTool,
}

// CanonicalName 返回工具名的规范形式（去除首尾空白）。
func CanonicalName(name string) string {
	return strings.TrimSpace(name)
}

// Toolkit 按名称返回已绑定的内置工具实例。
func Toolkit(names []string) []agentloop.Tool {
	out := make([]agentloop.Tool, 0, len(names))
	seen := map[string]struct{}{}
	for _, name := range names {
		name = CanonicalName(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		factory, ok := registry[name]
		if !ok {
			continue
		}
		out = append(out, factory())
	}
	return out
}

// ValidateNames 校验内置工具名称。
func ValidateNames(names []string) error {
	for _, name := range names {
		name = CanonicalName(name)
		if name == "" {
			continue
		}
		if _, ok := registry[name]; !ok {
			return fmt.Errorf("未知内置工具: %s", name)
		}
	}
	return nil
}

// NormalizeNames 去重并过滤未知名称。
func NormalizeNames(names []string) []string {
	out := make([]string, 0, len(names))
	seen := map[string]struct{}{}
	for _, name := range names {
		name = CanonicalName(name)
		if name == "" {
			continue
		}
		if _, ok := registry[name]; !ok {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
