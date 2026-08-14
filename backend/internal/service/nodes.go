package service

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"galgame-maker/internal/models"
)

// NodeService 管理 HTML 框（HtmlScene）、节点（点）与连线。
type NodeService struct {
	db *gorm.DB
}

// NewNodeService 创建节点服务。
func NewNodeService(db *gorm.DB) *NodeService {
	return &NodeService{db: db}
}

// ---------------------------------------------------------------- HTML 框

// sceneStem 返回 HTML 文件名去掉扩展名（同时作为同名子文件夹名）
func sceneStem(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// sceneDir 返回该 scene 对应的同名子文件夹路径（画布目录/<stem>）
func sceneDir(canvasPath, name string) string {
	return filepath.Join(canvasPath, sceneStem(name))
}

// sceneHTMLPath 返回 scene 的 HTML 文件路径：<dir>/<stem>.html；兼容旧「<dir>/<name>（带后缀）」布局
func sceneHTMLPath(canvasPath, name string) string {
	p := filepath.Join(sceneDir(canvasPath, name), sceneStem(name)+".html")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return filepath.Join(canvasPath, name) // 旧布局：画布目录直接存放
}

// uniqueSceneFile 子文件夹内重名自动加 Windows 风格后缀 (2)、(3)…，保证不覆盖、不重名。
func uniqueSceneFile(canvasPath, fileName string) string {
	for n := 2; ; n++ {
		dir := sceneDir(canvasPath, fileName)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fileName
		}
		fileName = fmt.Sprintf("%s (%d)", sceneStem(fileName), n)
	}
}

// AddScene 已移除：节点统一由「初始节点」与右侧加号（CreateNextNode）创建。

// UpdateScene 更新 HTML 框的名称、位置与宽度；改名时同步移动磁盘文件。
func (s *NodeService) UpdateScene(id uint, name string, x, y, width float64) (*models.HtmlScene, error) {
	var scene models.HtmlScene
	if err := s.db.First(&scene, id).Error; err != nil {
		return nil, errors.New("HTML 框不存在")
	}
	var canvas models.Canvas
	if err := s.db.First(&canvas, scene.CanvasID).Error; err != nil {
		return nil, errors.New("所属画布不存在")
	}
	scene.X, scene.Y = x, y
	if width > 0 {
		scene.Width = width
	}
	if name != "" {
		n := sanitizeFileName(strings.TrimSpace(name))
		// 节点名称即文件夹名：去掉 .html/.htm 后缀（兼容前端传旧格式名）
		n = strings.TrimSuffix(strings.TrimSuffix(n, ".html"), ".htm")
		if n != "" && n != sceneStem(scene.Name) {
			oldStem := sceneStem(scene.Name)
			newStem := n
			oldDir := filepath.Join(canvas.Path, oldStem)
			newDir := filepath.Join(canvas.Path, newStem)
			if oldDir != newDir {
				if _, err := os.Stat(oldDir); err == nil {
					// 子文件夹整体移动（含 html + mp4）
					if err := os.Rename(oldDir, newDir); err != nil {
						return nil, fmt.Errorf("重命名目录失败: %w", err)
					}
					// 内部 HTML 改名（若存在）
					_ = os.Rename(filepath.Join(newDir, scene.Name), filepath.Join(newDir, n+".html"))
					_ = os.Rename(filepath.Join(newDir, oldStem+".html"), filepath.Join(newDir, n+".html"))
				} else {
					// 旧布局：HTML 在外层，移到新子文件夹
					if err := os.MkdirAll(newDir, 0o755); err != nil {
						return nil, fmt.Errorf("创建目录失败: %w", err)
					}
					if err := os.Rename(filepath.Join(canvas.Path, scene.Name), filepath.Join(newDir, n+".html")); err != nil {
						return nil, fmt.Errorf("重命名 HTML 文件失败: %w", err)
					}
				}
			}
			scene.Name = n
		}
	}
	if err := s.db.Save(&scene).Error; err != nil {
		return nil, err
	}
	return &scene, nil
}

