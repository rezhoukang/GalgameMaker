package service

import (
	"crypto/hmac"
	"errors"
	"fmt"
	"galgame-maker/internal/models"
	"gorm.io/gorm"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type NodeService struct{ db *gorm.DB }

func NewNodeService(db *gorm.DB) *NodeService        { return &NodeService{db: db} }
func sceneStem(name string) string                   { return strings.TrimSuffix(name, filepath.Ext(name)) }
func assetDir(c models.Canvas, s models.Node) string { return filepath.Join(c.Path, s.AssetDir) }
func htmlPath(c models.Canvas, s models.Node) string {
	if s.HTMLFile == "" {
		return ""
	}
	return filepath.Join(assetDir(c, s), s.HTMLFile)
}
func graph(tx *gorm.DB, canvasID uint) ([]models.Node, []models.Port, error) {
	var canvas models.Canvas
	if err := tx.First(&canvas, canvasID).Error; err != nil {
		return nil, nil, err
	}
	ss, pp := []models.Node{}, []models.Port{}
	if err := tx.Where("canvas_id = ?", canvasID).Order("id").Find(&ss).Error; err != nil {
		return nil, nil, err
	}
	ids := []uint{}
	for _, sc := range ss {
		ids = append(ids, sc.ID)
	}
	if len(ids) > 0 {
		if err := tx.Where("node_id IN ?", ids).Order("sort_order, id").Find(&pp).Error; err != nil {
			return nil, nil, err
		}
	}
	return ss, pp, nil
}
func normalize(tx *gorm.DB, canvasID uint) error {
	ss, pp, err := graph(tx, canvasID)
	if err != nil {
		return err
	}
	if err := models.ValidateGraph(ss, pp); err != nil {
		return err
	}
	for _, sc := range ss {
		if err := tx.Model(&models.Node{}).Where("id = ?", sc.ID).Update("level", sc.Level).Error; err != nil {
			return err
		}
	}
	return nil
}
func (s *NodeService) UpdateScene(id uint, name string, x, y float64) (*models.Node, error) {
	var scene *models.Node
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		scene, err = NewNodeService(tx).updateScene(id, name, x, y)
		return err
	})
	return scene, err
}
func (s *NodeService) updateScene(id uint, name string, x, y float64) (*models.Node, error) {
	var sc models.Node
	if err := activeScenes(s.db).First(&sc, id).Error; err != nil {
		return nil, errors.New("节点不存在")
	}
	if name != "" {
		sc.Name = strings.TrimSpace(name)
		if sc.Name == "" {
			return nil, errors.New("节点名称不能为空")
		}
	}
	if err := s.db.Save(&sc).Error; err != nil {
		return nil, err
	}
	return &sc, nil
}

