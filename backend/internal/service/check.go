package service

import (
	"galgame-maker/internal/models"
	"os"
)

type CanvasIssue struct {
	SceneName string `json:"sceneName"`
	NodeName  string `json:"nodeName"`
	Problem   string `json:"problem"`
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
	outs := map[uint][]models.Port{}
	first := 0
	for _, p := range pp {
		outs[p.NodeID] = append(outs[p.NodeID], p)
	}
	for _, sc := range ss {
		if sc.IsFirst {
			first++
		}
		if _, err := contentFiles(c, sc); err != nil {
			add(sc.Name, "素材", err.Error())
		}
		hidden := 0
		for _, p := range outs[sc.ID] {
			if !p.IsShow {
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
	return issues, nil
}