// DeleteScene 删除节点文件夹及其端口、连线、磁盘文件；配对端口（连线另一端）级联删除，避免空点。
// DeleteScene 删除节点文件夹及其端口；配对端口（同哈希、位于其它节点的另一端）级联删除，避免空点。
func (s *NodeService) DeleteScene(id uint) error {
	var scene models.HtmlScene
	if err := s.db.First(&scene, id).Error; err != nil {
		return errors.New("HTML 框不存在")
	}
	var canvas models.Canvas
	if err := s.db.First(&canvas, scene.CanvasID).Error; err == nil {
		_ = os.RemoveAll(filepath.Join(canvas.Path, sceneStem(scene.Name))) // 子文件夹（含 html + mp4）
		_ = os.Remove(filepath.Join(canvas.Path, scene.Name))               // 兼容旧布局
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var nodes []models.Node
		if err := tx.Where("scene_id = ?", id).Find(&nodes).Error; err != nil {
			return err
		}
		hashes := make([]string, 0, len(nodes))
		for _, n := range nodes {
			hashes = append(hashes, n.Hash)
		}
		if len(hashes) > 0 {
			// 配对端口（同哈希、位于其它节点的另一端）也要删除，避免出现空点
			if err := tx.Where("hash IN ?", hashes).Delete(&models.Node{}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&models.HtmlScene{}, id).Error
	})
}

// pairNode 返回与 node 哈希相同的配对端口（另一端）；没有则返回 false。
func (s *NodeService) pairNode(node *models.Node) (*models.Node, bool) {
	var peer models.Node
	if err := s.db.Where("hash = ? AND id <> ?", node.Hash, node.ID).First(&peer).Error; err != nil {
		return nil, false
	}
	return &peer, true
}

// SaveSceneVideo 保存该 HTML 框关联的 mp4 到其同名子文件夹。
func (s *NodeService) SaveSceneVideo(id uint, r io.Reader) (*models.HtmlScene, error) {
	var scene models.HtmlScene
	if err := s.db.First(&scene, id).Error; err != nil {
		return nil, errors.New("HTML 框不存在")
	}
	var canvas models.Canvas
	if err := s.db.First(&canvas, scene.CanvasID).Error; err != nil {
		return nil, errors.New("所属画布不存在")
	}
	dir := sceneDir(canvas.Path, scene.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}
	videoName := sceneStem(scene.Name) + ".mp4"
	out, err := os.Create(filepath.Join(dir, videoName))
	if err != nil {
		return nil, fmt.Errorf("保存视频失败: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, r); err != nil {
		return nil, fmt.Errorf("写入视频失败: %w", err)
	}
	scene.Video = videoName
	if err := s.db.Save(&scene).Error; err != nil {
		return nil, err
	}
	return &scene, nil
}

// ---------------------------------------------------------------- 节点

// uniqueNodeName 已移除：出/入端口命名由 CreateNextNode 与魔法连接内部保证不重名。

// UpdateNode 更新节点（名称/坐标/跳转类型）。
func (s *NodeService) UpdateNode(id uint, name string, x, y float64, endingContent *string, targetKind *string) (*models.Node, error) {
	var node models.Node
	if err := s.db.First(&node, id).Error; err != nil {
		return nil, errors.New("节点不存在")
	}
	if name != "" {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return nil, errors.New("节点名称不能为空")
		}
		// 同一 HTML 框内节点名称不能重复
		var c int64
		s.db.Model(&models.Node{}).
			Where("scene_id = ? AND name = ? AND id <> ?", node.SceneID, trimmed, node.ID).
			Count(&c)
		if c > 0 {
			return nil, errors.New("同一 HTML 框内节点名称不能重复")
		}
		node.Name = trimmed
	}
	if targetKind != nil {
		kind := strings.TrimSpace(*targetKind)
		if kind != "mp4" && kind != "html" {
			return nil, errors.New("跳转类型只能是 mp4 或 html")
		}
		node.TargetKind = kind
	}
	if endingContent != nil {
		node.EndingContent = *endingContent
	}
	node.X = x
	node.Y = y
	if err := s.db.Save(&node).Error; err != nil {
		return nil, err
	}
	// 一对端口（出+入）配置共同：名字/结局/跳转类型同步到配对端口
	s.syncPortPair(&node)
	return &node, nil
}

// syncPortPair 把端口的名字/结局内容/跳转类型同步到与其配对的端口（哈希相同的另一端）。
func (s *NodeService) syncPortPair(node *models.Node) {
	peer, ok := s.pairNode(node)
	if !ok {
		return
	}
	_ = s.db.Model(&models.Node{}).Where("id = ?", peer.ID).Updates(map[string]interface{}{
		"name":           node.Name,
		"ending_content": node.EndingContent,
		"target_kind":    node.TargetKind,
	})
}

