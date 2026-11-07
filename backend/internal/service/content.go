package service

import (
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"galgame-maker/internal/models"
)

//go:embed templates/video.html
var videoPage string

//go:embed templates/content-bridge.js
var contentBridge string

// Each immutable content version owns its HTML and companion resources.
// Never export an entire project or follow links outside the content directory.
func contentFiles(canvas models.Canvas, node models.Node) ([]string, error) {
	if node.HTMLPath == "" {
		return nil, fmt.Errorf("请导入视频或 HTML 页面后导出")
	}
	rel := filepath.FromSlash(node.HTMLPath)
	if !filepath.IsLocal(rel) || filepath.Base(rel) != "index.html" {
		return nil, fmt.Errorf("节点页面路径无效")
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) != 3 || parts[0] != ".content" || parts[1] == "" || parts[1] == "." || parts[1] == ".." {
		return nil, fmt.Errorf("节点页面必须位于独立内容目录")
	}
	for _, dir := range []string{filepath.Join(canvas.Path, ".content"), filepath.Dir(htmlPath(canvas, node))} {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("节点内容目录不存在或不是普通目录")
		}
	}
	files := []string{}
	err := filepath.WalkDir(filepath.Dir(htmlPath(canvas, node)), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("节点资源不能使用符号链接")
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("节点资源必须为普通文件")
		}
		r, err := filepath.Rel(canvas.Path, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(r))
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !hasFile(htmlPath(canvas, node)) {
		return nil, fmt.Errorf("请导入视频或 HTML 页面后导出")
	}
	data, err := os.ReadFile(htmlPath(canvas, node))
	if err != nil {
		return nil, err
	}
	// Generated video pages always reference their own companion file.
	if string(data) == videoPage && !hasFile(filepath.Join(filepath.Dir(htmlPath(canvas, node)), "video.mp4")) {
		return nil, fmt.Errorf("节点页面引用的视频文件不存在")
	}
	return files, nil
}

func withContentBridge(data []byte) []byte {
	s := string(data)
	bridge := "\n<script>\n" + contentBridge + "\n</script>\n"
	if i := strings.LastIndex(strings.ToLower(s), "</body>"); i >= 0 {
		return []byte(s[:i] + bridge + s[i:])
	}
	return []byte(s + bridge)
}
