package cpk

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Package 已打开的 .cpk 包（zip 读取态）。
type Package struct {
	Manifest *Manifest
	// SkillDirs 包内技能目录名列表（skills/ 下含 SKILL.md 的目录）。
	SkillDirs []string

	r        *zip.ReadCloser
	fileSet  map[string]*zip.File
}

// Open 打开并校验一个 .cpk 文件。调用方负责 Close。
func Open(path string) (*Package, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("打开 cpk 失败: %w", err)
	}
	p := &Package{r: r, fileSet: map[string]*zip.File{}}
	for _, f := range r.File {
		name := normalizeZipName(f.Name)
		if name == "" {
			continue
		}
		p.fileSet[name] = f
	}
	mf, ok := p.fileSet[ManifestFile]
	if !ok {
		r.Close()
		return nil, fmt.Errorf("包内缺少 %s", ManifestFile)
	}
	data, err := readZipFile(mf)
	if err != nil {
		r.Close()
		return nil, err
	}
	m, err := ParseManifest(data)
	if err != nil {
		r.Close()
		return nil, err
	}
	p.Manifest = m

	// 校验系统提示词文件存在
	if _, ok := p.fileSet[m.SystemPrompt]; !ok {
		r.Close()
		return nil, fmt.Errorf("包内缺少系统提示词文件: %s", m.SystemPrompt)
	}
	if m.Avatar != "" {
		if _, ok := p.fileSet[m.Avatar]; !ok {
			r.Close()
			return nil, fmt.Errorf("包内缺少头像文件: %s", m.Avatar)
		}
	}

	// 收集技能目录：manifest.skills 显式声明优先，否则扫描 skills/ 下含 SKILL.md 的目录
	skillDirs, err := p.collectSkillDirs()
	if err != nil {
		r.Close()
		return nil, err
	}
	p.SkillDirs = skillDirs
	return p, nil
}

// Close 关闭底层 zip reader。
func (p *Package) Close() error {
	return p.r.Close()
}

// ReadFile 读取包内文件内容（相对包根目录路径）。
func (p *Package) ReadFile(name string) ([]byte, error) {
	f, ok := p.fileSet[normalizeZipName(name)]
	if !ok {
		return nil, fmt.Errorf("包内文件不存在: %s", name)
	}
	return readZipFile(f)
}

// ReadSystemPrompt 读取系统提示词全文。
func (p *Package) ReadSystemPrompt() (string, error) {
	b, err := p.ReadFile(p.Manifest.SystemPrompt)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Files 返回包内全部文件路径（相对包根目录，有序性同 zip 内顺序）。
func (p *Package) Files() []string {
	out := make([]string, 0, len(p.r.File))
	for _, f := range p.r.File {
		if name := normalizeZipName(f.Name); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// ExtractTo 把包内容原子地解压到 dir（先解到临时目录再 rename）。
// dir 已存在时返回错误，调用方应先清理。
func (p *Package) ExtractTo(dir string) error {
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("目标目录已存在: %s", dir)
	}
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".cpk-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	for _, f := range p.r.File {
		name := normalizeZipName(f.Name)
		if name == "" {
			continue
		}
		target := filepath.Join(tmp, filepath.FromSlash(name))
		// 防 zip slip：目标必须位于 tmp 内
		if !strings.HasPrefix(target, filepath.Clean(tmp)+string(os.PathSeparator)) {
			return fmt.Errorf("包内存在非法路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := extractZipFile(f, target); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, dir); err != nil {
		return fmt.Errorf("落地包目录失败: %w", err)
	}
	return nil
}

// collectSkillDirs 收集技能目录名并校验 SKILL.md 存在。
func (p *Package) collectSkillDirs() ([]string, error) {
	var dirs []string
	if len(p.Manifest.Skills) > 0 {
		dirs = append(dirs, p.Manifest.Skills...)
	} else {
		seen := map[string]struct{}{}
		for name := range p.fileSet {
			if !strings.HasPrefix(name, "skills/") {
				continue
			}
			rest := strings.TrimPrefix(name, "skills/")
			idx := strings.IndexByte(rest, '/')
			if idx <= 0 {
				continue
			}
			dir := rest[:idx]
			if _, ok := seen[dir]; !ok {
				seen[dir] = struct{}{}
				dirs = append(dirs, dir)
			}
		}
	}
	for _, d := range dirs {
		if _, ok := p.fileSet["skills/"+d+"/SKILL.md"]; !ok {
			return nil, fmt.Errorf("技能 %q 缺少 SKILL.md", d)
		}
	}
	return dirs, nil
}

// normalizeZipName 规范化 zip 内路径：统一为 / 分隔、去前缀 ./、剔除目录与隐藏文件。
func normalizeZipName(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	name = strings.TrimPrefix(name, "./")
	name = strings.TrimSuffix(name, "/")
	if name == "" {
		return ""
	}
	base := name
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	// 跳过 macOS 打包杂质
	if strings.HasPrefix(name, "__MACOSX/") || base == ".DS_Store" {
		return ""
	}
	return name
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("读取包内文件失败 %s: %w", f.Name, err)
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func extractZipFile(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("读取包内文件失败 %s: %w", f.Name, err)
	}
	defer rc.Close()
	mode := f.FileInfo().Mode()
	if mode == 0 {
		mode = 0o644
	}
	w, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer w.Close()
	if _, err := io.Copy(w, rc); err != nil {
		return fmt.Errorf("写入文件失败 %s: %w", target, err)
	}
	return nil
}
