package service

import (
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

// ExportService 负责导出：每个 HTML 框（scene）生成一个独立 HTML 文件。
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
	Label      string `json:"label"`
	TargetID   uint   `json:"target"`
	TargetKind string `json:"kind"` // mp4 / html
}

// SceneRes 节点导出的资源与选项。
type SceneRes struct {
	Name    string     `json:"name"`
	HTML    string     `json:"html"` // dist 内相对文件名，空=无 HTML
	MP4s    []string   `json:"mp4s"` // dist 内相对文件名列表（多个视频）
	Options []LinkInfo `json:"options"`
	Ending  string     `json:"ending"`
}

// Export 把指定画布的故事树导出为单页播放器 index.html + dist 资源目录。
// 全部故事线由 index.html 一个 HTML 驱动：视频用 <video> 切换播放，
// 页面节点用 iframe 加载，选项在播放器内跳转（不再为每个节点生成独立 HTML）。
func (s *ExportService) Export(canvasID uint) (*ExportResult, error) {
	if err := s.settings.EnsureStorageConfigured(); err != nil {
		return nil, err
	}
	rootDir := s.settings.GetOutputDir()
	ts := time.Now().Format("20060102-150405")
	outDir := filepath.Join(rootDir, ts)
	distDir := filepath.Join(outDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建导出目录失败: %w", err)
	}

	var scenes []models.HtmlScene
	if err := s.db.Where("canvas_id = ?", canvasID).Order("id ASC").Find(&scenes).Error; err != nil {
		return nil, err
	}
	var canvases []models.Canvas
	if err := s.db.Find(&canvases).Error; err != nil {
		return nil, err
	}
	canvasByID := map[uint]*models.Canvas{}
	for i := range canvases {
		canvasByID[canvases[i].ID] = &canvases[i]
	}
	var nodes []models.Node
	if len(scenes) > 0 {
		sceneIDs := make([]uint, 0, len(scenes))
		for _, sc := range scenes {
			sceneIDs = append(sceneIDs, sc.ID)
		}
		if err := s.db.Where("scene_id IN ?", sceneIDs).Find(&nodes).Error; err != nil {
			return nil, err
		}
	}
	sceneByID := map[uint]*models.HtmlScene{}
	for i := range scenes {
		sceneByID[scenes[i].ID] = &scenes[i]
	}

	// 出边：scene → 选项列表（出端口 !Entry 且有配对，目标 = 配对入端口所在场景）
	outgoing := map[uint][]LinkInfo{}
	for _, n := range nodes {
		if n.Entry {
			continue
		}
		var peer models.Node
		if err := s.db.Where("hash = ? AND id <> ?", n.Hash, n.ID).First(&peer).Error; err != nil {
			continue // 无配对 = 空端口（导出前检测会拦）
		}
		toScene := peer.SceneID
		if _, ok := sceneByID[toScene]; !ok {
			continue
		}
		kind := n.TargetKind
		if kind != "html" {
			kind = "mp4"
		}
		outgoing[n.SceneID] = append(outgoing[n.SceneID], LinkInfo{
			Label:      n.Name,
			TargetID:   toScene,
			TargetKind: kind,
		})
	}

	// first：优先区块级，其次端口级，最后第一个 scene
	firstID := uint(0)
	for _, sc := range scenes {
		if sc.IsFirst {
			firstID = sc.ID
			break
		}
	}
	if firstID == 0 {
		for _, n := range nodes {
			if n.IsFirst {
				firstID = n.SceneID
				break
			}
		}
	}
	if firstID == 0 && len(scenes) > 0 {
		firstID = scenes[0].ID
	}

	// 末节点（无出边）结局内容：取该节点内第一个填了结局内容的端口
	sceneEnding := map[uint]string{}
	for _, n := range nodes {
		if n.EndingContent == "" || len(outgoing[n.SceneID]) > 0 {
			continue
		}
		if _, ok := sceneEnding[n.SceneID]; !ok {
			sceneEnding[n.SceneID] = n.EndingContent
		}
	}

	// 逐节点复制资源
	result := &ExportResult{OutputDir: outDir}
	res := map[uint]*SceneRes{}
	for _, sc := range scenes {
		canvas, ok := canvasByID[sc.CanvasID]
		if !ok {
			continue
		}
		prefix := fmt.Sprintf("%d-%s", sc.ID, sanitizeFileName(sceneStem(sc.Name)))
		r := &SceneRes{Name: sc.Name}
		// HTML（可选）：<文件夹>/<stem>.html
		if src, err := os.ReadFile(sceneHTMLPath(canvas.Path, sc.Name)); err == nil && len(src) > 0 {
			htmlName := prefix + ".html"
			if err := os.WriteFile(filepath.Join(distDir, htmlName), src, 0o644); err != nil {
				return nil, fmt.Errorf("写入 HTML 失败 %s: %w", htmlName, err)
			}
			r.HTML = htmlName
			result.Files = append(result.Files, htmlName)
		}
		// MP4（可多个）：scene.Video 优先，其余按文件顺序
		dir := sceneDir(canvas.Path, sc.Name)
		mp4s := []string{}
		if sc.Video != "" {
			if _, err := os.Stat(filepath.Join(dir, sc.Video)); err == nil {
				mp4s = append(mp4s, sc.Video)
			}
		}
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if e.IsDir() || filepath.Ext(e.Name()) != ".mp4" || e.Name() == sc.Video {
					continue
				}
				mp4s = append(mp4s, e.Name())
			}
		}
		for i, mp4 := range mp4s {
			dstName := prefix + ".mp4"
			if i > 0 {
				dstName = fmt.Sprintf("%s-%d.mp4", prefix, i+1)
			}
			if err := s.copyVideoForWeb(filepath.Join(dir, mp4), filepath.Join(distDir, dstName)); err != nil {
				return nil, fmt.Errorf("写入视频失败 %s: %w", dstName, err)
			}
			r.MP4s = append(r.MP4s, dstName)
			result.Files = append(result.Files, dstName)
		}
		r.Options = outgoing[sc.ID]
		r.Ending = sceneEnding[sc.ID]
		res[sc.ID] = r
	}

	// 导出前检测：空端口 / 跳转资源缺失
	var issues []string
	for _, n := range nodes {
		scName := sceneByID[n.SceneID].Name
		var peer models.Node
		hasPeer := s.db.Where("hash = ? AND id <> ?", n.Hash, n.ID).First(&peer).Error == nil
		if !hasPeer {
			issues = append(issues, fmt.Sprintf("节点「%s」的端口「%s」是空端口（没有配对的端口）", scName, n.Name))
			continue
		}
		if n.Entry {
			continue // 入端口无需资源检测
		}
		kind := n.TargetKind
		if kind != "html" {
			kind = "mp4"
		}
		if tr, ok2 := res[peer.SceneID]; ok2 {
			missing := (kind == "mp4" && len(tr.MP4s) == 0) || (kind == "html" && tr.HTML == "")
			if missing {
				kindLabel := "MP4 视频"
				if kind == "html" {
					kindLabel = "HTML 页面"
				}
				issues = append(issues, fmt.Sprintf(
					"节点「%s」的端口「%s」跳转到「%s」，但目标文件夹里没有 %s", scName, n.Name, tr.Name, kindLabel))
			}
		}
	}
	if len(issues) > 0 {
		return nil, fmt.Errorf("导出检测失败：\n- %s", strings.Join(issues, "\n- "))
	}

	if err := s.writePlayerIndex(outDir, firstID, res); err != nil {
		return nil, err
	}
	result.Files = append(result.Files, "index.html")
	result.FileCount = len(result.Files)
	return result, nil
}

