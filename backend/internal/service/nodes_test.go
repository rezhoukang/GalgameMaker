package service

import (
	"encoding/json"
	"galgame-maker/internal/config"
	"galgame-maker/internal/database"
	"galgame-maker/internal/models"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixture struct {
	db     *gorm.DB
	nodes  *NodeService
	tree   *TreeService
	cfg    *config.Config
	canvas *models.Canvas
	root   *models.Node
}

func setup(t *testing.T) fixture {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{DBPath: filepath.Join(dir, "test.db"), StorageRoot: filepath.Join(dir, "assets"), OutputDir: filepath.Join(dir, "output")}
	db, err := database.Init(cfg)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	tree := NewTreeService(db, NewSettingsService(db, cfg))
	folder, err := tree.CreateFolder(nil, "folder")
	if err != nil {
		t.Fatal(err)
	}
	canvas, err := tree.CreateCanvas(folder.ID, "story")
	if err != nil {
		t.Fatal(err)
	}
	nodes := NewNodeService(db)
	root, err := nodes.CreateInitialScene(canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{db, nodes, tree, cfg, canvas, root}
}
func (f fixture) child(t *testing.T, id uint) (*models.Node, *models.Port) {
	t.Helper()
	s, p, _, err := f.nodes.CreateNextNode(id)
	if err != nil {
		t.Fatal(err)
	}
	return s, p
}
func (f fixture) view(t *testing.T) *CanvasView {
	t.Helper()
	v, err := f.tree.GetCanvasView(f.canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestOrdinaryTreeAndRootProtection(t *testing.T) {
	f := setup(t)
	a, pa := f.child(t, f.root.ID)
	b, _ := f.child(t, a.ID)
	v := f.view(t)
	if len(v.Scenes) != 3 || len(v.Nodes) != 2 {
		t.Fatal("ports must be stored once")
	}
	if v.Scenes[2].Level != 3 || v.Scenes[0].PortNum != 1 {
		t.Fatal("derived levels/counts incorrect")
	}
	if err := f.nodes.DeleteScene(f.root.ID); err == nil {
		t.Fatal("root deletion allowed")
	}
	if err := f.nodes.RedirectNode(pa.ID, b.Hash, "prev"); err == nil {
		t.Fatal("cycle allowed")
	}
	if f.view(t).Token != v.Token {
		t.Fatal("rejected operation changed graph")
	}
	if err := f.nodes.BreakConnection(pa.ID); err == nil {
		t.Fatal("detached ordinary subtree allowed")
	}
}
func TestMoveSubtreeRecalculatesDepth(t *testing.T) {
	f := setup(t)
	a, _ := f.child(t, f.root.ID)
	b, p := f.child(t, f.root.ID)
	c, _ := f.child(t, b.ID)
	if err := f.nodes.RedirectNode(p.ID, a.Hash, "prev"); err != nil {
		t.Fatal(err)
	}
	var restored models.Node
	f.db.First(&restored, c.ID)
	if restored.Level != 4 {
		t.Fatalf("level=%d", restored.Level)
	}
}
func TestMagicDeletionAndUndo(t *testing.T) {
	f := setup(t)
	a, p := f.child(t, f.root.ID)
	b, _ := f.child(t, a.ID)
	keep, _ := f.child(t, f.root.ID)
	if _, _, err := f.nodes.CreateMagicPort(b.ID, keep.Hash); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.nodes.CreateMagicPort(keep.ID, b.Hash); err != nil {
		t.Fatal(err)
	}
	before := f.view(t)
	if err := f.nodes.DeleteNode(p.ID); err != nil {
		t.Fatal(err)
	}
	after := f.view(t)
	if len(after.Scenes) != 2 || len(after.Nodes) != 1 {
		t.Fatal("magic edges expanded deletion or dangling edge survived")
	}
	if after.Scenes[1].ID != keep.ID {
		t.Fatal("magic target removed")
	}
	if _, err := f.nodes.Restore(f.canvas.ID, after.Token, *before); err != nil {
		t.Fatal(err)
	}
	if len(f.view(t).Scenes) != 4 || len(f.view(t).Nodes) != 5 {
		t.Fatal("undo did not restore graph")
	}
}
func TestHiddenPortRestoreAndSnapshotGuards(t *testing.T) {
	f := setup(t)
	_, p := f.child(t, f.root.ID)
	hidden := false
	if _, err := f.nodes.UpdateNode(p.ID, "", nil, nil, &hidden); err != nil {
		t.Fatal(err)
	}
	before := f.view(t)
	if _, err := f.nodes.UpdateNode(p.ID, "changed", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	after := f.view(t)
	if _, err := f.nodes.Restore(f.canvas.ID, before.Token, *before); err == nil {
		t.Fatal("stale expected token accepted")
	}
	encoded, _ := json.Marshal(before)
	var tampered CanvasView
	json.Unmarshal(encoded, &tampered)
	tampered.Scenes[0].AssetDir = "../../outside"
	if _, err := f.nodes.Restore(f.canvas.ID, after.Token, tampered); err == nil {
		t.Fatal("tampered snapshot accepted")
	}
	if _, err := f.nodes.Restore(f.canvas.ID, after.Token, *before); err != nil {
		t.Fatal(err)
	}
	if f.view(t).Nodes[0].ShowOption {
		t.Fatal("false was replaced by database default")
	}
}
func TestContentVersionsAndCopy(t *testing.T) {
	f := setup(t)
	target, _ := f.child(t, f.root.ID)
	old, err := f.nodes.SaveSceneVideo(f.root.ID, strings.NewReader("old video"))
	if err != nil {
		t.Fatal(err)
	}
	copied := f.view(t)
	newer, err := f.nodes.SaveSceneVideo(f.root.ID, strings.NewReader("new video"))
	if err != nil {
		t.Fatal(err)
	}
	if old.Video == newer.Video {
		t.Fatal("upload overwrote old content")
	}
	if err := f.nodes.CopyProperties(target.ID, f.root.ID, *copied); err != nil {
		t.Fatal(err)
	}
	var dst models.Node
	f.db.First(&dst, target.ID)
	if dst.Video != old.Video || dst.Hash != target.Hash || dst.IsRoot {
		t.Fatal("paste did not use captured properties or overwrote identity")
	}
	if err := f.nodes.DeleteScene(target.ID); err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(filepath.Join(assetDir(*f.canvas, *old), old.Video))
	if err != nil || string(bytes) != "old video" {
		t.Fatal("undo resources were destroyed")
	}
	current := f.view(t)
	if _, err := f.nodes.Restore(f.canvas.ID, current.Token, *copied); err != nil {
		t.Fatal(err)
	}
	var root models.Node
	f.db.First(&root, f.root.ID)
	if root.Video != old.Video {
		t.Fatal("undo failed to restore video")
	}
}
func TestCrossCanvasAndStartIsolation(t *testing.T) {
	f := setup(t)
	a, _ := f.child(t, f.root.ID)
	other, err := f.tree.CreateCanvas(f.canvas.FolderID, "other")
	if err != nil {
		t.Fatal(err)
	}
	root, err := f.nodes.CreateInitialScene(other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.nodes.CreateMagicPort(a.ID, root.Hash); err == nil {
		t.Fatal("cross-project magic connection allowed")
	}
	if _, err := f.nodes.SetFirstScene(a.ID); err != nil {
		t.Fatal(err)
	}
	var check models.Node
	f.db.First(&check, root.ID)
	if !check.IsFirst {
		t.Fatal("other project's start was cleared")
	}
}
func TestExportValidationAndRoutes(t *testing.T) {
	f := setup(t)
	a, p := f.child(t, f.root.ID)
	issues, err := f.nodes.CheckCanvas(f.canvas.ID)
	if err != nil || len(issues) == 0 {
		t.Fatal("empty content not rejected")
	}
	for _, id := range []uint{f.root.ID, a.ID} {
		if _, err := f.nodes.UpdateSceneHTML(id, []byte("<h1>content</h1>")); err != nil {
			t.Fatal(err)
		}
	}
	kind := "html"
	hidden := false
	if _, err := f.nodes.UpdateNode(p.ID, "", nil, &kind, &hidden); err != nil {
		t.Fatal(err)
	}
	exporter := NewExportService(f.db, NewSettingsService(f.db, f.cfg))
	result, err := exporter.Export(f.canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(result.OutputDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "#/node/") || !strings.Contains(string(page), `"showOption":false`) {
		t.Fatal("routes or hidden option absent")
	}
	_, q := f.child(t, f.root.ID)
	if _, err := f.nodes.UpdateNode(q.ID, "", nil, &kind, nil); err != nil {
		t.Fatal(err)
	}
	issues, err = f.nodes.CheckCanvas(f.canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, i := range issues {
		if strings.Contains(i.Problem, "自动继续") {
			found = true
		}
	}
	if !found {
		t.Fatal("ambiguous automatic exit allowed")
	}
}

func TestFolderRenameKeepsNestedResources(t *testing.T) {
	f := setup(t)
	parentID := f.canvas.FolderID
	folder, err := f.tree.CreateFolder(&parentID, "nested")
	if err != nil {
		t.Fatal(err)
	}
	canvas, err := f.tree.CreateCanvas(folder.ID, "nested-story")
	if err != nil {
		t.Fatal(err)
	}
	root, err := f.nodes.CreateInitialScene(canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	root, err = f.nodes.SaveSceneVideo(root.ID, strings.NewReader("video"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.tree.RenameItem("folder", parentID, "renamed"); err != nil {
		t.Fatal(err)
	}
	var moved models.Folder
	f.db.First(&moved, folder.ID)
	var movedCanvas models.Canvas
	f.db.First(&movedCanvas, canvas.ID)
	if !strings.Contains(moved.Path, "renamed") || !hasFile(filepath.Join(assetDir(movedCanvas, *root), root.Video)) {
		t.Fatal("nested paths not updated")
	}
	if _, err := f.tree.CreateFolder(&parentID, "nested"); err == nil {
		t.Fatal("duplicate folder allowed")
	}
}
