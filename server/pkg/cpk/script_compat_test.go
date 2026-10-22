package cpk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func cpkpackScript(t *testing.T) string {
	t.Helper()
	p := filepath.Join("..", "..", "examples", "cpk", "software-dev-agent", "skills", "cpkpack", "scripts", "cpkpack.js")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("找不到 cpkpack.js: %v", err)
	}
	return p
}

// TestNodeCpkpackMatchesGoPack 确认技能脚本打出的包与 Go cpkpack 文件集合、校验清单一致。
func TestNodeCpkpackMatchesGoPack(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("未安装 node")
	}
	src := buildSamplePkg(t)
	jsDir := t.TempDir()
	jsOut := filepath.Join(jsDir, "weather-agent-1.0.0.cpk")
	goOut := filepath.Join(t.TempDir(), "go.cpk")

	cmd := exec.Command("node", cpkpackScript(t), "-src", src, "-out", jsDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node cpkpack: %v\n%s", err, out)
	}
	if err := Pack(src, goOut); err != nil {
		t.Fatal(err)
	}

	jsPkg, err := Open(jsOut)
	if err != nil {
		t.Fatalf("Open js: %v", err)
	}
	defer jsPkg.Close()
	goPkg, err := Open(goOut)
	if err != nil {
		t.Fatal(err)
	}
	defer goPkg.Close()

	jsSum, err := jsPkg.ReadFile(ChecksumFile)
	if err != nil {
		t.Fatal(err)
	}
	goSum, err := goPkg.ReadFile(ChecksumFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(jsSum) != string(goSum) {
		t.Fatalf("校验清单不一致\njs:\n%s\ngo:\n%s", jsSum, goSum)
	}
	for _, f := range jsPkg.Files() {
		if f == "server" || strings.HasPrefix(f, "server/") {
			t.Fatalf("server 目录不应入包: %s", f)
		}
	}
}

func TestNodeCpkpackExampleOpens(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("未安装 node")
	}
	src := filepath.Join("..", "..", "examples", "cpk", "software-dev-agent")
	outDir := t.TempDir()
	cmd := exec.Command("node", cpkpackScript(t), "-src", src, "-out", outDir)
	buf, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node cpkpack: %v\n%s", err, buf)
	}
	pkgs, err := filepath.Glob(filepath.Join(outDir, "*.cpk"))
	if err != nil || len(pkgs) != 1 {
		t.Fatalf("cpkpack 产物异常: %v %v", pkgs, err)
	}
	p, err := Open(pkgs[0])
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer p.Close()
	if p.Manifest.ID != "software-dev-agent" || p.Manifest.Name != "软件开发助手" {
		t.Fatalf("manifest: %+v", p.Manifest)
	}
	if len(p.SkillDirs) != 2 || p.SkillDirs[0] != "cpkpack" || p.SkillDirs[1] != "mirror-source" {
		t.Fatalf("SkillDirs: %v", p.SkillDirs)
	}
	if _, err := p.ReadFile("skills/cpkpack/scripts/cpkpack.js"); err != nil {
		t.Fatal(err)
	}
}