// DeleteNode 删除端口，并级联处理配对端口（哈希相同的另一端），保证不出现空出点/空入点。
//   - 删出端口 → 连带删除配对入端口所在节点的整棵子树；
//   - 删入端口 / 孤立端口 → 级联删除配对端口，保留目标节点。
func (s *NodeService) DeleteNode(id uint) error {
	var node models.Node
	if err := s.db.First(&node, id).Error; err != nil {
		return errors.New("节点不存在")
	}
	peer, hasPeer := s.pairNode(&node)

	// 规则 1：删普通出端口 → 连带删除配对入端口所在节点的整棵子树（魔法出端口不删子树）
	if !node.Entry && !node.Magic && hasPeer {
		return s.deleteSceneSubtree(peer.SceneID)
	}

	// 规则 2：入端口 / 魔法端口 / 孤立端口 → 级联删除配对端口，保留目标节点
	if hasPeer {
		if err := s.db.Delete(&models.Node{}, peer.ID).Error; err != nil {
			return err
		}
		_ = s.recalcScenePorts(peer.SceneID)
	}
	if err := s.db.Delete(&models.Node{}, id).Error; err != nil {
		return err
	}
	return s.recalcScenePorts(node.SceneID)
}

// BreakConnection 断开端口连接：删除当前端口与其配对端口（同哈希），保留所在节点（不删子树）。
func (s *NodeService) BreakConnection(id uint) error {
	var node models.Node
	if err := s.db.First(&node, id).Error; err != nil {
		return errors.New("节点不存在")
	}
	peer, hasPeer := s.pairNode(&node)
	if hasPeer {
		if err := s.db.Delete(&models.Node{}, peer.ID).Error; err != nil {
			return err
		}
		_ = s.recalcScenePorts(peer.SceneID)
	}
	if err := s.db.Delete(&models.Node{}, id).Error; err != nil {
		return err
	}
	return s.recalcScenePorts(node.SceneID)
}

// RedirectNode 重定向端口的一端到指定节点（按节点哈希），改变该端口的「上一/下一节点」。
//   - side="prev"：移动出端口 → 修改「上一节点」
//   - side="next"：移动入端口 → 修改「下一节点」
//
// 本质是「节点—端口—节点」：端口只有一个，出/入只是前端展示，这一端连向谁由哈希重定向决定。
func (s *NodeService) RedirectNode(id uint, targetHash, side string) error {
	var node models.Node
	if err := s.db.First(&node, id).Error; err != nil {
		return errors.New("节点不存在")
	}
	var target models.HtmlScene
	if err := s.db.Where("hash = ?", targetHash).First(&target).Error; err != nil {
		return errors.New("未找到该节点哈希")
	}
	peer, hasPeer := s.pairNode(&node)
	if !hasPeer {
		return errors.New("该端口没有配对的另一端，无法重定向")
	}
	// 出端口 = 上一节点端；入端口 = 下一节点端
	outP, inP := &node, peer
	if node.Entry {
		outP, inP = peer, &node
	}
	var move *models.Node
	var keepScene uint
	if side == "prev" {
		move = outP
		keepScene = inP.SceneID
	} else {
		move = inP
		keepScene = outP.SceneID
	}
	if keepScene == target.ID {
		return errors.New("不能连回同一节点")
	}
	oldScene := move.SceneID
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// 目标节点内已有同名端口 → 移动端与配对端一起改名（配对两端名字保持相同）
		var c int64
		if err := tx.Model(&models.Node{}).
			Where("scene_id = ? AND name = ? AND id <> ?", target.ID, move.Name, move.ID).
			Count(&c).Error; err != nil {
			return err
		}
		if c > 0 {
			newName := move.Name
			for i := 2; ; i++ {
				newName = fmt.Sprintf("%s %d", move.Name, i)
				var cc int64
				if err := tx.Model(&models.Node{}).Where("scene_id = ? AND name = ?", target.ID, newName).
					Count(&cc).Error; err != nil {
					return err
				}
				if cc == 0 {
					break
				}
			}
			if err := tx.Model(&models.Node{}).Where("id IN ?", []uint{move.ID, peer.ID}).
				Update("name", newName).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.Node{}).Where("id = ?", move.ID).Update("scene_id", target.ID).Error
	}); err != nil {
		return err
	}
	_ = s.recalcScenePorts(oldScene)
	_ = s.recalcScenePorts(target.ID)
	return nil
}

