package strutil

import (
	"strconv"
	"strings"
)

// JoinCSV 将字符串切片以逗号拼接，自动 trim 并跳过空项。
func JoinCSV(items []string) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		s := strings.TrimSpace(item)
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ",")
}

// SplitCSV 将逗号分隔字符串拆成切片，自动 trim 并跳过空项。
func SplitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}
	}
	raw := strings.Split(s, ",")
	parts := make([]string, 0, len(raw))
	for _, item := range raw {
		t := strings.TrimSpace(item)
		if t != "" {
			parts = append(parts, t)
		}
	}
	return parts
}

// JoinUint64CSV 将 uint64 切片以逗号拼接。
func JoinUint64CSV(ids []uint64) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ids))
	seen := map[uint64]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		parts = append(parts, strconv.FormatUint(id, 10))
	}
	return strings.Join(parts, ",")
}

// SplitUint64CSV 将逗号分隔字符串解析为 uint64 切片。
func SplitUint64CSV(s string) []uint64 {
	parts := SplitCSV(s)
	if len(parts) == 0 {
		return []uint64{}
	}
	out := make([]uint64, 0, len(parts))
	seen := map[uint64]struct{}{}
	for _, p := range parts {
		id, err := strconv.ParseUint(p, 10, 64)
		if err != nil || id == 0 {
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
