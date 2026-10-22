package skills

import "testing"

func TestExtractAndStrip(t *testing.T) {
	in := "/skill:builtin:git-commit-zh 帮我整理 src\n第二行"
	ids := ExtractSkillIDs(in)
	if len(ids) != 1 || ids[0] != "builtin:git-commit-zh" {
		t.Fatalf("ids=%v", ids)
	}
	got := StripSkillTokens(in)
	want := "帮我整理 src\n第二行"
	if got != want {
		t.Fatalf("strip got %q want %q", got, want)
	}
}
