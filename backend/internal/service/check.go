package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"galgame-maker/internal/models"
)

// CanvasIssue 画布检测问题（导出前检测用）。
type CanvasIssue struct {
	SceneName string `json:"sceneName"`
	NodeName  string `json:"nodeName"`
	Problem   string `json:"problem"`
}

// CheckCanvas 检测画布问题：
//  1. 空端口：任何端口必须有连线（出点有出边 / 入点有入边），不允许孤立点。
//  2. 双角色端口：既被连入又有出边的端口（树形模型不允许）。
//  3. 跳转资源缺失：出端口 targetKind 指向的目标节点缺少对应资源（mp4/html）。
func (s *NodeService) CheckCanvas(canvasID uint) ([]CanvasIssue, error) {
	var scenes []models.HtmlScene
	if err := s.db.Where("canvas_id = ?", canvasID).Find(&scenes).Error; err != nil {
		return nil, err
	}
	var nodes []models.Node
	sceneIDs := make([]uint, 0, len(scenes))
	for _, sc := range scenes {
		sceneIDs = append(sceneIDs, sc.ID)
	}
	if len(sceneIDs) > 0 {
		if err := s.db.Where("scene_id IN ?", sceneIDs).Find(&nodes).Error; err != nil {
			return nil, err
		}
	}
	var canvas models.Canvas
	if err := s.db.First(&canvas, canvasID).Error; err != nil {
		return nil, errors.New("画布不存在")
	}

	sceneName := map[uint]string{}
	for _, sc := range scenes {
		sceneName[sc.ID] = sc.Name
	}
	// 目标场景资源缓存
	resolve := func(sc models.HtmlScene, kind string) bool {
		dir := sceneDir(canvas.Path, sc.Name)
		if kind == "html" {
			_, err := os.Stat(sceneHTMLPath(canvas.Path, sc.Name))
			return err == nil
		}
		// mp4：scene.Video 或文件夹内任意 .mp4
		if sc.Video != "" {
			_, err := os.Stat(filepath.Join(dir, sc.Video))
			return err == nil
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return false
		}
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".mp4" {
				return true
			}
		}
		return false
	}

	issues := []CanvasIssue{}
	for _, n := range nodes {
		name := sceneName[n.SceneID]
		peer, hasPeer := s.pairNode(&n)
		if !hasPeer {
			issues = append(issues, CanvasIssue{
				SceneName: name, NodeName: n.Name, Problem: "空端口：没有配对的端口（哈希无另一端）",
			})
			continue
		}
		if n.Entry {
			continue // 入端口：无需资源检测
		}
		// 出端口：目标资源检测
		kind := n.TargetKind
		if kind == "" {
			kind = "mp4"
		}
		sc, ok := sceneByIDMap(scenes, peer.SceneID)
		if !ok {
			continue
		}
		if !resolve(sc, kind) {
			kindLabel := map[string]string{"mp4": "MP4 视频", "html": "HTML 页面"}[kind]
			issues = append(issues, CanvasIssue{
				SceneName: name, NodeName: n.Name,
				Problem: fmt.Sprintf("跳转目标「%s」文件夹里没有 %s", sc.Name, kindLabel),
			})
		}
	}
	return issues, nil
}

func sceneByIDMap(scenes []models.HtmlScene, id uint) (models.HtmlScene, bool) {
	for _, sc := range scenes {
		if sc.ID == id {
			return sc, true
		}
	}
	return models.HtmlScene{}, false
}
