package service

import (
	"encoding/json"
	"testing"
)

func TestProjectSchemaAndCoordinateFreeSnapshot(t *testing.T) {
	f := setup(t)
	m := f.db.Migrator()
	if !m.HasTable("projects") || m.HasTable("canvas") {
		t.Fatal("projects must be the only project table")
	}
	columns, err := m.ColumnTypes("nodes")
	if err != nil {
		t.Fatal(err)
	}
	foundProject := false
	for _, column := range columns {
		switch column.Name() {
		case "x", "y", "canvas_id":
			t.Fatalf("obsolete node column: %s", column.Name())
		case "project_id":
			foundProject = true
		}
	}
	if !foundProject {
		t.Fatal("node project reference missing")
	}
	if _, err := f.nodes.UpdateScene(f.root.ID, "renamed"); err != nil {
		t.Fatal(err)
	}
	view := f.view(t)
	data, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["projectId"]; !ok {
		t.Fatal("snapshot projectId missing")
	}
	var nodes []map[string]json.RawMessage
	if err := json.Unmarshal(payload["scenes"], &nodes); err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatal("root node missing")
	}
	for _, field := range []string{"x", "y", "canvasId"} {
		if _, ok := nodes[0][field]; ok {
			t.Fatalf("obsolete API field: %s", field)
		}
	}
	if string(nodes[0]["name"]) != `"renamed"` || string(nodes[0]["projectId"]) != string(payload["projectId"]) {
		t.Fatal("node rename or project reference incorrect")
	}
}
