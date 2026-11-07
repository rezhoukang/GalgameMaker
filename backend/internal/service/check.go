package service

import (
	"fmt"
	"galgame-maker/internal/models"
	"os"
	"path/filepath"
	"strings"
)

type CanvasIssue struct {
	SceneName string `json:"sceneName"`
	NodeName  string `json:"nodeName"`
	Problem   string `json:"problem"`
}

func videoFiles(c models.Canvas, sc models.Node) []string {
	if sc.Video != "" {
		return []string{sc.Video}
	}
	files := []string{}
	entries, _ := os.ReadDir(assetDir(c, sc))
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".mp4") {
			files = append(files, e.Name())
		}
	}
	return files
}
func hasFile(path string) bool {
	if path == "" {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}
func (s *NodeService) CheckCanvas(canvasID uint) ([]CanvasIssue, error) {
	var c models.Canvas
	if err := s.db.First(&c, canvasID).Error; err != nil {
		return nil, err
	}
	ss, pp, err := graph(s.db, canvasID)
	if err != nil {
		return nil, err
	}
	issues := []CanvasIssue{}
	add := func(sc, name, msg string) { issues = append(issues, CanvasIssue{sc, name, msg}) }
	if err := models.ValidateGraph(ss, pp); err != nil {
		add(c.Name, "结构", err.Error())
		return issues, nil
	}
	byHash := map[string]models.Node{}
	outs := map[uint][]models.Port{}
	first := 0
	for _, p := range pp {
		outs[p.SceneID] = append(outs[p.SceneID], p)
	}
	for _, sc := range ss {
		byHash[sc.Hash] = sc
		if sc.IsFirst {
			first++
		}
		videos := videoFiles(c, sc)
		hasVideo := len(videos) > 0 && hasFile(filepath.Join(assetDir(c, sc), videos[0]))
		if !hasVideo && !hasFile(htmlPath(c, sc)) {
			add(sc.Name, "素材", "请导入视频或 HTML 页面后导出")
		}
		hidden := 0
		for _, p := range outs[sc.ID] {
			if !p.ShowOption {
				hidden++
			}
		}
		if hidden > 0 && len(outs[sc.ID]) != 1 {
			add(sc.Name, "选项", "自动继续要求恰好一个隐藏出口，不能与其他出口混用")
		}
	}
	if first != 1 {
		add(c.Name, "起点", "必须恰好有一个播放起点")
	}
	for _, p := range pp {
		target := byHash[p.ExportHash]
		available := hasFile(htmlPath(c, target))
		label := "HTML 页面"
		if p.TargetKind == "mp4" {
			label = "视频"
			videos := videoFiles(c, target)
			available = len(videos) > 0 && hasFile(filepath.Join(assetDir(c, target), videos[0]))
		}
		if !available {
			add(target.Name, p.Name, fmt.Sprintf("跳转目标缺少%s", label))
		}
	}
	return issues, nil
}
