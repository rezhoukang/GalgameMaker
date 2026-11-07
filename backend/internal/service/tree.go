package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"galgame-maker/internal/models"
)

// TreeService 管理 文件夹/画布 的树结构与真实文件系统映射。
type TreeService struct {
	db       *gorm.DB
	settings *SettingsService
}

// NewTreeService 创建树服务。
func NewTreeService(db *gorm.DB, settings *SettingsService) *TreeService {
	return &TreeService{db: db, settings: settings}
}

// ---------------------------------------------------------------- 树查询

// TreeCanvas 树中的画布节点。
type TreeCanvas struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	NodeCount int    `json:"nodeCount"`
}

// TreeFolder 树中的文件夹节点。
type TreeFolder struct {
	ID       uint         `json:"id"`
	ParentID *uint        `json:"parentId"`
	Name     string       `json:"name"`
	Canvases []TreeCanvas `json:"canvases"`
}

// GetTree 返回完整树结构（文件夹 → 画布）。
func (s *TreeService) GetTree() ([]TreeFolder, error) {
	var folders []models.Folder
	if err := s.db.Order("created_at ASC").Find(&folders).Error; err != nil {
		return nil, err
	}

	type canvasRow struct {
		ID        uint
		FolderID  uint
		Name      string
		NodeCount int
	}
	var rows []canvasRow
	if err := s.db.Model(&models.Canvas{}).
		Select("canvas.id, canvas.folder_id, canvas.name, COUNT(DISTINCT scenes.id) AS node_count").
		Joins("LEFT JOIN scenes ON scenes.canvas_id = canvas.id").
		Group("canvas.id").Scan(&rows).Error; err != nil {
		return nil, err
	}

	canvasByFolder := map[uint][]TreeCanvas{}
	for _, r := range rows {
		canvasByFolder[r.FolderID] = append(canvasByFolder[r.FolderID], TreeCanvas{
			ID: r.ID, Name: r.Name, NodeCount: r.NodeCount,
		})
	}

	tree := make([]TreeFolder, 0, len(folders))
	for _, f := range folders {
		tree = append(tree, TreeFolder{
			ID: f.ID, ParentID: f.ParentID, Name: f.Name,
			Canvases: canvasByFolder[f.ID],
		})
	}
	return tree, nil
}

// CanvasView 打开画布时返回的完整视图：HTML 框 + 端口。
// 连接关系不再有独立连线表：一对「出端口 + 入端口」共用同一哈希，哈希相同即相连。
type CanvasView struct {
	CanvasID uint           `json:"canvasId"`
	Scenes   []models.Scene `json:"scenes"`
	Nodes    []models.Port  `json:"nodes"`
	Token    string         `json:"token"`
}

func (s *TreeService) GetCanvasView(canvasID uint) (*CanvasView, error) {
	return readView(s.db, canvasID)
}

// ---------------------------------------------------------------- 创建

// CreateFolder 新建文件夹。
func (s *TreeService) CreateFolder(parentID *uint, name string) (*models.Folder, error) {
	var item *models.Folder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = NewTreeService(tx, s.settings).createFolder(parentID, name)
		return err
	})
	return item, err
}
func (s *TreeService) createFolder(parentID *uint, name string) (*models.Folder, error) {
	if err := s.settings.EnsureStorageConfigured(); err != nil {
		return nil, err
	}
	name = sanitizeFileName(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." {
		return nil, errors.New("名称无效")
	}
	base := s.settings.GetStoragePath()

	targetDir := base
	if parentID != nil {
		var parent models.Folder
		if err := s.db.First(&parent, *parentID).Error; err != nil {
			return nil, errors.New("父文件夹不存在")
		}
		targetDir = parent.Path
	}

	dir := filepath.Join(targetDir, name)
	if _, err := os.Stat(dir); err == nil {
		return nil, errors.New("同名文件夹已存在")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建文件夹失败: %w", err)
	}

	folder := &models.Folder{ParentID: parentID, Name: name, Path: dir}
	if err := s.db.Create(folder).Error; err != nil {
		return nil, err
	}
	return folder, nil
}

