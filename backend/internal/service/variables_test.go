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
	if err := f.nodes.SaveVariables(f.canvas.ID, []models.Variable{{Name: "勇气", InitialValue: 2}, {Name: "记忆"}}); err != nil {
		t.Fatal(err)
	}
	vars := f.view(t).Variables
	rules := PortRules{ConditionMode: "all", Conditions: []models.PortCondition{{VariableID: vars[0].ID, Operator: ">=", Value: 2}}, Effects: []models.PortEffect{{VariableID: vars[0].ID, Set: 0, Delta: -1}, {VariableID: vars[1].ID, Set: 0, Delta: 3}}}
	hidden := false
	if _, err := f.nodes.UpdateNode(p.ID, "前进", &hidden, rules); err != nil {
		t.Fatal(err)
	}
	before := f.view(t)
	invalid := []PortRules{
		{ConditionMode: "invalid"},
		{ConditionMode: "all", Effects: []models.PortEffect{{VariableID: vars[0].ID, Set: 2, Delta: 0}}},
		{ConditionMode: "all", Effects: []models.PortEffect{{VariableID: vars[0].ID, Set: -1, Delta: 1}}},
		{ConditionMode: "all", Conditions: []models.PortCondition{{VariableID: vars[0].ID, Operator: "=", Value: 1}}},
		{ConditionMode: "all", Effects: []models.PortEffect{{VariableID: vars[0].ID, Set: 0, Delta: 1}, {VariableID: vars[0].ID, Set: 0, Delta: 2}}},
		{ConditionMode: "all", Effects: []models.PortEffect{{VariableID: vars[0].ID, Set: 0, Delta: maxStoryValue + 1}}},
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
	invalid = append(invalid, PortRules{ConditionMode: "all", Effects: []models.PortEffect{{VariableID: foreign[0].ID, Set: 0, Delta: 1}}})
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
	if err = f.nodes.SaveVariables(f.canvas.ID, []models.Variable{{Name: "轮次"}}); err != nil {
		t.Fatal(err)
	}
	variable := f.view(t).Variables[0]
	rules := PortRules{ConditionMode: "any", Conditions: []models.PortCondition{{VariableID: variable.ID, Operator: "==", Value: 0}}, Effects: []models.PortEffect{{VariableID: variable.ID, Set: 1, Delta: 0}}}
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
	current := f.view(t)
	if len(current.Nodes) != 0 {
		t.Fatal("deleted subtree still has ports and rules")
	}
	restored, err := f.nodes.Restore(f.canvas.ID, current.Token, *before)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.Variables, before.Variables) || len(restored.Nodes) != len(before.Nodes) {
		t.Fatal("undo changed variables or ports")
	}
	for i, p := range restored.Nodes {
		old := before.Nodes[i]
		if p.IsShow || p.ConditionMode != old.ConditionMode || !reflect.DeepEqual(p.Conditions, old.Conditions) || !reflect.DeepEqual(p.Effects, old.Effects) {
			t.Fatal("undo changed rules")
		}
	}
	if err = f.nodes.BreakConnection(magic.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err = f.db.Model(&models.Port{}).Where("id = ?", magic.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
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

func TestPortJSONConditionsAndEffects(t *testing.T) {
	f := setup(t)
	_, p := f.child(t, f.root.ID)
	if f.db.Migrator().HasColumn(&models.Port{}, "reset_loop") || f.db.Migrator().HasColumn(&models.Variable{}, "reset_on_loop") {
		t.Fatal("obsolete loop reset fields still exist")
	}
	if f.db.Migrator().HasTable("port_conditions") || f.db.Migrator().HasTable("port_effects") || !f.db.Migrator().HasColumn(&models.Port{}, "conditions") || !f.db.Migrator().HasColumn(&models.Port{}, "effects") {
		t.Fatal("incorrect condition/effect storage schema")
	}
	read := func() (string, string) {
		t.Helper()
		var raw struct{ Conditions, Effects string }
		if err := f.db.Table("ports").Select("conditions", "effects").Where("id = ?", p.ID).Scan(&raw).Error; err != nil {
			t.Fatal(err)
		}
		return raw.Conditions, raw.Effects
	}
	if c, e := read(); c != "[]" || e != "[]" {
		t.Fatalf("empty rules must be arrays: %q %q", c, e)
	}
	if err := f.nodes.SaveVariables(f.canvas.ID, []models.Variable{{Name: "值"}}); err != nil {
		t.Fatal(err)
	}
	id := f.view(t).Variables[0].ID
	rules := PortRules{ConditionMode: "all", Conditions: []models.PortCondition{{VariableID: id, Operator: ">=", Value: 0}}, Effects: []models.PortEffect{{VariableID: id, Set: 1, Delta: 2}}}
	if _, err := f.nodes.UpdateNode(p.ID, "选项", nil, rules); err != nil {
		t.Fatal(err)
	}
	c, e := read()
	var conditions []models.PortCondition
	var effects []models.PortEffect
	if err := json.Unmarshal([]byte(c), &conditions); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(e), &effects); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(conditions, rules.Conditions) || !reflect.DeepEqual(effects, rules.Effects) {
		t.Fatal("stored conditions or effects differ from rules")
	}
	// 从数据库读取 JSON 规则后，完整导出仍携带同样的条件和效果。
	for _, scene := range f.view(t).Scenes {
		if _, err := f.nodes.UpdateSceneHTML(scene.ID, []byte("<html><body>test</body></html>")); err != nil {
			t.Fatal(err)
		}
	}
	exported, err := NewExportService(f.db, NewSettingsService(f.db, f.cfg)).Export(f.canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(exported.OutputDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `"conditions":`+c) || !strings.Contains(string(page), `"effects":`+e) {
		t.Fatal("export lost persisted port rules")
	}
	hidden := false
	name := "改名"
	if _, err := f.nodes.UpdateScene(f.root.ID, "", PortSettings{ID: p.ID, Name: &name, IsShow: &hidden}); err != nil {
		t.Fatal(err)
	}
	if c2, e2 := read(); c2 != c || e2 != e {
		t.Fatal("editing port name/visibility changed rules")
	}
	rules.Effects = nil
	if _, err := f.nodes.UpdateNode(p.ID, "", nil, rules); err != nil {
		t.Fatal(err)
	}
	if err := f.nodes.SaveVariables(f.canvas.ID, nil); err == nil {
		t.Fatal("condition-only variable reference was ignored")
	}
	if _, err := f.nodes.UpdateNode(p.ID, "", nil, PortRules{ConditionMode: "all"}); err != nil {
		t.Fatal(err)
	}
	if c, e := read(); c != "[]" || e != "[]" {
		t.Fatal("cleared rules were not saved as empty arrays")
	}
}

func TestPlayerExportIncludesVariableRules(t *testing.T) {
	f := setup(t)
	vars := []models.Variable{{ID: 1, ProjectID: f.canvas.ID, Name: "勇气</script>", InitialValue: 2}}
	scenes := map[uint]*SceneRes{1: {Hash: "root", HTML: "a.html", Options: []LinkInfo{{ID: 9, TargetID: 1, ConditionMode: "any", Conditions: []models.PortCondition{{VariableID: 1, Operator: ">=", Value: 2}}, Effects: []models.PortEffect{{VariableID: 1, Set: 0, Delta: -1}}}}}}
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
	if !strings.Contains(page, `"delta":-1`) || !strings.Contains(page, `"set":0`) {
		t.Fatal("missing rules")
	}
}