// deleteSceneSubtree 递归删除以 rootID 为根的整棵子树（沿出端口的配对），
// 每个节点（Scene）用 DeleteScene 连带其所有端口与配对端口、磁盘文件。
func (s *NodeService) deleteSceneSubtree(rootID uint) error {
	visited := map[uint]bool{}
	var collect func(id uint) error
	collect = func(id uint) error {
		if visited[id] {
			return nil
		}
		visited[id] = true
		// 先递归删除该 Scene 出端口连到的所有子 Scene
		var outNodes []models.Node
		if err := s.db.Where("scene_id = ? AND entry = ?", id, false).Find(&outNodes).Error; err != nil {
			return err
		}
		for _, n := range outNodes {
			if peer, ok := s.pairNode(&n); ok {
				if err := collect(peer.SceneID); err != nil {
					return err
				}
			}
		}
		return s.DeleteScene(id)
	}
	return collect(rootID)
}

// SetFirstScene 把某 HTML 框设为全局唯一的「开始」区块；其他区块全部取消。
func (s *NodeService) SetFirstScene(id uint) (*models.HtmlScene, error) {
	var scene models.HtmlScene
	if err := s.db.First(&scene, id).Error; err != nil {
		return nil, errors.New("HTML 框不存在")
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.HtmlScene{}).Where("is_first = ?", true).Update("is_first", false).Error; err != nil {
			return err
		}
		scene.IsFirst = true
		return tx.Save(&scene).Error
	}); err != nil {
		return nil, err
	}
	return &scene, nil
}

// ---------------------------------------------------------------- 端口统计

// recalcScenePorts 重算某 HTML 框的三个端口统计字段：
// out_count（有配对的出端口数）、in_count（有配对的入端口数）、has_magic（是否有魔法端口）。
func (s *NodeService) recalcScenePorts(sceneID uint) error {
	var nodes []models.Node
	if err := s.db.Where("scene_id = ?", sceneID).Find(&nodes).Error; err != nil {
		return err
	}
	outCount, inCount, magicCount := 0, 0, 0
	for _, n := range nodes {
		if n.Magic {
			magicCount++
		}
		_, hasPeer := s.pairNode(&n)
		if !hasPeer {
			continue
		}
		if n.Entry {
			inCount++
		} else {
			outCount++
		}
	}
	return s.db.Model(&models.HtmlScene{}).Where("id = ?", sceneID).Updates(map[string]interface{}{
		"out_count": outCount,
		"in_count":  inCount,
		"has_magic": magicCount > 0,
	}).Error
}

// CreateInitialScene 给空白画布创建初始节点（彻底的空文件夹，不含任何端口）。
// 出端口由右侧加号（CreateNextNode）成对创建，保证不出现空出点/空入点。
// 画布内已有节点时不动，返回 nil。
func (s *NodeService) CreateInitialScene(canvasID uint) (*models.HtmlScene, error) {
	var count int64
	if err := s.db.Model(&models.HtmlScene{}).Where("canvas_id = ?", canvasID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, nil
	}
	var canvas models.Canvas
	if err := s.db.First(&canvas, canvasID).Error; err != nil {
		return nil, errors.New("所属画布不存在")
	}
	fileName := uniqueSceneFile(canvas.Path, "初始节点")
	dir := sceneDir(canvas.Path, fileName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}
	// 彻底的空文件夹：不放任何占位文件，不预建端口
	scene := &models.HtmlScene{CanvasID: canvasID, Hash: NewHash(), Name: fileName, X: 60, Y: 120}
	if err := s.db.Create(scene).Error; err != nil {
		return nil, err
	}
	return scene, nil
}

