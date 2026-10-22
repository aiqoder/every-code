package skills

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillDir(t *testing.T) {
	skillDir := filepath.Join("Users", "me", "packages", "weather-agent", "1.0.0", "skills", "weather")
	md := filepath.Join(skillDir, "SKILL.md")
	if got := (Skill{Path: md}).Dir(); got != skillDir {
		t.Fatalf("SKILL.md Path: got %q want %q", got, skillDir)
	}
	if got := (Skill{Path: skillDir}).Dir(); got != skillDir {
		t.Fatalf("dir Path: got %q want %q", got, skillDir)
	}
	if got := (Skill{}).Dir(); got != "" {
		t.Fatalf("empty Path: got %q", got)
	}
}

func TestFormatBoundSkills_GroupByParentDir(t *testing.T) {
	pkgRoot := filepath.Join(string(filepath.Separator), "home", "u", "packages", "weather-agent", "1.0.0", "skills")
	localRoot := filepath.Join(string(filepath.Separator), "home", "u", "workspaces", "abc", "skills")
	weatherDir := filepath.Join(pkgRoot, "weather")
	forecastDir := filepath.Join(pkgRoot, "forecast")
	noteDir := filepath.Join(localRoot, "note")

	got := FormatBoundSkills([]Skill{
		{ID: "pkg:weather-agent:forecast", Name: "forecast", Description: "预报", Path: forecastDir},
		{ID: "pkg:weather-agent:weather", Name: "weather", Description: "查天气", Path: weatherDir},
		{ID: "local:note", Name: "note", Description: "笔记", Path: noteDir},
		{ID: "builtin:git", Name: "git", Description: "提交"},
	})
	if got == "" {
		t.Fatal("empty")
	}
	if !strings.Contains(got, boundSkillsPathIntro) {
		t.Fatalf("missing intro:\n%s", got)
	}
	if !strings.Contains(got, "### "+pkgRoot) {
		t.Fatalf("missing pkg group:\n%s", got)
	}
	if !strings.Contains(got, "### "+localRoot) {
		t.Fatalf("missing local group:\n%s", got)
	}
	if !strings.Contains(got, "### "+noDirGroupTitle) {
		t.Fatalf("missing no-dir group:\n%s", got)
	}
	if !strings.Contains(got, "- weather（pkg:weather-agent:weather）：查天气") {
		t.Fatalf("missing weather line:\n%s", got)
	}
	if !strings.Contains(got, "目录："+weatherDir) {
		t.Fatalf("missing weather dir:\n%s", got)
	}
	if !strings.Contains(got, "- git（builtin:git）：提交") {
		t.Fatalf("missing builtin line:\n%s", got)
	}

	pkgIdx := strings.Index(got, "### "+pkgRoot)
	localIdx := strings.Index(got, "### "+localRoot)
	noneIdx := strings.Index(got, "### "+noDirGroupTitle)
	if pkgIdx < 0 || localIdx < 0 || noneIdx < 0 || noneIdx < pkgIdx || noneIdx < localIdx {
		t.Fatalf("group order wrong: pkg=%d local=%d none=%d\n%s", pkgIdx, localIdx, noneIdx, got)
	}
}

func TestFormatBoundSkills_Empty(t *testing.T) {
	if got := FormatBoundSkills(nil); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatBoundSkills_SKILLMdPath(t *testing.T) {
	skillDir := filepath.Join(string(filepath.Separator), "tmp", "skills", "demo")
	got := FormatBoundSkills([]Skill{
		{ID: "claude:demo", Name: "demo", Path: filepath.Join(skillDir, "SKILL.md")},
	})
	if !strings.Contains(got, "目录："+skillDir) {
		t.Fatalf("should normalize SKILL.md to dir:\n%s", got)
	}
	parent := filepath.Dir(skillDir)
	if !strings.Contains(got, "### "+parent) {
		t.Fatalf("missing parent group %q:\n%s", parent, got)
	}
}
