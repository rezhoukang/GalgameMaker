package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTMLContentReplacementUndoAndExport(t *testing.T) {
	f := setup(t)
	child, _ := f.child(t, f.root.ID)
	video, err := f.nodes.SaveSceneVideo(f.root.ID, strings.NewReader("video bytes"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(htmlPath(*f.canvas, *video))
	if err != nil || !strings.Contains(string(data), `src="video.mp4"`) {
		t.Fatal("video HTML was not created", err)
	}
	before := f.view(t)
	if err := f.nodes.CopyProperties(child.ID, f.root.ID, *before); err != nil {
		t.Fatal(err)
	}
	exporter := NewExportService(f.db, NewSettingsService(f.db, f.cfg))
	shared, err := exporter.Export(f.canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	if shared.FileCount != 3 {
		t.Fatalf("shared version should be exported once: %v", shared.Files)
	}
	page, err := f.nodes.UpdateSceneHTML(f.root.ID, []byte("<html><body><h1>New page</h1></body></html>"))
	if err != nil {
		t.Fatal(err)
	}
	if page.HTMLPath == video.HTMLPath {
		t.Fatal("content replacement reused immutable path")
	}
	current := f.view(t)
	if err := f.nodes.CopyProperties(child.ID, f.root.ID, *current); err != nil {
		t.Fatal(err)
	}
	result, err := exporter.Export(f.canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.FileCount != 2 {
		t.Fatalf("old video leaked into current export: %v", result.Files)
	}
	for _, file := range result.Files {
		if !hasFile(filepath.Join(result.OutputDir, file)) {
			t.Fatalf("missing export file: %s", file)
		}
	}
	pageData, err := os.ReadFile(filepath.Join(result.OutputDir, "dist", page.HTMLPath))
	if err != nil || !strings.Contains(string(pageData), "galgame-content") || !strings.Contains(string(pageData), "New page") {
		t.Fatal("HTML or playback bridge missing", err)
	}
	current = f.view(t)
	if _, err := f.nodes.Restore(f.canvas.ID, current.Token, *before); err != nil {
		t.Fatal(err)
	}
	if got := f.view(t).Scenes[0].HTMLPath; got != video.HTMLPath {
		t.Fatal("undo did not restore video page")
	}
	if !hasFile(filepath.Join(filepath.Dir(htmlPath(*f.canvas, *video)), "video.mp4")) {
		t.Fatal("video resource lost after replacement and undo")
	}
}

func TestContentValidationMissingVideoAndUnsafePath(t *testing.T) {
	f := setup(t)
	video, err := f.nodes.SaveSceneVideo(f.root.ID, strings.NewReader("video bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(filepath.Dir(htmlPath(*f.canvas, *video)), "video.mp4")); err != nil {
		t.Fatal(err)
	}
	issues, err := f.nodes.CheckCanvas(f.canvas.ID)
	if err != nil || len(issues) != 1 || !strings.Contains(issues[0].Problem, "视频文件不存在") {
		t.Fatal("missing companion video accepted", issues, err)
	}
	for _, path := range []string{"../index.html", "index.html", ".content/../index.html"} {
		video.HTMLPath = path
		if _, err := contentFiles(*f.canvas, *video); err == nil {
			t.Fatalf("unsafe export path accepted: %s", path)
		}
	}
}
