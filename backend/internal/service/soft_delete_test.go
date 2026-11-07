package service

import (
	"galgame-maker/internal/models"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeletedCanvasRetainsDataAndRejectsAccess(t *testing.T) {
	f := setup(t)
	child, port := f.child(t, f.root.ID)
	magic, _, err := f.nodes.CreateMagicPort(child.ID, f.root.Hash)
	if err != nil {
		t.Fatal(err)
	}
	media, err := f.nodes.SaveSceneVideo(child.ID, strings.NewReader("retained video"))
	if err != nil {
		t.Fatal(err)
	}
	before := f.view(t)
	if err := f.tree.DeleteItem("canvas", f.canvas.ID); err != nil {
		t.Fatal(err)
	}
	var stored models.Canvas
	if err := f.db.Unscoped().First(&stored, f.canvas.ID).Error; err != nil || !stored.DeleteAt.Valid {
		t.Fatal("project was not soft deleted", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(contentPath(stored, *media)), "video.mp4")); err != nil {
		t.Fatal("media lost", err)
	}
	tree, err := f.tree.GetTree()
	if err != nil || len(tree) != 1 || len(tree[0].Canvases) != 0 {
		t.Fatal("deleted project visible", err)
	}
	checks := map[string]func() error{
		"read":  func() error { _, e := f.tree.GetCanvasView(f.canvas.ID); return e },
		"check": func() error { _, e := f.nodes.CheckCanvas(f.canvas.ID); return e },
		"export": func() error {
			_, e := NewExportService(f.db, NewSettingsService(f.db, f.cfg)).Export(f.canvas.ID)
			return e
		},
		"initial":      func() error { _, e := f.nodes.CreateInitialScene(f.canvas.ID); return e },
		"scene update": func() error { _, e := f.nodes.UpdateScene(child.ID, "changed", 0, 0); return e },
		"scene delete": func() error { return f.nodes.DeleteScene(child.ID) },
		"port update":  func() error { _, e := f.nodes.UpdateNode(port.ID, "changed", nil); return e },
		"port delete":  func() error { return f.nodes.DeleteNode(port.ID) },
		"magic delete": func() error { return f.nodes.BreakConnection(magic.ID) },
		"redirect":     func() error { return f.nodes.RedirectNode(magic.ID, child.Hash, "next") },
		"start":        func() error { _, e := f.nodes.SetFirstScene(child.ID); return e },
		"next":         func() error { _, _, _, e := f.nodes.CreateNextNode(child.ID); return e },
		"magic":        func() error { _, _, e := f.nodes.CreateMagicPort(child.ID, f.root.Hash); return e },
		"video":        func() error { _, e := f.nodes.SaveSceneVideo(child.ID, strings.NewReader("new")); return e },
		"html":         func() error { _, e := f.nodes.UpdateSceneHTML(child.ID, []byte("new")); return e },
		"copy":         func() error { return f.nodes.CopyProperties(child.ID, f.root.ID, *before) },
		"undo":         func() error { _, e := f.nodes.Restore(f.canvas.ID, before.Token, *before); return e },
		"rename":       func() error { return f.tree.RenameItem("canvas", f.canvas.ID, "changed") },
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			if err := check(); err == nil {
				t.Fatal("deleted project was accessible")
			}
		})
	}
	var scenes []models.Node
	var ports []models.Port
	f.db.Where("canvas_id = ?", f.canvas.ID).Find(&scenes)
	f.db.Order("sort_order, id").Find(&ports)
	if len(scenes) != len(before.Scenes) || len(ports) != len(before.Nodes) {
		t.Fatal("retained graph changed")
	}
	for i, p := range ports {
		if p.Name != before.Nodes[i].Name || p.ExportHash != before.Nodes[i].ExportHash {
			t.Fatal("retained port changed")
		}
	}
	next, err := f.tree.CreateCanvas(f.canvas.FolderID, f.canvas.Name)
	if err != nil || next.Path == f.canvas.Path {
		t.Fatal("retained path reused", err)
	}
	if err := f.tree.RenameItem("canvas", next.ID, f.canvas.Name); err == nil {
		t.Fatal("rename overwrote retained path")
	}
	oldTime := stored.DeleteAt.Time
	_ = f.tree.DeleteItem("canvas", f.canvas.ID)
	f.db.Unscoped().First(&stored, f.canvas.ID)
	if !oldTime.Equal(stored.DeleteAt.Time) {
		t.Fatal("repeated deletion changed timestamp")
	}
}

