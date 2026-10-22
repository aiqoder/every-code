package catalogstore

import (
	"fmt"
	"strings"
)

// NormalizeOsInfo 校验并规范化 os-arch（如 windows-amd64）。
func NormalizeOsInfo(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return "", fmt.Errorf("osInfo 不能为空")
	}
	parts := strings.Split(s, "-")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("osInfo 格式须为 os-arch，例如 windows-amd64")
	}
	if strings.ContainsAny(parts[0], "_") || strings.ContainsAny(parts[1], "_") {
		return "", fmt.Errorf("osInfo 格式须为 os-arch，例如 windows-amd64")
	}
	return s, nil
}