// CreateCanvas 新建画布（树里的文件）。画布是一个目录，用于存放 HTML 文件。
func (s *TreeService) CreateCanvas(folderID uint, name string) (*models.Canvas, error) {
	var item *models.Canvas
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = NewTreeService(tx, s.settings).createCanvas(folderID, name)
		return err
	})
	return item, err
}
func (s *TreeService) createCanvas(folderID uint, name string) (*models.Canvas, error) {
	if err := s.settings.EnsureStorageConfigured(); err != nil {
		return nil, err
	}
	var folder models.Folder
	if err := s.db.First(&folder, folderID).Error; err != nil {
		return nil, errors.New("所属文件夹不存在")
	}
	name = sanitizeFileName(strings.TrimSpace(name))
	if name == "" {
		name = "未命名"
	}

	// 文件夹内重名自动加序号
	count := 0
	baseName := name
	for {
		var c int64
		if err := s.db.Unscoped().Model(&models.Canvas{}).Where("folder_id = ? AND name = ?", folderID, name).Count(&c).Error; err != nil {
			return nil, err
		}
		if c == 0 {
			_, err := os.Stat(filepath.Join(folder.Path, name+"PIC"))
			if os.IsNotExist(err) {
				break
			}
			if err != nil {
				return nil, err
			}
		}
		count++
		name = fmt.Sprintf("%s(%d)", baseName, count)
	}

	// 画布真实目录：目录名 = 画布名 + PIC 后缀（前端只显示名字）
	dir := filepath.Join(folder.Path, name+"PIC")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建画布目录失败: %w", err)
	}

	canvas := &models.Canvas{FolderID: folderID, Name: name, Path: dir}
	if err := s.db.Create(canvas).Error; err != nil {
		return nil, err
	}
	return canvas, nil
}

// ---------------------------------------------------------------- 重命名

// RenameItem 重命名（folder / canvas）。
func (s *TreeService) RenameItem(kind string, id uint, newName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return NewTreeService(tx, s.settings).renameItem(kind, id, newName)
	})
}
func (s *TreeService) renameItem(kind string, id uint, newName string) error {
	newName = sanitizeFileName(strings.TrimSpace(newName))
	if newName == "" || newName == "." || newName == ".." {
		return errors.New("名称无效")
	}

	switch kind {
	case "folder":
		var f models.Folder
		if err := s.db.First(&f, id).Error; err != nil {
			return errors.New("文件夹不存在")
		}
		newPath := filepath.Join(filepath.Dir(f.Path), newName)
		if err := checkRenamePath(f.Path, newPath); err != nil {
			return err
		}
		if err := os.Rename(f.Path, newPath); err != nil {
			return fmt.Errorf("重命名文件夹失败: %w", err)
		}
		return s.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.Folder{}).Where("id = ?", id).Updates(
				map[string]any{"name": newName, "path": newPath}).Error; err != nil {
				return err
			}
			// 级联更新其下画布目录路径
			return s.shiftCanvasPaths(tx, f.Path, newPath)
		})

	case "canvas":
		var c models.Canvas
		if err := s.db.First(&c, id).Error; err != nil {
			return errors.New("画布不存在")
		}
		// 目录名保持 PIC 后缀
		newPath := filepath.Join(filepath.Dir(c.Path), newName+"PIC")
		if err := checkRenamePath(c.Path, newPath); err != nil {
			return err
		}
		if err := os.Rename(c.Path, newPath); err != nil {
			return fmt.Errorf("重命名画布失败: %w", err)
		}
		return s.db.Model(&models.Canvas{}).Where("id = ?", id).Updates(
			map[string]any{"name": newName, "path": newPath}).Error

	default:
		return errors.New("不支持的重命名类型")
	}
}

