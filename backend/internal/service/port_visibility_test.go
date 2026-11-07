package service

import "testing"

func TestScenePortSettingsBatchAndUndo(t *testing.T) {
	f := setup(t)
	child, ordinary := f.child(t, f.root.ID)
	magic, _, err := f.nodes.CreateMagicPort(f.root.ID, child.Hash)
	if err != nil {
		t.Fatal(err)
	}
	before := f.view(t)
	hidden := false
	ordinaryName, magicName := "  去学校  ", "返回起点"
	_, err = f.nodes.UpdateScene(f.root.ID, "renamed", PortSettings{ID: ordinary.ID, Name: &ordinaryName, IsShow: &hidden}, PortSettings{ID: magic.ID, Name: &magicName, IsShow: &hidden})
	if err != nil {
		t.Fatal(err)
	}
	after := f.view(t)
	for _, p := range after.Nodes {
		if p.ID == ordinary.ID && p.Name != "去学校" || p.ID == magic.ID && p.Name != magicName {
			t.Fatal("option name was not saved or trimmed")
		}
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
		for _, old := range before.Nodes {
			if p.ID == old.ID && p.Name != old.Name {
				t.Fatal("undo did not restore option names")
			}
		}
	}
}

func TestScenePortSettingsRejectsForeignOrInvalidPortsAtomically(t *testing.T) {
	f := setup(t)
	child, own := f.child(t, f.root.ID)
	_, foreign := f.child(t, child.ID)
	hidden := false
	blank := "  "
	for _, invalid := range []PortSettings{{ID: foreign.ID, IsShow: &hidden}, {ID: 999999, IsShow: &hidden}, {ID: own.ID}, {ID: own.ID, Name: &blank, IsShow: &hidden}} {
		before := f.view(t)
		if _, err := f.nodes.UpdateScene(f.root.ID, "must roll back", PortSettings{ID: own.ID, IsShow: &hidden}, invalid); err == nil {
			t.Fatal("invalid port update accepted")
		}
		if f.view(t).Token != before.Token {
			t.Fatal("failed batch partially changed graph")
		}
	}
	if err := f.tree.DeleteItem("canvas", f.canvas.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.nodes.UpdateScene(f.root.ID, "", PortSettings{ID: own.ID, IsShow: &hidden}); err == nil {
		t.Fatal("deleted project accepted port changes")
	}
}
