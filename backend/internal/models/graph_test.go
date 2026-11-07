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
				{ID: 1, CanvasID: 1, Hash: "root", Name: "root", Level: 1},
				{ID: 2, CanvasID: 1, Hash: "child", Name: "child"},
			}
			ports := []Port{
				{ID: 1, Hash: "ordinary", NodeID: 1, ExportHash: "child", Name: "ordinary"},
				{ID: 2, Hash: "extra", NodeID: 1, ExportHash: tc.target, Name: "extra", Magic: tc.magic},
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

func TestRootIdentifiedByLevel(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		rootLevel, childLevel int
		wantErr               bool
	}{
		{"unique root", 1, 2, false},
		{"recalculate child depth", 1, 9, false},
		{"missing root", 2, 3, true},
		{"multiple roots", 1, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes := []Node{
				{ID: 1, CanvasID: 1, Hash: "root", Name: "root", Level: tc.rootLevel},
				{ID: 2, CanvasID: 1, Hash: "child", Name: "child", Level: tc.childLevel},
			}
			ports := []Port{{ID: 1, Hash: "edge", NodeID: 1, ExportHash: "child", Name: "choice"}}
			err := ValidateGraph(nodes, ports)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateGraph error = %v, want error = %v", err, tc.wantErr)
			}
			if err == nil && (nodes[0].Level != 1 || nodes[1].Level != 2) {
				t.Fatal("ordinary levels were not recalculated")
			}
		})
	}
}