// Delete only the ordinary subtree; immutable resources remain available to undo.
func (s *NodeService) DeleteScene(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var sc models.Node
		if err := activeScenes(tx).First(&sc, id).Error; err != nil {
			return errors.New("节点不存在")
		}
		if sc.IsRoot {
			return errors.New("根节点不可删除")
		}
		ss, pp, err := graph(tx, sc.CanvasID)
		if err != nil {
			return err
		}
		byHash := map[string]uint{}
		for _, n := range ss {
			byHash[n.Hash] = n.ID
		}
		children := map[uint][]uint{}
		for _, p := range pp {
			if !p.Magic {
				children[p.NodeID] = append(children[p.NodeID], byHash[p.ExportHash])
			}
		}
		removed := map[uint]bool{}
		queue := []uint{id}
		for len(queue) > 0 {
			n := queue[0]
			queue = queue[1:]
			if removed[n] {
				continue
			}
			removed[n] = true
			queue = append(queue, children[n]...)
		}
		ids, hashes := []uint{}, []string{}
		for _, n := range ss {
			if removed[n.ID] {
				if n.IsRoot {
					return errors.New("不能删除根节点")
				}
				ids = append(ids, n.ID)
				hashes = append(hashes, n.Hash)
			}
		}
		if err := tx.Where("node_id IN ? OR export_hash IN ?", ids, hashes).Delete(&models.Port{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ids).Delete(&models.Node{}).Error; err != nil {
			return err
		}
		var first int64
		tx.Model(&models.Node{}).Where("canvas_id = ? AND is_first = ?", sc.CanvasID, true).Count(&first)
		if first == 0 {
			if err := tx.Model(&models.Node{}).Where("canvas_id = ? AND is_root = ?", sc.CanvasID, true).Update("is_first", true).Error; err != nil {
				return err
			}
		}
		return normalize(tx, sc.CanvasID)
	})
}
func (s *NodeService) UpdateNode(id uint, name string, targetKind *string, showOption *bool) (*models.Port, error) {
	var p models.Port
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := activePorts(tx).First(&p, id).Error; err != nil {
			return errors.New("出端口不存在")
		}
		if name != "" {
			p.Name = strings.TrimSpace(name)
			if p.Name == "" {
				return errors.New("选项内容不能为空")
			}
		}
		if targetKind != nil {
			p.TargetKind = *targetKind
		}
		if showOption != nil {
			p.ShowOption = *showOption
		}
		if p.TargetKind != "mp4" && p.TargetKind != "html" {
			return errors.New("内容类型只能是 mp4 或 html")
		}
		return tx.Save(&p).Error
	})
	return &p, err
}
func (s *NodeService) DeleteNode(id uint) error {
	var p models.Port
	if err := activePorts(s.db).First(&p, id).Error; err != nil {
		return errors.New("出端口不存在")
	}
	if p.Magic {
		return s.BreakConnection(id)
	}
	var target models.Node
	if err := activeScenes(s.db).Where("hash = ?", p.ExportHash).First(&target).Error; err != nil {
		return err
	}
	return s.DeleteScene(target.ID)
}
func (s *NodeService) BreakConnection(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var p models.Port
		if err := activePorts(tx).First(&p, id).Error; err != nil {
			return err
		}
		if !p.Magic {
			return errors.New("普通分支不能悬空，请修改上一节点移动分支，或删除分支")
		}
		var sc models.Node
		if err := activeScenes(tx).First(&sc, p.NodeID).Error; err != nil {
			return err
		}
		if err := tx.Delete(&p).Error; err != nil {
			return err
		}
		return normalize(tx, sc.CanvasID)
	})
}
func (s *NodeService) RedirectNode(id uint, hash, side string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var p models.Port
		var from, to models.Node
		if err := activePorts(tx).First(&p, id).Error; err != nil {
			return err
		}
		if err := activeScenes(tx).First(&from, p.NodeID).Error; err != nil {
			return err
		}
		if err := tx.Where("hash = ? AND canvas_id = ?", strings.TrimSpace(hash), from.CanvasID).First(&to).Error; err != nil {
			return errors.New("目标节点不存在于当前工程")
		}
		if side == "prev" {
			p.NodeID = to.ID
		} else if side == "next" {
			p.ExportHash = to.Hash
		} else {
			return errors.New("连接方向无效")
		}
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		return normalize(tx, from.CanvasID)
	})
}
func (s *NodeService) SetFirstScene(id uint) (*models.Node, error) {
	var sc models.Node
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := activeScenes(tx).First(&sc, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Node{}).Where("canvas_id = ?", sc.CanvasID).Update("is_first", false).Error; err != nil {
			return err
		}
		sc.IsFirst = true
		return tx.Save(&sc).Error
	})
	return &sc, err
}
func (s *NodeService) CreateInitialScene(canvasID uint) (*models.Node, error) {
	var sc *models.Node
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var canvas models.Canvas
		if err := tx.First(&canvas, canvasID).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&models.Node{}).Where("canvas_id = ?", canvasID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		hash := NewHash()
		sc = &models.Node{CanvasID: canvasID, Hash: hash, AssetDir: hash, Name: "初始节点", Level: 1, IsRoot: true, IsFirst: true}
		return tx.Create(sc).Error
	})
	return sc, err
}
func nextPort(tx *gorm.DB, sc models.Node, target string, magic bool) models.Port {
	var count int64
	tx.Model(&models.Port{}).Where("node_id = ?", sc.ID).Count(&count)
	var max int
	tx.Model(&models.Port{}).Where("node_id = ?", sc.ID).Select("COALESCE(MAX(sort_order), 0)").Scan(&max)
	return models.Port{Hash: NewHash(), Name: fmt.Sprintf("选项 %d", count+1), NodeID: sc.ID, ExportHash: target, Magic: magic, ShowOption: true, TargetKind: "mp4", SortOrder: max + 1}
}
func (s *NodeService) CreateNextNode(sceneID uint) (*models.Node, *models.Port, *models.Port, error) {
	var sc models.Node
	var p models.Port
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var parent models.Node
		if err := activeScenes(tx).First(&parent, sceneID).Error; err != nil {
			return err
		}
		hash := NewHash()
		sc = models.Node{CanvasID: parent.CanvasID, Hash: hash, AssetDir: hash, Name: "新节点", Level: parent.Level + 1}
		if err := tx.Create(&sc).Error; err != nil {
			return err
		}
		p = nextPort(tx, parent, sc.Hash, false)
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		return normalize(tx, parent.CanvasID)
	})
	return &sc, &p, nil, err
}
func (s *NodeService) CreateMagicPort(sceneID uint, hash string) (*models.Port, *models.Port, error) {
	var p models.Port
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var from, to models.Node
		if err := activeScenes(tx).First(&from, sceneID).Error; err != nil {
			return err
		}
		if err := tx.Where("hash = ? AND canvas_id = ?", strings.TrimSpace(hash), from.CanvasID).First(&to).Error; err != nil {
			return errors.New("目标节点不存在于当前工程")
		}
		p = nextPort(tx, from, to.Hash, true)
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		return normalize(tx, from.CanvasID)
	})
	return &p, nil, err
}

