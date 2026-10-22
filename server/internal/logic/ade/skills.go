package ade

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SkillStore 系统技能仓库：server/skills/<id>/SKILL.md 及附带文件。
// 只提供预览、下载、上传整包替换三种操作。
type SkillStore struct {
	root string
}

// NewSkillStore 构造技能仓库，root 通常为 server/skills。
func NewSkillStore(root string) *SkillStore {
	return &SkillStore{root: filepath.Clean(root)}
}

// Root 返回技能根目录。
func (s *SkillStore) Root() string { return s.root }

// SkillMeta 目录项。
type SkillMeta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FileCount   int    `json:"fileCount"`
	UpdatedAt   string `json:"updatedAt"`
}

// SkillFile 预览用的文件节点。
type SkillFile struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"isDir"`
}

// List 列出所有系统技能。
func (s *SkillStore) List() ([]SkillMeta, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return []SkillMeta{}, nil
		}
		return nil, err
	}
	out := make([]SkillMeta, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		meta := SkillMeta{ID: id, Name: id}
		info, _ := e.Info()
		if info != nil {
			meta.UpdatedAt = info.ModTime().Format("2006-01-02 15:04:05")
		}
		body, err := os.ReadFile(filepath.Join(s.root, id, "SKILL.md"))
		if err == nil {
			fm := parseFrontmatter(string(body))
			if fm["name"] != "" {
				meta.Name = fm["name"]
			}
			meta.Description = fm["description"]
		}
		meta.FileCount = countFiles(filepath.Join(s.root, id))
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Files 返回某技能的文件树。
func (s *SkillStore) Files(id string) ([]SkillFile, error) {
	dir, err := s.skillDir(id)
	if err != nil {
		return nil, err
	}
	out := make([]SkillFile, 0, 32)
	err = filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() && path == dir {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		out = append(out, SkillFile{
			Path:  filepath.ToSlash(rel),
			Size:  info.Size(),
			IsDir: info.IsDir(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// ReadFile 读取技能内单个文本文件，禁止逃出技能目录。
func (s *SkillStore) ReadFile(id, rel string) (string, error) {
	dir, err := s.skillDir(id)
	if err != nil {
		return "", err
	}
	target, err := safeJoin(dir, rel)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("文件不存在")
	}
	if info.IsDir() {
		return "", fmt.Errorf("这是一个目录")
	}
	if info.Size() > 2<<20 {
		return "", fmt.Errorf("文件过大，请在本地查看")
	}
	body, err := os.ReadFile(target)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Zip 把技能目录打成 zip。
func (s *SkillStore) Zip(id string) ([]byte, string, error) {
	dir, err := s.skillDir(id)
	if err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err = filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
	if err != nil {
		_ = zw.Close()
		return nil, "", err
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), id + ".zip", nil
}

// Replace 上传 zip 整包替换技能目录。zip 内必须含 SKILL.md。
// 允许上层包一层同名目录。
func (s *SkillStore) Replace(id string, body []byte) (*SkillMeta, error) {
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, `/\`) || strings.HasPrefix(id, ".") {
		return nil, fmt.Errorf("技能 id 不合法")
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("不是有效的 zip 文件")
	}

	// 找出 SKILL.md 所在前缀。
	prefix := ""
	found := false
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		if strings.EqualFold(filepath.Base(name), "SKILL.md") {
			prefix = strings.TrimSuffix(name, filepath.Base(name))
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("zip 内没有 SKILL.md")
	}

	tmp := filepath.Join(s.root, ".tmp-"+id)
	if err := os.RemoveAll(tmp); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rel := strings.TrimPrefix(name, prefix)
		if rel == "" || strings.HasSuffix(rel, "/") {
			continue
		}
		target, err := safeJoin(tmp, rel)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return nil, err
		}
		_, copyErr := io.Copy(out, io.LimitReader(rc, 32<<20))
		out.Close()
		rc.Close()
		if copyErr != nil {
			return nil, copyErr
		}
	}

	dir := filepath.Join(s.root, id)
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, dir); err != nil {
		return nil, err
	}

	meta := SkillMeta{ID: id, Name: id, FileCount: countFiles(dir)}
	if raw, err := os.ReadFile(filepath.Join(dir, "SKILL.md")); err == nil {
		fm := parseFrontmatter(string(raw))
		if fm["name"] != "" {
			meta.Name = fm["name"]
		}
		meta.Description = fm["description"]
	}
	return &meta, nil
}

func (s *SkillStore) skillDir(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, `/\`) || strings.HasPrefix(id, ".") {
		return "", fmt.Errorf("技能 id 不合法")
	}
	dir := filepath.Join(s.root, id)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("技能不存在")
	}
	return dir, nil
}

// safeJoin 拼接绝对路径并校验没有逃出 base。
func safeJoin(base, rel string) (string, error) {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	target := filepath.Join(baseAbs, filepath.FromSlash(rel))
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if targetAbs != baseAbs && !strings.HasPrefix(targetAbs, baseAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("路径越界")
	}
	return targetAbs, nil
}

func countFiles(dir string) int {
	n := 0
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			n++
		}
		return nil
	})
	return n
}

// parseFrontmatter 粗解析 SKILL.md 的 YAML 头，只取 name / description。
func parseFrontmatter(body string) map[string]string {
	out := map[string]string{}
	trim := strings.TrimLeft(body, " \t\r\n")
	if !strings.HasPrefix(trim, "---") {
		return out
	}
	rest := trim[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return out
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)
		switch key {
		case "name", "description":
			out[key] = val
		}
	}
	return out
}