func TestFolderSoftDeleteCascadeAndRetainedPaths(t *testing.T) {
	f := setup(t)
	parentID := f.canvas.FolderID
	nested, err := f.tree.CreateFolder(&parentID, "nested")
	if err != nil {
		t.Fatal(err)
	}
	canvas, err := f.tree.CreateCanvas(nested.ID, "nested story")
	if err != nil {
		t.Fatal(err)
	}
	root, err := f.nodes.CreateInitialScene(canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	media, err := f.nodes.SaveSceneVideo(root.ID, strings.NewReader("retained"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.tree.DeleteItem("canvas", canvas.ID); err != nil {
		t.Fatal(err)
	}
	var earlier models.Canvas
	f.db.Unscoped().First(&earlier, canvas.ID)
	if err := f.tree.DeleteItem("folder", nested.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.tree.RenameItem("folder", parentID, "renamed"); err != nil {
		t.Fatal(err)
	}
	var moved models.Canvas
	f.db.Unscoped().First(&moved, canvas.ID)
	if moved.Path == canvas.Path {
		t.Fatal("deleted child path not updated")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(contentPath(moved, *media)), "video.mp4")); err != nil {
		t.Fatal(err)
	}
	if err := f.tree.DeleteItem("folder", parentID); err != nil {
		t.Fatal(err)
	}
	var folders []models.Folder
	f.db.Unscoped().Find(&folders)
	var parent models.Folder
	for _, folder := range folders {
		if !folder.DeleteAt.Valid {
			t.Fatal("descendant not deleted")
		}
		if folder.ID == parentID {
			parent = folder
		}
	}
	var direct models.Canvas
	f.db.Unscoped().First(&direct, f.canvas.ID)
	if !direct.DeleteAt.Valid || !direct.DeleteAt.Time.Equal(parent.DeleteAt.Time) {
		t.Fatal("cascade timestamp mismatch")
	}
	f.db.Unscoped().First(&moved, canvas.ID)
	if !earlier.DeleteAt.Time.Equal(moved.DeleteAt.Time) {
		t.Fatal("earlier deletion overwritten")
	}
	tree, err := f.tree.GetTree()
	if err != nil || len(tree) != 0 {
		t.Fatal("deleted folder visible", err)
	}
	if _, err := f.tree.CreateFolder(&parentID, "blocked"); err == nil {
		t.Fatal("created in deleted parent")
	}
	if _, err := f.tree.CreateCanvas(nested.ID, "blocked"); err == nil {
		t.Fatal("created in deleted folder")
	}
	if _, err := f.tree.GetCanvasView(f.canvas.ID); err == nil {
		t.Fatal("descendant still accessible")
	}
	if initial, err := f.tree.EnsureDefaults(); err != nil || initial != nil {
		t.Fatal("restart recreated deleted defaults", err)
	}
	if err := f.tree.ResetAll(); err != nil {
		t.Fatal(err)
	}
	var count int64
	f.db.Unscoped().Model(&models.Folder{}).Where("delete_at IS NOT NULL").Count(&count)
	if count != 0 {
		t.Fatal("explicit reset retained deleted folders")
	}
	if _, err := os.Stat(moved.Path); !os.IsNotExist(err) {
		t.Fatal("explicit reset retained old media")
	}
}

func TestFolderSoftDeleteRollsBack(t *testing.T) {
	f := setup(t)
	if err := f.db.Exec("CREATE TRIGGER reject_folder_delete BEFORE UPDATE OF delete_at ON folders BEGIN SELECT RAISE(ABORT, 'test rollback'); END").Error; err != nil {
		t.Fatal(err)
	}
	if err := f.tree.DeleteItem("folder", f.canvas.FolderID); err == nil {
		t.Fatal("expected rollback")
	}
	if _, err := f.tree.GetCanvasView(f.canvas.ID); err != nil {
		t.Fatal("project deletion partially committed", err)
	}
	var folder models.Folder
	if err := f.db.First(&folder, f.canvas.FolderID).Error; err != nil {
		t.Fatal("folder deletion partially committed", err)
	}
}
