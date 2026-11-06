package database

import (
	"fmt"
	"galgame-maker/internal/models"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Legacy tables remain untouched. Only a fully validated conversion is committed.
func migratePorts(db *gorm.DB) error {
	var marker models.Config
	if db.First(&marker, "key = ?", "schema.outgoing.v1").Error == nil {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var scenes []models.HtmlScene
		if err := tx.Find(&scenes).Error; err != nil {
			return err
		}
		byID := map[uint]*models.HtmlScene{}
		for i := range scenes {
			byID[scenes[i].ID] = &scenes[i]
		}
		type oldPort struct {
			ID                        uint
			Hash, Name                string
			SceneID                   uint
			TargetKind, EndingContent string
			Entry, Magic, IsFirst     bool
			CreatedAt, UpdatedAt      time.Time
		}
		var old []oldPort
		if tx.Migrator().HasTable("nodes") {
			if err := tx.Table("nodes").Order("id").Find(&old).Error; err != nil {
				return err
			}
		}
		pairs := map[string][]oldPort{}
		for _, p := range old {
			pairs[p.Hash] = append(pairs[p.Hash], p)
		}
		ports := []models.Node{}
		for _, p := range old {
			pair := pairs[p.Hash]
			if len(pair) != 2 || pair[0].Entry == pair[1].Entry {
				return fmt.Errorf("旧端口 %d 配对异常，迁移已取消，旧数据保留", p.ID)
			}
			if p.Entry {
				continue
			}
			peer := pair[0]
			if peer.ID == p.ID {
				peer = pair[1]
			}
			from, to := byID[p.SceneID], byID[peer.SceneID]
			if from == nil || to == nil || from.CanvasID != to.CanvasID {
				return fmt.Errorf("旧端口 %d 目标缺失或跨工程，迁移已取消", p.ID)
			}
			kind := p.TargetKind
			if kind == "" {
				kind = "mp4"
			}
			name := strings.TrimSpace(p.Name)
			if name == "" {
				name = "选项"
			}
			ports = append(ports, models.Node{ID: p.ID, Hash: p.Hash, Name: name, SceneID: p.SceneID, ExportHash: to.Hash, Magic: p.Magic, ShowOption: true, SortOrder: int(p.ID), TargetKind: kind, EndingContent: p.EndingContent, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt})
		}
		var canvases []models.Canvas
		if err := tx.Find(&canvases).Error; err != nil {
			return err
		}
		for _, canvas := range canvases {
			ss, pp := []models.HtmlScene{}, []models.Node{}
			incoming := map[string]bool{}
			for _, p := range ports {
				if byID[p.SceneID].CanvasID == canvas.ID {
					pp = append(pp, p)
					if !p.Magic {
						incoming[p.ExportHash] = true
					}
				}
			}
			for _, sc := range scenes {
				if sc.CanvasID == canvas.ID {
					ss = append(ss, sc)
				}
			}
			if len(ss) == 0 {
				continue
			}
			root := -1
			for i, sc := range ss {
				if !incoming[sc.Hash] && (root < 0 || sc.IsFirst) {
					root = i
				}
			}
			if root < 0 {
				return fmt.Errorf("工程 %d 普通连接存在环，迁移已取消", canvas.ID)
			}
			ss[root].IsRoot = true
			first := false
			for _, sc := range ss {
				first = first || sc.IsFirst
			}
			if !first {
				ss[root].IsFirst = true
			}
			if err := models.ValidateGraph(ss, pp); err != nil {
				return fmt.Errorf("工程 %d: %w", canvas.ID, err)
			}
			for i := range ss {
				sc := &ss[i]
				sc.AssetDir = strings.TrimSuffix(sc.Name, filepath.Ext(sc.Name))
				if _, err := os.Stat(filepath.Join(canvas.Path, sc.AssetDir, sc.AssetDir+".html")); err == nil {
					sc.HTMLFile = sc.AssetDir + ".html"
				}
				// Older flat HTML layouts stay readable without moving user files.
				if sc.HTMLFile == "" {
					if _, err := os.Stat(filepath.Join(canvas.Path, sc.Name)); err == nil && strings.HasSuffix(sc.Name, ".html") {
						sc.HTMLFile = "../" + sc.Name
					}
				}
				if err := tx.Save(sc).Error; err != nil {
					return err
				}
			}
		}
		if len(ports) > 0 {
			if err := tx.Create(&ports).Error; err != nil {
				return err
			}
		}
		return tx.Create(&models.Config{Key: "schema.outgoing.v1", Value: "1"}).Error
	})
}
