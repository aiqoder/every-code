package cpk

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MaxAvatarDim 头像宽、高分别不得超过此值（像素）。
const MaxAvatarDim = 256

// Pack 把 srcDir 目录打包为 .cpk（zip）文件。
// 自动剔除包根 server/ 目录（服务端代码由开发者自行部署，不入包），
// 并生成 manifest.sha256 文件清单（逐文件 sha256，供安装端校验完整性）。
func Pack(srcDir, outPath string) error {
	srcDir = filepath.Clean(srcDir)
	info, err := os.Stat(srcDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("源目录无效: %s", srcDir)
	}

	// 先校验 manifest
	mfData, err := os.ReadFile(filepath.Join(srcDir, ManifestFile))
	if err != nil {
		return fmt.Errorf("源目录缺少 %s: %w", ManifestFile, err)
	}
	m, err := ParseManifest(mfData)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(srcDir, filepath.FromSlash(m.SystemPrompt))); err != nil {
		return fmt.Errorf("源目录缺少系统提示词文件: %s", m.SystemPrompt)
	}
	if m.Avatar != "" {
		if err := validateAvatarFile(filepath.Join(srcDir, filepath.FromSlash(m.Avatar))); err != nil {
			return err
		}
	}

	// 收集待打包文件（剔除包根 server/ 与杂质）
	var files []string
	err = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			// 剔除包根 server/
			if isRootServerDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		base := filepath.Base(path)
		if shouldSkipFile(base) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("源目录无可打包文件")
	}
	sort.Strings(files)

	// 计算逐文件 sha256
	checksums := make([]string, 0, len(files))
	for _, rel := range files {
		sum, err := fileSHA256(filepath.Join(srcDir, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		checksums = append(checksums, sum+"  "+rel)
	}

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	for _, rel := range files {
		if err := addToZip(zw, filepath.Join(srcDir, filepath.FromSlash(rel)), rel); err != nil {
			zw.Close()
			return err
		}
	}
	// 写入 sha256 清单
	w, err := zw.Create(ChecksumFile)
	if err != nil {
		zw.Close()
		return err
	}
	if _, err := w.Write([]byte(strings.Join(checksums, "\n") + "\n")); err != nil {
		zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return nil
}

func validateAvatarFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("源目录缺少头像文件: %s", filepath.Base(path))
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return fmt.Errorf("头像无法解析为图片: %w", err)
	}
	if cfg.Width > MaxAvatarDim || cfg.Height > MaxAvatarDim {
		return fmt.Errorf("头像尺寸不能超过 %d×%d，当前为 %d×%d", MaxAvatarDim, MaxAvatarDim, cfg.Width, cfg.Height)
	}
	return nil
}

// shouldSkipDir 跳过点开头目录、系统杂质与常见依赖安装目录。
func shouldSkipDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "__MACOSX", "node_modules", "venv", "__pycache__":
		return true
	default:
		return false
	}
}

// shouldSkipFile 跳过点开头文件、校验清单与已有 .cpk。
func shouldSkipFile(name string) bool {
	return strings.HasPrefix(name, ".") || name == ChecksumFile || strings.HasSuffix(name, ".cpk")
}

// isRootServerDir 判断相对路径是否为包根 server 目录（或其子目录）。
func isRootServerDir(rel string) bool {
	return rel == "server" || strings.HasPrefix(rel, "server/")
}

func addToZip(zw *zip.Writer, srcPath, rel string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = rel
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
