package models

import "testing"

func TestIncomingValidationWithoutStoredCounts(t *testing.T) {
	for _, tc := range []struct {
		name    string
		magic   bool
		target  string
		wantErr bool
	}{
		{"multiple magic inputs", true, "child", false},
		{"magic return to root", true, "root", false},
		{"multiple ordinary inputs", false, "child", true},
		{"ordinary input to root", false, "root", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes := []Node{
				{ID: 1, CanvasID: 1, Hash: "root", Name: "root", IsRoot: true},
				{ID: 2, CanvasID: 1, Hash: "child", Name: "child"},
			}
			ports := []Port{
				{ID: 1, Hash: "ordinary", SceneID: 1, ExportHash: "child", Name: "ordinary", TargetKind: "mp4"},
				{ID: 2, Hash: "extra", SceneID: 1, ExportHash: tc.target, Name: "extra", TargetKind: "mp4", Magic: tc.magic},
			}
			err := ValidateGraph(nodes, ports)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateGraph error = %v, want error = %v", err, tc.wantErr)
			}
			if err == nil && (nodes[0].Level != 1 || nodes[1].Level != 2) {
				t.Fatal("magic ports affected ordinary levels")
			}
		})
	}
}