// shiftCanvasPaths 文件夹重命名后，把其下所有画布路径前缀整体替换。
func (s *TreeService) shiftCanvasPaths(tx *gorm.DB, oldPrefix, newPrefix string) error {
	var folders []models.Folder
	if err := tx.Unscoped().Find(&folders).Error; err != nil {
		return err
	}
	for _, f := range folders {
		rel, err := filepath.Rel(oldPrefix, f.Path)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}
		if err := tx.Unscoped().Model(&models.Folder{}).Where("id = ?", f.ID).Update("path", filepath.Join(newPrefix, rel)).Error; err != nil {
			return err
		}
	}
	var canvases []models.Canvas
	if err := tx.Unscoped().Find(&canvases).Error; err != nil {
		return err
	}
	for _, c := range canvases {
		rel, err := filepath.Rel(oldPrefix, c.Path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}
		if err := tx.Unscoped().Model(&models.Canvas{}).Where("id = ?", c.ID).Update("path", filepath.Join(newPrefix, rel)).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------- 删除

// DeleteItem hides folders/projects while retaining their graph and media.
func (s *TreeService) DeleteItem(kind string, id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		deletedAt := tx.NowFunc()
		switch kind {
		case "folder":
			var folder models.Folder
			if err := tx.First(&folder, id).Error; err != nil {
				return errors.New("文件夹不存在或已删除")
			}
			folders := []models.Folder{folder}
			if err := s.collectChildFolders(tx, &folders, id); err != nil {
				return err
			}
			ids := make([]uint, 0, len(folders))
			for _, f := range folders {
				ids = append(ids, f.ID)
			}
			if err := tx.Model(&models.Canvas{}).Where("folder_id IN ?", ids).Update("delete_at", deletedAt).Error; err != nil {
				return err
			}
			return tx.Model(&models.Folder{}).Where("id IN ?", ids).Update("delete_at", deletedAt).Error
		case "canvas":
			var canvas models.Canvas
			if err := tx.First(&canvas, id).Error; err != nil {
				return errors.New("工程不存在或已删除")
			}
			return tx.Model(&canvas).Update("delete_at", deletedAt).Error
		default:
			return errors.New("不支持的删除类型")
		}
	})
}

// collectChildFolders 收集某文件夹的所有子孙文件夹。
func (s *TreeService) collectChildFolders(tx *gorm.DB, acc *[]models.Folder, parentID uint) error {
	var children []models.Folder
	if err := tx.Where("parent_id = ?", parentID).Find(&children).Error; err != nil {
		return err
	}
	for _, c := range children {
		*acc = append(*acc, c)
		if err := s.collectChildFolders(tx, acc, c.ID); err != nil {
			return err
		}
	}
	return nil
}

// ResetAll 初始化：清空数据库、清空存储目录，再创建默认结构。
func (s *TreeService) ResetAll() error {
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM ports").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM scenes").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM canvas").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM folders").Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	// 清空存储目录，再创建默认结构
	if err := s.ClearStorage(); err != nil {
		return err
	}
	_, err := s.EnsureDefaults()
	return err
}

// EnsureDefaults 若项目为空，创建默认的「未分类」文件夹和「新画布」，并返回该画布。
func (s *TreeService) EnsureDefaults() (*models.Canvas, error) {
	var count int64
	if err := s.db.Unscoped().Model(&models.Folder{}).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, nil
	}
	folder, err := s.CreateFolder(nil, "未分类")
	if err != nil {
		return nil, err
	}
	return s.CreateCanvas(folder.ID, "新画布")
}

// ClearStorage 清空本地存储目录中的所有内容（文件夹/画布目录）。
func (s *TreeService) ClearStorage() error {
	root := s.settings.GetStoragePath()
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil // 目录不存在则忽略
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(root, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func checkRenamePath(oldPath, newPath string) error {
	if oldPath == newPath {
		return nil
	}
	if target, err := os.Stat(newPath); err == nil {
		if source, e := os.Stat(oldPath); e == nil && os.SameFile(source, target) {
			return nil
		}
		return errors.New("同名目录已存在（可能属于已删除项目）")
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}
