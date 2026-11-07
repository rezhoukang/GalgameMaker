package service

import (
	"encoding/json"
	"galgame-maker/internal/models"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestVariableRulesOwnershipAndAtomicSave(t *testing.T) {
	f := setup(t)
	_, p := f.child(t, f.root.ID)
	if err := f.nodes.SaveVariables(f.canvas.ID, []models.Variable{{Name: "勇气", InitialValue: 2}, {Name: "记忆", ResetOnLoop: true}}); err != nil {
		t.Fatal(err)
	}
	vars := f.view(t).Variables
	rules := PortRules{ConditionMode: "all", Conditions: []models.PortCondition{{VariableID: vars[0].ID, Operator: ">=", Value: 2}}, Effects: []models.PortEffect{{VariableID: vars[0].ID, Delta: -1}, {VariableID: vars[1].ID, Delta: 3}}}
	hidden := false
	if _, err := f.nodes.UpdateNode(p.ID, "前进", &hidden, rules); err != nil {
		t.Fatal(err)
	}
	before := f.view(t)
	invalid := []PortRules{
		{ConditionMode: "invalid"},
		{ConditionMode: "all", Conditions: []models.PortCondition{{VariableID: vars[0].ID, Operator: "=", Value: 1}}},
		{ConditionMode: "all", Effects: []models.PortEffect{{VariableID: vars[0].ID, Delta: 1}, {VariableID: vars[0].ID, Delta: 2}}},
		{ConditionMode: "all", Effects: []models.PortEffect{{VariableID: vars[0].ID, Delta: maxStoryValue + 1}}},
	}
	other, err := f.tree.CreateCanvas(f.canvas.FolderID, "other")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.nodes.SaveVariables(other.ID, []models.Variable{{Name: "外部"}}); err != nil {
		t.Fatal(err)
	}
	foreign, err := projectVariables(f.db, other.ID)
	if err != nil {
		t.Fatal(err)
	}
	invalid = append(invalid, PortRules{ConditionMode: "all", Effects: []models.PortEffect{{VariableID: foreign[0].ID, Delta: 1}}})
	for _, r := range invalid {
		if _, err := f.nodes.UpdateNode(p.ID, "不应保存", nil, r); err == nil {
			t.Fatal("accepted invalid rules")
		}
		if f.view(t).Token != before.Token {
			t.Fatal("invalid update was not atomic")
		}
	}
	for _, bad := range [][]models.Variable{
		{}, {{Name: "重复"}, {Name: "重复"}}, {{Name: "越界", InitialValue: maxStoryValue + 1}}, {foreign[0]},
	} {
		if err := f.nodes.SaveVariables(f.canvas.ID, bad); err == nil {
			t.Fatal("accepted invalid definitions or removed referenced variables")
		}
		if f.view(t).Token != before.Token {
			t.Fatal("definitions changed on failure")
		}
	}
	vars[0].Name, vars[1].Name = vars[1].Name, vars[0].Name
	if err := f.nodes.SaveVariables(f.canvas.ID, vars); err != nil {
		t.Fatal(err)
	}
	if _, err := f.nodes.UpdateNode(p.ID, "", nil, PortRules{ConditionMode: "all"}); err != nil {
		t.Fatal(err)
	}
	if err := f.nodes.SaveVariables(f.canvas.ID, nil); err != nil {
		t.Fatal(err)
	}
}

func TestVariableRulesUndoAndDeletion(t *testing.T) {
	f := setup(t)
	child, p := f.child(t, f.root.ID)
	magic, _, err := f.nodes.CreateMagicPort(f.root.ID, child.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.nodes.SaveVariables(f.canvas.ID, []models.Variable{{Name: "轮次", ResetOnLoop: false}}); err != nil {
		t.Fatal(err)
	}
	variable := f.view(t).Variables[0]
	rules := PortRules{ConditionMode: "any", ResetLoop: true, Conditions: []models.PortCondition{{VariableID: variable.ID, Operator: "==", Value: 0}}, Effects: []models.PortEffect{{VariableID: variable.ID, Delta: 1}}}
	hidden := false
	for _, id := range []uint{p.ID, magic.ID} {
		if _, err = f.nodes.UpdateNode(id, "loop", &hidden, rules); err != nil {
			t.Fatal(err)
		}
	}
	before := f.view(t)
	if err = f.nodes.DeleteScene(child.ID); err != nil {
		t.Fatal(err)
	}
	for _, model := range []interface{}{&models.PortCondition{}, &models.PortEffect{}} {
		var count int64
		if err = f.db.Model(model).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("orphan rules %d: %v", count, err)
		}
	}
	current := f.view(t)
	restored, err := f.nodes.Restore(f.canvas.ID, current.Token, *before)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.Variables, before.Variables) || len(restored.Nodes) != len(before.Nodes) {
		t.Fatal("undo changed variables or ports")
	}
	for i, p := range restored.Nodes {
		old := before.Nodes[i]
		if p.IsShow || p.ConditionMode != old.ConditionMode || p.ResetLoop != old.ResetLoop || !reflect.DeepEqual(p.Conditions, old.Conditions) || !reflect.DeepEqual(p.Effects, old.Effects) {
			t.Fatal("undo changed rules")
		}
	}
	if err = f.nodes.BreakConnection(magic.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	f.db.Model(&models.PortEffect{}).Where("port_id = ?", magic.ID).Count(&count)
	if count != 0 {
		t.Fatal("magic rules survived deletion")
	}
	if err = f.db.Delete(f.canvas).Error; err != nil {
		t.Fatal(err)
	}
	if err = f.nodes.SaveVariables(f.canvas.ID, nil); err == nil {
		t.Fatal("edited deleted project")
	}
	if _, err = f.nodes.UpdateNode(p.ID, "", nil, rules); err == nil {
		t.Fatal("edited deleted project port")
	}
}

func TestPlayerExportIncludesVariableRules(t *testing.T) {
	f := setup(t)
	vars := []models.Variable{{ID: 1, ProjectID: f.canvas.ID, Name: "勇气</script>", InitialValue: 2, ResetOnLoop: true}}
	scenes := map[uint]*SceneRes{1: {Hash: "root", HTML: "a.html", Options: []LinkInfo{{ID: 9, TargetID: 1, ConditionMode: "any", ResetLoop: true, Conditions: []models.PortCondition{{VariableID: 1, Operator: ">=", Value: 2}}, Effects: []models.PortEffect{{VariableID: 1, Delta: -1}}}}}}
	dir := t.TempDir()
	exporter := NewExportService(f.db, NewSettingsService(f.db, f.cfg))
	if err := exporter.writePlayerIndex(dir, 1, scenes, vars, "project"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(data)
	for _, placeholder := range []string{"__STORY_", "__FIRST_NODE__"} {
		if strings.Contains(page, placeholder) {
			t.Fatal("unresolved template", placeholder)
		}
	}
	start := strings.Index(page, `id="story-meta">`) + len(`id="story-meta">`)
	end := strings.Index(page[start:], "</script>") + start
	var meta struct {
		Variables            []models.Variable
		ProjectKey, Revision string
	}
	if err = json.Unmarshal([]byte(page[start:end]), &meta); err != nil {
		t.Fatal(err)
	}
	if len(meta.Variables) != 1 || meta.Variables[0].Name != vars[0].Name || meta.ProjectKey != "project" || meta.Revision == "" {
		t.Fatal("invalid metadata")
	}
	if !strings.Contains(page, `"delta":-1`) || !strings.Contains(page, `"resetLoop":true`) {
		t.Fatal("missing rules")
	}
}
