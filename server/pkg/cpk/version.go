package cpk

import (
	"fmt"
	"strconv"
	"strings"
)

// Version 语义化版本（major.minor.patch，支持可选 v 前缀与预发布后缀）。
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

// ParseVersion 解析 semver 字符串，如 "1.2.3"、"v1.2.3"、"1.2.3-beta.1"。
func ParseVersion(s string) (Version, error) {
	var v Version
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return v, fmt.Errorf("版本号为空")
	}
	if i := strings.IndexByte(s, '-'); i >= 0 {
		v.Prerelease = s[i+1:]
		s = s[:i]
		if v.Prerelease == "" {
			return v, fmt.Errorf("预发布标识为空: %q", s)
		}
	}
	// 忽略构建元数据 +build
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) > 3 {
		return v, fmt.Errorf("版本号段数过多: %q", s)
	}
	nums := []int{0, 0, 0}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, fmt.Errorf("版本号段无效: %q", p)
		}
		nums[i] = n
	}
	v.Major, v.Minor, v.Patch = nums[0], nums[1], nums[2]
	return v, nil
}

// String 返回规范化版本串（不含 v 前缀）。
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	return s
}

// Compare 比较两个版本：a<b 返回 -1，a==b 返回 0，a>b 返回 1。
// 预发布版本低于同号正式版（1.0.0-beta < 1.0.0）。
func Compare(a, b string) (int, error) {
	va, err := ParseVersion(a)
	if err != nil {
		return 0, err
	}
	vb, err := ParseVersion(b)
	if err != nil {
		return 0, err
	}
	if va.Major != vb.Major {
		return cmpInt(va.Major, vb.Major), nil
	}
	if va.Minor != vb.Minor {
		return cmpInt(va.Minor, vb.Minor), nil
	}
	if va.Patch != vb.Patch {
		return cmpInt(va.Patch, vb.Patch), nil
	}
	switch {
	case va.Prerelease == vb.Prerelease:
		return 0, nil
	case va.Prerelease == "":
		return 1, nil
	case vb.Prerelease == "":
		return -1, nil
	default:
		return strings.Compare(va.Prerelease, vb.Prerelease), nil
	}
}

// LessThan a 版本是否低于 b。
func LessThan(a, b string) (bool, error) {
	c, err := Compare(a, b)
	return c < 0, err
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
