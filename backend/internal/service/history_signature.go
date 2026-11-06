package service

import (
	"encoding/json"
	"galgame-maker/internal/models"
	"time"
)

// Freeze the original signature layout independently of the live database/API.
// The old application always emitted zero coordinates and first=false for ports.
// Keeping those constants only in the signing payload preserves localStorage
// history and copied-property signatures without retaining obsolete model fields.
type historyPortV1 struct {
	ID            uint      `json:"id"`
	Hash          string    `json:"hash"`
	Name          string    `json:"name"`
	SceneID       uint      `json:"sceneId"`
	X             float64   `json:"x"`
	Y             float64   `json:"y"`
	First         bool      `json:"first"`
	TargetKind    string    `json:"targetKind"`
	EndingContent string    `json:"endingContent"`
	Entry         bool      `json:"entry"`
	Magic         bool      `json:"magic"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ExportHash    string    `json:"exportHash"`
	ShowOption    bool      `json:"showOption"`
	SortOrder     int       `json:"sortOrder"`
}

func marshalHistoryV1(view CanvasView) ([]byte, error) {
	var ports []historyPortV1
	if view.Nodes != nil {
		ports = make([]historyPortV1, 0, len(view.Nodes))
	}
	for _, p := range view.Nodes {
		ports = append(ports, historyPortV1{ID: p.ID, Hash: p.Hash, Name: p.Name, SceneID: p.SceneID,
			TargetKind: p.TargetKind, EndingContent: p.EndingContent, Entry: p.Entry, Magic: p.Magic,
			CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, ExportHash: p.ExportHash, ShowOption: p.ShowOption, SortOrder: p.SortOrder})
	}
	return json.Marshal(struct {
		CanvasID uint               `json:"canvasId"`
		Scenes   []models.HtmlScene `json:"scenes"`
		Nodes    []historyPortV1    `json:"nodes"`
		Token    string             `json:"token"`
	}{CanvasID: view.CanvasID, Scenes: view.Scenes, Nodes: ports})
}
