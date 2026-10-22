package cpk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validManifest = `
id: weather-agent
name: 天气助手
description: 查询天气的智能体
version: 1.0.0
author: tester
systemPrompt: prompt.md
maxSteps: 20
server:
  baseUrl: https://api.example.com/
env:
  - name: WEATHER_API_KEY
    label: 天气接口密钥
    secret: true
    required: true
  - name: DEFAULT_CITY
    label: 默认城市
`

func TestParseManifest_Valid(t *testing.T) {
	m, err := ParseManifest([]byte(validManifest))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if m.ID != "weather-agent" || m.Name != "天气助手" || m.Version != "1.0.0" {
		t.Fatalf("字段解析错误: %+v", m)
	}
	// baseUrl 尾部斜杠应被裁剪
	if m.Server.BaseURL != "https://api.example.com" {
		t.Fatalf("baseUrl 未规范化: %q", m.Server.BaseURL)
	}
	if len(m.Env) != 2 || m.Env[0].Name != "WEATHER_API_KEY" || !m.Env[0].Secret || !m.Env[0].Required {
		t.Fatalf("env 解析错误: %+v", m.Env)
	}
	if m.Env[1].Name != "DEFAULT_CITY" || m.Env[1].Secret || m.Env[1].Required {
		t.Fatalf("可选环境变量解析错误: %+v", m.Env[1])
	}
	if m.Env[0].NormalizedType() != EnvTypeString || len(m.Env[0].Items) != 0 {
		t.Fatalf("缺省 type 应为 string: %+v", m.Env[0])
	}
}

func TestParseManifest_EnvList(t *testing.T) {
	yaml := `
id: baota-panel
name: 宝塔
version: 1.0.0
systemPrompt: prompt.md
env:
  - name: BT_PANELS
    label: 面板清单
    required: true
    type: list
    items:
      - name: alias
        label: 别名
      - name: host
        label: 主机
        required: true
      - name: password
        label: 密码
        secret: true
`
	m, err := ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	ev := m.Env[0]
	if ev.Type != EnvTypeList || ev.Secret {
		t.Fatalf("list 顶层 secret 应被清除: %+v", ev)
	}
	if len(ev.Items) != 3 || !ev.Items[1].Required || !ev.Items[2].Secret {
		t.Fatalf("items 解析错误: %+v", ev.Items)
	}
}

func TestEnvList_SerializeParse(t *testing.T) {
	items := []EnvItem{
		{Name: "alias"},
		{Name: "host", Required: true},
		{Name: "port"},
		{Name: "user"},
		{Name: "password", Secret: true},
	}
	raw := SerializeEnvList(items, [][]string{
		{"hk", "1.2.3.4", "22", "root", "p|ass"},
		{"", "", "", "", ""},
		{"jp", "5.6.7.8", "", "", ""},
	})
	want := "hk | 1.2.3.4 | 22 | root | p|ass\njp | 5.6.7.8"
	if raw != want {
		t.Fatalf("SerializeEnvList:\n got %q\nwant %q", raw, want)
	}
	rows := ParseEnvList(items, raw)
	if len(rows) != 2 || rows[0][4] != "p|ass" || rows[1][0] != "jp" {
		t.Fatalf("ParseEnvList: %+v", rows)
	}
	if err := ValidateEnvListValue(EnvVar{Name: "BT_PANELS", Required: true, Type: EnvTypeList, Items: items}, raw); err != nil {
		t.Fatalf("ValidateEnvListValue: %v", err)
	}
	if err := ValidateEnvListValue(EnvVar{Name: "BT_PANELS", Required: true, Type: EnvTypeList, Items: items}, ""); err == nil {
		t.Fatal("必填 list 空值应报错")
	}
}

