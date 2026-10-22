package cpk

import (
	"fmt"
	"strings"
)

// SerializeEnvList 将结构化行序列化为管道符多行字符串。
// 每行按 items 顺序用 " | " 拼接，去掉末尾空字段；全空行丢弃。
func SerializeEnvList(items []EnvItem, rows [][]string) string {
	if len(items) == 0 || len(rows) == 0 {
		return ""
	}
	var lines []string
	for _, row := range rows {
		vals := make([]string, len(items))
		any := false
		for i := range items {
			if i < len(row) {
				vals[i] = strings.TrimSpace(row[i])
			}
			if vals[i] != "" {
				any = true
			}
		}
		if !any {
			continue
		}
		end := len(vals)
		for end > 0 && vals[end-1] == "" {
			end--
		}
		lines = append(lines, strings.Join(vals[:end], " | "))
	}
	return strings.Join(lines, "\n")
}

// ParseEnvList 将管道符多行字符串解析为结构化行（列数对齐 items）。
// 空行与仅空白的行丢弃；列不足时右侧补空，列过多时合并进最后一列（兼容密码含 |）。
func ParseEnvList(items []EnvItem, raw string) [][]string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	if strings.TrimSpace(raw) == "" || len(items) == 0 {
		return nil
	}
	var rows [][]string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := splitPipeFields(line)
		row := make([]string, len(items))
		for i := range items {
			if i < len(parts) {
				row[i] = strings.TrimSpace(parts[i])
			}
		}
		if len(parts) > len(items) {
			// 多余段并入最后一列（密码里可能含 |）
			extra := strings.Join(parts[len(items)-1:], "|")
			row[len(items)-1] = strings.TrimSpace(extra)
		}
		rows = append(rows, row)
	}
	return rows
}

func splitPipeFields(line string) []string {
	if !strings.Contains(line, "|") {
		return []string{line}
	}
	raw := strings.Split(line, "|")
	out := make([]string, len(raw))
	for i, p := range raw {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

// ValidateEnvListValue 校验 type=list 的已序列化取值是否满足 schema。
// required 为顶层必填时，至少需要一行且每行满足列级 required。
func ValidateEnvListValue(ev EnvVar, value string) error {
	if ev.NormalizedType() != EnvTypeList {
		return nil
	}
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("环境变量 %s 的值不能包含空字符", ev.Name)
	}
	rows := ParseEnvList(ev.Items, value)
	if len(rows) == 0 {
		if ev.Required {
			label := ev.Label
			if label == "" {
				label = ev.Name
			}
			return fmt.Errorf("请填写%s", label)
		}
		return nil
	}
	for ri, row := range rows {
		for i, item := range ev.Items {
			cell := ""
			if i < len(row) {
				cell = strings.TrimSpace(row[i])
			}
			if item.Required && cell == "" {
				label := item.Label
				if label == "" {
					label = item.Name
				}
				return fmt.Errorf("第 %d 行请填写%s", ri+1, label)
			}
		}
	}
	return nil
}