// CreateNextNode 从节点的右侧加号创建下一跳：
// 在当前节点新增一个出端口，同时在其右侧新建一个空白节点（自动带入端口）。
// 出/入端口共用同一个哈希（配对端口），哈希相同即相连。
// 返回：新节点、出端口、入端口。
func (s *NodeService) CreateNextNode(sceneID uint) (*models.HtmlScene, *models.Node, *models.Node, error) {
	var scene models.HtmlScene
	if err := s.db.First(&scene, sceneID).Error; err != nil {
		return nil, nil, nil, errors.New("节点不存在")
	}
	var canvas models.Canvas
	if err := s.db.First(&canvas, scene.CanvasID).Error; err != nil {
		return nil, nil, nil, errors.New("所属画布不存在")
	}
	// 1. 当前节点新增出端口（名称不重名：出、出 2、…）
	outName := "出"
	for n := 2; ; n++ {
		var c int64
		s.db.Model(&models.Node{}).Where("scene_id = ? AND name = ?", sceneID, outName).Count(&c)
		if c == 0 {
			break
		}
		outName = fmt.Sprintf("出 %d", n)
	}
	pairHash := NewHash() // 配对端口共用同一哈希（同一 IP）
	outNode := &models.Node{Hash: pairHash, Name: outName, SceneID: sceneID, TargetKind: "mp4", Entry: false}

	// 2. 新空白节点：右侧下一跳（彻底的空文件夹，HTML/MP4 后期导入）
	fileName := uniqueSceneFile(canvas.Path, "新节点")
	dir := sceneDir(canvas.Path, fileName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, nil, fmt.Errorf("创建目录失败: %w", err)
	}
	newScene := &models.HtmlScene{
		CanvasID: scene.CanvasID,
		Hash:     NewHash(),
		Name:     fileName,
		X:        scene.X + 440, // 父节点右侧（前端 relayout 会重排覆盖）
		Y:        scene.Y,
	}

	// 3. 新节点入端口：同名 + 与出端口同一哈希（事务）
	entry := &models.Node{Hash: pairHash, Name: outName, SceneID: newScene.ID, Entry: true}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(outNode).Error; err != nil {
			return err
		}
		if err := tx.Create(newScene).Error; err != nil {
			return err
		}
		entry.SceneID = newScene.ID
		return tx.Create(entry).Error
	}); err != nil {
		return nil, nil, nil, err
	}
	_ = s.recalcScenePorts(sceneID)
	_ = s.recalcScenePorts(newScene.ID)
	return newScene, outNode, entry, nil
}

// CreateMagicPort 新建魔法出端口并连向指定哈希的目标节点（画布不画线、导出生效）。
// 出端口名「魔」（不重名）；目标节点自动生成同名入端口，两端共用同一哈希。
func (s *NodeService) CreateMagicPort(sceneID uint, targetHash string) (*models.Node, *models.Node, error) {
	var scene models.HtmlScene
	if err := s.db.First(&scene, sceneID).Error; err != nil {
		return nil, nil, errors.New("所属节点不存在")
	}
	var target models.HtmlScene
	if err := s.db.Where("hash = ?", targetHash).First(&target).Error; err != nil {
		return nil, nil, errors.New("未找到该节点哈希")
	}
	if scene.ID == target.ID {
		return nil, nil, errors.New("不能连回自身节点")
	}
	// 出端口名「魔」不重名（魔、魔 2、…）
	name := "魔"
	for n := 2; ; n++ {
		var c int64
		s.db.Model(&models.Node{}).Where("scene_id = ? AND name = ?", sceneID, name).Count(&c)
		if c == 0 {
			break
		}
		name = fmt.Sprintf("魔 %d", n)
	}
	pairHash := NewHash()
	out := &models.Node{Hash: pairHash, Name: name, SceneID: sceneID, TargetKind: "mp4", Entry: false, Magic: true}
	if err := s.db.Create(out).Error; err != nil {
		return nil, nil, err
	}
	entry := &models.Node{Hash: pairHash, Name: name, SceneID: target.ID, Entry: true, Magic: true}
	if err := s.db.Create(entry).Error; err != nil {
		_ = s.db.Delete(&models.Node{}, out.ID) // 回滚：入端口失败则删掉刚建的出端口
		return nil, nil, err
	}
	_ = s.recalcScenePorts(sceneID)
	_ = s.recalcScenePorts(target.ID)
	return out, entry, nil
}

// UpdateSceneHTML 覆盖节点的 HTML 文件内容（文件名固定为 <文件夹名>.html，缺失时创建）。
func (s *NodeService) UpdateSceneHTML(id uint, data []byte) (*models.HtmlScene, error) {
	var scene models.HtmlScene
	if err := s.db.First(&scene, id).Error; err != nil {
		return nil, errors.New("节点不存在")
	}
	var canvas models.Canvas
	if err := s.db.First(&canvas, scene.CanvasID).Error; err != nil {
		return nil, errors.New("所属画布不存在")
	}
	dir := sceneDir(canvas.Path, scene.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, sceneStem(scene.Name)+".html"), data, 0o644); err != nil {
		return nil, fmt.Errorf("写入 HTML 失败: %w", err)
	}
	return &scene, nil
}