// writePlayerIndex 生成唯一入口 index.html（单页播放器）：
// 内嵌全部故事数据，视频用 <video> 切换、页面用 iframe 加载、选项在页内跳转，
// 一个 HTML 驱动全部故事线。kind=mp4 播目标节点第一个视频；kind=html 加载目标节点页面。
func (s *ExportService) writePlayerIndex(outDir string, firstID uint, res map[uint]*SceneRes) error {
	// 数据 JSON（map 键为字符串场景 id）
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Galgame 互动视频</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#000;font-family:'Microsoft YaHei',sans-serif;overflow:hidden}
#stage{position:fixed;inset:0;background:#000}
#stage video{width:100vw;height:100vh;object-fit:contain;outline:none}
#page{position:fixed;inset:0;border:none;width:100%%;height:100%%}
#bar{position:fixed;top:16px;left:16px;z-index:9;display:flex;align-items:center;gap:10px;background:rgba(0,0,0,.55);padding:8px 14px;border-radius:12px}
#bar button{background:transparent;border:none;color:#fff;font-size:20px;cursor:pointer;line-height:1}
#bar input[type=range]{width:110px;height:4px;accent-color:#fff;cursor:pointer}
#restart{position:fixed;top:16px;right:16px;z-index:9;display:none;background:rgba(0,0,0,.55);color:#fff;border:none;padding:12px 30px;font-size:16px;font-weight:700;border-radius:12px;cursor:pointer}
#restart:hover{background:rgba(0,0,0,.78)}
#opts{position:fixed;left:0;right:0;bottom:42px;display:none;gap:14px;justify-content:center;align-items:center;flex-wrap:wrap;z-index:10}
.opt{display:inline-block;padding:14px 34px;background:rgba(0,0,0,.78);color:#fff;border:1px solid rgba(255,255,255,.22);font-size:16px;font-weight:600;border-radius:10px;box-shadow:0 4px 16px rgba(0,0,0,.4);cursor:pointer;transition:.15s}
.opt:hover{transform:translateY(-2px);background:rgba(46,46,52,.9)}
#next{position:fixed;right:20px;bottom:42px;display:none;background:rgba(0,0,0,.7);color:#fff;border:1px solid rgba(255,255,255,.25);padding:12px 26px;font-size:15px;font-weight:700;border-radius:10px;cursor:pointer;z-index:10}
#next:hover{background:rgba(46,46,52,.95)}
#ending{position:fixed;inset:0;background:#000;z-index:8;display:none;align-items:center;justify-content:center;padding:48px}
#ending-text{color:#fff;font-size:26px;line-height:1.9;text-align:center;max-width:680px;white-space:pre-wrap}
#start-wrap{position:fixed;inset:0;z-index:20;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:18px;background:radial-gradient(1200px 600px at 50%% -10%%,#1c1c28,#0e0e14 60%%)}
#start-wrap h1{color:#fff;font-size:40px;letter-spacing:2px}
#start-wrap .sub{color:#777;font-size:14px}
#start{display:inline-block;padding:16px 64px;background:#d97706;color:#fff;font-size:20px;font-weight:700;border:none;border-radius:14px;box-shadow:0 8px 30px rgba(217,119,6,.4);cursor:pointer;transition:.15s}
#start:hover{transform:translateY(-2px);box-shadow:0 12px 36px rgba(217,119,6,.5)}
</style>
</head>
<body>
<div id="start-wrap"><h1>Galgame 互动视频</h1><p class="sub">点击开始</p><button id="start">▶ 开始</button></div>
<div id="stage"><video id="video" playsinline></video></div>
<iframe id="page" style="display:none"></iframe>
<div id="bar">
<button id="mute" type="button" title="静音/取消静音">🔊</button>
<input id="vol" type="range" min="0" max="100" value="100" title="音量">
</div>
<button id="restart" type="button">↺ 重来</button>
<div id="opts"></div>
<button id="next" type="button">继续 ▸</button>
<div id="ending"><div id="ending-text"></div></div>
<script type="application/json" id="story-data">%s</script>
<script>
var DATA = JSON.parse(document.getElementById('story-data').textContent);
var FIRST = %d;
var cur = null;

var stage = document.getElementById('stage');
var video = document.getElementById('video');
var page = document.getElementById('page');
var optsBox = document.getElementById('opts');
var nextBtn = document.getElementById('next');
var ending = document.getElementById('ending');
var endingText = document.getElementById('ending-text');
var restartBtn = document.getElementById('restart');
var muteBtn = document.getElementById('mute');
var volBar = document.getElementById('vol');

video.volume = 1;
muteBtn.addEventListener('click', function(){ video.muted = !video.muted; muteBtn.textContent = video.muted ? '🔇' : '🔊'; });
volBar.addEventListener('input', function(){ video.volume = volBar.value / 100; video.muted = video.volume === 0; muteBtn.textContent = video.muted ? '🔇' : '🔊'; });
document.addEventListener('keydown', function(e){
  if(e.key === 'ArrowUp'){ video.volume = Math.min(1, video.volume + 0.1); video.muted = false; syncBar(); }
  else if(e.key === 'ArrowDown'){ video.volume = Math.max(0, video.volume - 0.1); video.muted = video.volume === 0; syncBar(); }
});
function syncBar(){ muteBtn.textContent = video.muted ? '🔇' : '🔊'; volBar.value = Math.round(video.volume * 100); }

function hideAll(){
  stage.style.display = 'none';
  page.style.display = 'none';
  optsBox.style.display = 'none';
  nextBtn.style.display = 'none';
  ending.style.display = 'none';
  restartBtn.style.display = 'none';
  video.pause();
}

// 展示当前节点：kind=mp4 播视频（无视频自动降级页面）；kind=html 加载页面
function show(id, kind){
  cur = id;
  var sc = DATA[String(id)];
  if(!sc){ return; }
  hideAll();
  restartBtn.style.display = 'block';
  if(kind !== 'html' && sc.mp4s && sc.mp4s.length > 0){ playVideo(sc.mp4s[0]); return; }
  if(sc.html){ showPage(sc.html); return; }
  if(sc.mp4s && sc.mp4s.length > 0){ playVideo(sc.mp4s[0]); return; }
  afterContent(); // 无任何资源：直接出选项/结局
}

function playVideo(src){
  stage.style.display = 'block';
  video.src = 'dist/' + src;
  video.play().catch(function(){});
  video.onended = afterContent;
  video.onerror = afterContent;
}

function showPage(src){
  page.style.display = 'block';
  page.src = 'dist/' + src;
  // 页面节点：加载后由「继续」按钮进入选项/结局
  nextBtn.style.display = 'block';
}

function afterContent(){
  var sc = DATA[String(cur)];
  if(!sc){ return; }
  if(sc.options && sc.options.length > 0){
    optsBox.innerHTML = '';
    sc.options.forEach(function(o){
      var btn = document.createElement('button');
      btn.className = 'opt';
      btn.textContent = o.label;
      btn.onclick = function(){ show(o.target, o.kind); };
      optsBox.appendChild(btn);
    });
    optsBox.style.display = 'flex';
  } else {
    endingText.textContent = sc.ending || '';
    ending.style.display = 'flex';
  }
}

nextBtn.addEventListener('click', afterContent);
restartBtn.addEventListener('click', function(){ show(FIRST, 'auto'); });
document.getElementById('start').addEventListener('click', function(){
  document.getElementById('start-wrap').style.display = 'none';
  show(FIRST, 'auto');
});
</script>
</body>
</html>
`)
	page := fmt.Sprintf(b.String(), data, firstID)
	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte(page), 0o644)
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
