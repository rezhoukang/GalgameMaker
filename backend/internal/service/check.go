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
	var c models.Project
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
	first := 0
	for _, sc := range ss {
		if sc.IsFirst {
			first++
		}
		if _, err := contentFiles(c, sc); err != nil {
			add(sc.Name, "素材", err.Error())
		}
	}
	if first != 1 {
		add(c.Name, "起点", "必须恰好有一个播放起点")
	}
	return issues, nil
}