func TestParseManifest_Invalid(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{"缺少name", "id: a\nversion: 1.0.0\nsystemPrompt: p.md\n"},
		{"id大写", "id: Abc\nname: x\nversion: 1.0.0\nsystemPrompt: p.md\n"},
		{"id非法字符", "id: a/b\nname: x\nversion: 1.0.0\nsystemPrompt: p.md\n"},
		{"version非法", "id: a\nname: x\nversion: 1.x\nsystemPrompt: p.md\n"},
		{"缺少systemPrompt", "id: a\nname: x\nversion: 1.0.0\n"},
		{"systemPrompt穿越", "id: a\nname: x\nversion: 1.0.0\nsystemPrompt: ../p.md\n"},
		{"maxSteps越界", "id: a\nname: x\nversion: 1.0.0\nsystemPrompt: p.md\nmaxSteps: 1000\n"},
		{"env名非法", "id: a\nname: x\nversion: 1.0.0\nsystemPrompt: p.md\nenv:\n  - name: 1TOKEN\n"},
		{"env保留前缀", "id: a\nname: x\nversion: 1.0.0\nsystemPrompt: p.md\nenv:\n  - name: CHENGZI_TOKEN\n"},
		{"env名重复", "id: a\nname: x\nversion: 1.0.0\nsystemPrompt: p.md\nenv:\n  - name: TOKEN\n  - name: token\n"},
		{"env type非法", "id: a\nname: x\nversion: 1.0.0\nsystemPrompt: p.md\nenv:\n  - name: TOKEN\n    type: map\n"},
		{"list缺items", "id: a\nname: x\nversion: 1.0.0\nsystemPrompt: p.md\nenv:\n  - name: TOKEN\n    type: list\n"},
		{"string带items", "id: a\nname: x\nversion: 1.0.0\nsystemPrompt: p.md\nenv:\n  - name: TOKEN\n    items:\n      - name: a\n"},
		{"baseUrl协议非法", "id: a\nname: x\nversion: 1.0.0\nsystemPrompt: p.md\nserver:\n  baseUrl: ftp://a.com\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ParseManifest([]byte(c.yaml)); err == nil {
				t.Fatal("应返回校验错误")
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.0.0-beta", "1.0.0", -1},
		{"1.0.0", "1.0.0-beta", 1},
		{"1.2", "1.2.0", 0},
	}
	for _, c := range cases {
		got, err := Compare(c.a, c.b)
		if err != nil {
			t.Fatalf("Compare(%s,%s): %v", c.a, c.b, err)
		}
		if got != c.want {
			t.Fatalf("Compare(%s,%s)=%d, want %d", c.a, c.b, got, c.want)
		}
	}
	if _, err := ParseVersion("abc"); err == nil {
		t.Fatal("应拒绝非法版本号")
	}
}

// buildSamplePkg 构造一个示例包源目录。
func buildSamplePkg(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("manifest.yaml", validManifest)
	write("prompt.md", "你是一个天气助手。")
	write("skills/weather/SKILL.md", "---\nname: weather\ndescription: 查天气\n---\n调用 scripts/query.js")
	write("skills/weather/scripts/query.js", "console.log('query')")
	write("server/server.js", "// 不应入包")
	return dir
}

func TestPackAndOpen_RoundTrip(t *testing.T) {
	src := buildSamplePkg(t)
	out := filepath.Join(t.TempDir(), "weather-agent.cpk")
	if err := Pack(src, out); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	p, err := Open(out)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer p.Close()

	if p.Manifest.ID != "weather-agent" {
		t.Fatalf("manifest id: %q", p.Manifest.ID)
	}
	if len(p.SkillDirs) != 1 || p.SkillDirs[0] != "weather" {
		t.Fatalf("SkillDirs: %v", p.SkillDirs)
	}

	// 包根 server/ 应被剔除
	for _, f := range p.Files() {
		if f == "server" || strings.HasPrefix(f, "server/") {
			t.Fatalf("server 目录不应入包: %s", f)
		}
	}
	// 技能脚本应入包
	if _, err := p.ReadFile("skills/weather/scripts/query.js"); err != nil {
		t.Fatalf("技能脚本应入包: %v", err)
	}
	// sha256 清单应存在
	if _, err := p.ReadFile(ChecksumFile); err != nil {
		t.Fatalf("缺少 %s: %v", ChecksumFile, err)
	}
	// 系统提示词可读
	prompt, err := p.ReadSystemPrompt()
	if err != nil || !strings.Contains(prompt, "天气助手") {
		t.Fatalf("ReadSystemPrompt: %v %q", err, prompt)
	}
}

func TestExtractTo(t *testing.T) {
	src := buildSamplePkg(t)
	out := filepath.Join(t.TempDir(), "a.cpk")
	if err := Pack(src, out); err != nil {
		t.Fatal(err)
	}
	p, err := Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	dest := filepath.Join(t.TempDir(), "packages", "weather-agent", "1.0.0")
	if err := p.ExtractTo(dest); err != nil {
		t.Fatalf("ExtractTo: %v", err)
	}
	// 已存在时再次解压应报错
	if err := p.ExtractTo(dest); err == nil {
		t.Fatal("目标已存在应报错")
	}
	// 文件应完整落地
	b, err := os.ReadFile(filepath.Join(dest, "skills", "weather", "SKILL.md"))
	if err != nil || !strings.Contains(string(b), "weather") {
		t.Fatalf("落地文件校验失败: %v", err)
	}
}

func TestOpen_Errors(t *testing.T) {
	// 非 zip 文件
	bad := filepath.Join(t.TempDir(), "bad.cpk")
	if err := os.WriteFile(bad, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(bad); err == nil {
		t.Fatal("非 zip 应报错")
	}
	// 缺少 prompt 文件
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(validManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "noprompt.cpk")
	if err := Pack(dir, out); err == nil {
		t.Fatal("缺少 prompt 文件应在打包期报错")
	}
}
