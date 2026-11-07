package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"galgame-maker/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func signView(db *gorm.DB, view CanvasView) (string, error) {
	var key models.Config
	if err := db.Where("key = ?", "history.signing.key").First(&key).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		key = models.Config{Key: "history.signing.key", Value: hex.EncodeToString(b)}
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&key).Error; err != nil {
			return "", err
		}
		if err := db.First(&key, "key = ?", key.Key).Error; err != nil {
			return "", err
		}
	}
	view.Token = ""
	data, err := json.Marshal(view)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(key.Value))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func readView(db *gorm.DB, canvasID uint) (*CanvasView, error) {
	var canvas models.Canvas
	if err := db.First(&canvas, canvasID).Error; err != nil {
		return nil, errors.New("工程不存在")
	}
	ss, pp, err := graph(db, canvasID)
	if err != nil {
		return nil, err
	}
	v := &CanvasView{CanvasID: canvasID, Scenes: ss, Nodes: pp}
	v.Token, err = signView(db, *v)
	return v, err
}

// Signed snapshots cannot invent asset paths or overwrite another project's records.
// Expected token prevents an old browser tab from overwriting newer edits.
func (s *NodeService) Restore(canvasID uint, expected string, want CanvasView) (*CanvasView, error) {
	var result *CanvasView
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if want.CanvasID != canvasID {
			return errors.New("历史记录不属于当前工程")
		}
		token, err := signView(tx, want)
		if err != nil {
			return err
		}
		if !hmac.Equal([]byte(token), []byte(want.Token)) {
			return errors.New("历史记录无效或已损坏")
		}
		current, err := readView(tx, canvasID)
		if err != nil {
			return err
		}
		if expected == "" || expected != current.Token {
			return errors.New("工程已在其他页面修改，请重新打开工程后操作")
		}
		rootHash := ""
		for _, sc := range current.Scenes {
			if sc.Level == 1 {
				rootHash = sc.Hash
			}
		}
		first := 0
		for _, sc := range want.Scenes {
			if sc.CanvasID != canvasID {
				return errors.New("历史记录包含其他工程节点")
			}
			if sc.Level == 1 && sc.Hash != rootHash {
				return errors.New("不能替换根节点")
			}
			if sc.IsFirst {
				first++
			}
		}
		if first != 1 {
			return errors.New("必须恰好有一个播放起点")
		}
		if err := models.ValidateGraph(want.Scenes, want.Nodes); err != nil {
			return err
		}
		ids := []uint{}
		for _, sc := range current.Scenes {
			ids = append(ids, sc.ID)
		}
		if len(ids) > 0 {
			if err := tx.Where("node_id IN ?", ids).Delete(&models.Port{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("canvas_id = ?", canvasID).Delete(&models.Node{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&want.Scenes).Error; err != nil {
			return err
		}
		// GORM default:true would turn a restored false value into true on INSERT.
		for _, p := range want.Nodes {
			show := p.ShowOption
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
			if !show {
				if err := tx.Model(&p).Update("show_option", false).Error; err != nil {
					return err
				}
			}
		}
		result, err = readView(tx, canvasID)
		return err
	})
	return result, err
}