// Every upload is immutable, allowing snapshots to restore earlier content.
func (s *NodeService) saveContent(id uint, r io.Reader, ext string) (*models.Node, error) {
	var scene *models.Node
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		scene, err = NewNodeService(tx).saveContentTx(id, r, ext)
		return err
	})
	return scene, err
}
func (s *NodeService) saveContentTx(id uint, r io.Reader, ext string) (*models.Node, error) {
	var sc models.Node
	var canvas models.Canvas
	if err := activeScenes(s.db).First(&sc, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&canvas, sc.CanvasID).Error; err != nil {
		return nil, err
	}
	name := filepath.ToSlash(filepath.Join(".versions", NewHash()+ext))
	path := filepath.Join(assetDir(canvas, sc), name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	n, err := io.Copy(f, r)
	closeErr := f.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if n == 0 {
		return nil, errors.New("素材文件不能为空")
	}
	if ext == ".mp4" {
		sc.Video = name
	} else {
		sc.HTMLFile = name
	}
	if err := s.db.Save(&sc).Error; err != nil {
		return nil, err
	}
	return &sc, nil
}
func (s *NodeService) SaveSceneVideo(id uint, r io.Reader) (*models.Node, error) {
	return s.saveContent(id, r, ".mp4")
}
func (s *NodeService) UpdateSceneHTML(id uint, data []byte) (*models.Node, error) {
	return s.saveContent(id, strings.NewReader(string(data)), ".html")
}

// Copy the signed content captured at copy time, even if its source was later edited.
func (s *NodeService) CopyProperties(id, sourceID uint, snapshot CanvasView) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		token, err := signView(tx, snapshot)
		if err != nil {
			return err
		}
		if !hmac.Equal([]byte(token), []byte(snapshot.Token)) {
			return errors.New("复制的属性无效")
		}
		var dst models.Node
		if err := activeScenes(tx).First(&dst, id).Error; err != nil {
			return err
		}
		if dst.CanvasID != snapshot.CanvasID {
			return errors.New("仅支持同工程复制属性")
		}
		for _, src := range snapshot.Scenes {
			if src.ID == sourceID {
				dst.AssetDir, dst.Video, dst.HTMLFile = src.AssetDir, src.Video, src.HTMLFile
				return tx.Save(&dst).Error
			}
		}
		return errors.New("复制的节点不存在")
	})
}

// Nodes/ports remain hard-deleted. Their owning project controls access.
func activeScenes(db *gorm.DB) *gorm.DB {
	return db.Where("canvas_id IN (?)", db.Model(&models.Canvas{}).Select("id"))
}
func activePorts(db *gorm.DB) *gorm.DB {
	return db.Where("node_id IN (?)", activeScenes(db).Model(&models.Node{}).Select("id"))
}
