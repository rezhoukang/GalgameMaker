package service

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"galgame-maker/internal/models"
)

// ExportService 导出 HTML 节点、引用的资源和统一播放器。
type ExportService struct {
	db       *gorm.DB
	settings *SettingsService
}

// NewExportService 创建导出服务。
func NewExportService(db *gorm.DB, settings *SettingsService) *ExportService {
	return &ExportService{db: db, settings: settings}
}

// ExportResult 导出结果。
type ExportResult struct {
	OutputDir string   `json:"outputDir"`
	FileCount int      `json:"fileCount"`
	Files     []string `json:"files"`
}

// LinkInfo 一条跳转链接（出端口选项）。
type LinkInfo struct {
	ID            uint                   `json:"id"`            // 原出端口 ID，用于路径历史
	ConditionMode string                 `json:"conditionMode"` // 条件组合方式
	Conditions    []models.PortCondition `json:"conditions"`    // 出现条件
	Effects       []models.PortEffect    `json:"effects"`       // 变量增减
	Label         string                 `json:"label"`
	TargetID      uint                   `json:"target"`
	IsShow        bool                   `json:"isShow"`
}

// SceneRes 节点导出的资源与选项。
type SceneRes struct {
	Hash    string     `json:"hash"`
	Name    string     `json:"name"`
	HTML    string     `json:"html"` // dist 内相对 HTML 路径
	Options []LinkInfo `json:"options"`
}

// Export 把指定画布的故事树导出为单页播放器 index.html + dist 资源目录。
// 每个节点通过 iframe 加载 HTML，页面通知播放器显示选项。
func (s *ExportService) Export(canvasID uint) (*ExportResult, error) {
	if err := s.settings.EnsureStorageConfigured(); err != nil {
		return nil, err
	}
	issues, err := NewNodeService(s.db).CheckCanvas(canvasID)
	if err != nil {
		return nil, err
	}
	if len(issues) > 0 {
		messages := []string{}
		for _, i := range issues {
			messages = append(messages, i.SceneName+"："+i.Problem)
		}
		return nil, fmt.Errorf("导出检测失败：\n%s", strings.Join(messages, "\n"))
	}
	var canvas models.Project
	if err := s.db.First(&canvas, canvasID).Error; err != nil {
		return nil, err
	}
	scenes, ports, err := graph(s.db, canvasID)
	if err != nil {
		return nil, err
	}
	outDir, err := os.MkdirTemp(s.settings.GetOutputDir(), time.Now().Format("20060102-150405")+"-")
	if err != nil {
		return nil, err
	}
	distDir := filepath.Join(outDir, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return nil, err
	}
	result := &ExportResult{OutputDir: outDir}
	res := map[uint]*SceneRes{}
	byHash := map[string]uint{}
	firstID := uint(0)
	for _, sc := range scenes {
		byHash[sc.Hash] = sc.ID
		if sc.IsFirst {
			firstID = sc.ID
		}
	}
	copied := map[string]bool{}
	for _, sc := range scenes {
		r := &SceneRes{Name: sc.Name, Hash: sc.Hash, HTML: sc.ContentPath, Options: []LinkInfo{}}
		if !copied[sc.ContentPath] {
			files, err := contentFiles(canvas, sc)
			if err != nil {
				return nil, err
			}
			for _, rel := range files {
				src, dst := filepath.Join(canvas.Path, rel), filepath.Join(distDir, rel)
				if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
					return nil, err
				}
				if rel == sc.ContentPath {
					data, err := os.ReadFile(src)
					if err != nil {
						return nil, err
					}
					if err := os.WriteFile(dst, withContentBridge(data), 0644); err != nil {
						return nil, err
					}
				} else if strings.EqualFold(filepath.Ext(rel), ".mp4") {
					if err := s.copyVideoForWeb(src, dst); err != nil {
						return nil, err
					}
				} else if err := copyFile(src, dst); err != nil {
					return nil, err
				}
				result.Files = append(result.Files, filepath.ToSlash(filepath.Join("dist", rel)))
			}
			copied[sc.ContentPath] = true
		}
		for _, p := range ports {
			if p.NodeID == sc.ID {
				r.Options = append(r.Options, LinkInfo{ID: p.ID, Label: p.Name, TargetID: byHash[p.ExportHash], IsShow: p.IsShow, ConditionMode: p.ConditionMode, Conditions: p.Conditions, Effects: p.Effects})
			}
		}
		res[sc.ID] = r
	}
	vars, err := projectVariables(s.db, canvasID)
	if err != nil {
		return nil, err
	}
	projectKey := ""
	for _, sc := range scenes {
		if sc.Level == 1 {
			projectKey = sc.Hash
		}
	}
	if err := s.writePlayerIndex(outDir, firstID, res, vars, projectKey); err != nil {
		return nil, err
	}
	result.Files = append(result.Files, "index.html")
	result.FileCount = len(result.Files)
	return result, nil
}

//go:embed templates/player.html
var playerTemplate string

//go:embed templates/story-runtime.js
var storyRuntime string

func (s *ExportService) writePlayerIndex(outDir string, firstID uint, res map[uint]*SceneRes, vars []models.Variable, projectKey string) error {
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}
	definition, err := json.Marshal(struct {
		Nodes     map[uint]*SceneRes
		Variables []models.Variable
		First     uint
	}{res, vars, firstID})
	if err != nil {
		return err
	}
	metadata, err := json.Marshal(struct {
		Variables  []models.Variable `json:"variables"`
		ProjectKey string            `json:"projectKey"`
		Revision   string            `json:"revision"`
	}{vars, projectKey, fmt.Sprintf("%x", sha256.Sum256(definition))})
	if err != nil {
		return err
	}
	page := strings.NewReplacer("__STORY_DATA__", string(data), "__STORY_META__", string(metadata), "__STORY_RUNTIME__", storyRuntime, "__FIRST_NODE__", fmt.Sprint(firstID)).Replace(playerTemplate)
	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte(page), 0644)
}

// copyVideoForWeb 把源视频复制到目标路径供浏览器播放：
// 若视频流是 H.265/HEVC（Chrome 等不支持），用 ffmpeg 转码为 H.264 + faststart；
// ffmpeg 不可用、检测失败或已是 H.264 时直接复制；转码失败会报错（由调用方决定）。
func (s *ExportService) copyVideoForWeb(src, dst string) error {
	codec, err := probeVideoCodec(src)
	if err != nil || (codec != "hevc" && codec != "h265") {
		return copyFile(src, dst)
	}
	ffmpeg := findExec("ffmpeg")
	if ffmpeg == "" {
		return copyFile(src, dst)
	}
	tmp := dst + ".tmp.mp4"
	cmd := exec.Command(ffmpeg, "-y", "-i", src,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-movflags", "+faststart", tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("ffmpeg 转码失败: %v: %s", err, string(out))
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("移动转码结果失败: %w", err)
	}
	return nil
}

// probeVideoCodec 用 ffprobe 读取第一个视频流的编码名（如 h264/hevc），失败返回空。
func probeVideoCodec(path string) (string, error) {
	ffprobe := findExec("ffprobe")
	if ffprobe == "" {
		return "", fmt.Errorf("未找到 ffprobe")
	}
	cmd := exec.Command(ffprobe, "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// findExec 在 PATH 中查找可执行文件，找不到时兜底常见 Windows 安装路径。
func findExec(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	candidates := []string{
		`C:\Program Files\ffmpeg\bin\` + name + ".exe",
		`C:\ffmpeg\bin\` + name + ".exe",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// copyFile 直接复制一个文件。
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
