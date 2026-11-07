package service

import "testing"

func TestScenePortVisibilityBatchAndUndo(t *testing.T) {
	f := setup(t)
	child, ordinary := f.child(t, f.root.ID)
	magic, _, err := f.nodes.CreateMagicPort(f.root.ID, child.Hash)
	if err != nil {
		t.Fatal(err)
	}
	before := f.view(t)
	hidden := false
	_, err = f.nodes.UpdateScene(f.root.ID, "renamed", PortVisibility{ordinary.ID, &hidden}, PortVisibility{magic.ID, &hidden})
	if err != nil {
		t.Fatal(err)
	}
	after := f.view(t)
	for _, p := range after.Nodes {
		if p.IsShow {
			t.Fatal("batch did not hide all outgoing options")
		}
	}
	if after.Scenes[0].Name != "renamed" {
		t.Fatal("node name not saved with batch")
	}
	if _, err := f.nodes.Restore(f.canvas.ID, after.Token, *before); err != nil {
		t.Fatal(err)
	}
	for _, p := range f.view(t).Nodes {
		if !p.IsShow {
			t.Fatal("undo did not restore options")
		}
	}
}

func TestScenePortVisibilityRejectsForeignOrInvalidPortsAtomically(t *testing.T) {
	f := setup(t)
	child, own := f.child(t, f.root.ID)
	_, foreign := f.child(t, child.ID)
	hidden := false
	for _, invalid := range []PortVisibility{{foreign.ID, &hidden}, {999999, &hidden}, {own.ID, nil}} {
		before := f.view(t)
		if _, err := f.nodes.UpdateScene(f.root.ID, "must roll back", PortVisibility{own.ID, &hidden}, invalid); err == nil {
			t.Fatal("invalid port update accepted")
		}
		if f.view(t).Token != before.Token {
			t.Fatal("failed batch partially changed graph")
		}
	}
	if err := f.tree.DeleteItem("canvas", f.canvas.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.nodes.UpdateScene(f.root.ID, "", PortVisibility{own.ID, &hidden}); err == nil {
		t.Fatal("deleted project accepted port changes")
	}
}
