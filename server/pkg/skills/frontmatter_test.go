package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrontmatterFoldedDescription(t *testing.T) {
	content := "---\nname: Excel 表格处理\ndescription: >\n  创建、编辑、分析 Excel 表格：公式、图表、透视表\n  与格式设置。\n---\n\n# body\n"
	name, desc := ParseFrontmatter(content)
	if name != "Excel 表格处理" {
		t.Fatalf("name=%q", name)
	}
	want := "创建、编辑、分析 Excel 表格：公式、图表、透视表 与格式设置。"
	if desc != want {
		t.Fatalf("desc=%q want %q", desc, want)
	}
}

func TestParseFrontmatterLiteralDescription(t *testing.T) {
	content := "---\nname: Demo\ndescription: |\n  first line\n  second line\n---\n"
	_, desc := ParseFrontmatter(content)
	if desc != "first line second line" {
		t.Fatalf("desc=%q", desc)
	}
}

func TestParseFrontmatterInlineDescription(t *testing.T) {
	content := "---\nname: Demo\ndescription: 简要说明该技能的用途。\n---\n"
	name, desc := ParseFrontmatter(content)
	if name != "Demo" || desc != "简要说明该技能的用途。" {
		t.Fatalf("name=%q desc=%q", name, desc)
	}
}

func TestListClaudeCodeProjectSkills(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".claude", "skills", "demo-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: Demo Skill\ndescription: from workspace\n---\n\n# hi\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := ListClaudeCode(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range list {
		if s.ID == "project:demo-skill" && s.Name == "Demo Skill" && s.Description == "from workspace" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing project skill, got %#v", list)
	}
}
