package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"galgame-maker/internal/models"
	"testing"
)

func TestLegacyHistorySurvivesPortFieldCleanup(t *testing.T) {
	f := setup(t)
	child, p := f.child(t, f.root.ID)
	before := f.view(t)
	unsigned := *before
	unsigned.Token = ""
	data, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatal(err)
	}
	// Reproduce the old JSON writer, with its three retired fields immediately
	// before targetKind. The current public API must not include those fields.
	legacy := bytes.ReplaceAll(data, []byte("\"targetKind\":"), []byte("\"x\":0,\"y\":0,\"first\":false,\"targetKind\":"))
	var key models.Config
	if err := f.db.First(&key, "key = ?", "history.signing.key").Error; err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte(key.Value))
	mac.Write(legacy)
	oldToken := hex.EncodeToString(mac.Sum(nil))
	if oldToken != before.Token {
		t.Fatal("saved browser history would be invalidated")
	}
	var cached CanvasView
	if err := json.Unmarshal(legacy, &cached); err != nil {
		t.Fatal(err)
	}
	cached.Token = oldToken
	if _, err := f.nodes.UpdateNode(p.ID, "changed", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	current := f.view(t)
	restored, err := f.nodes.Restore(f.canvas.ID, current.Token, cached)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Nodes[0].Name != p.Name {
		t.Fatal("old history did not restore options")
	}
	if err := f.nodes.CopyProperties(child.ID, f.root.ID, cached); err != nil {
		t.Fatal("old copied-property snapshot rejected:", err)
	}
	raw, _ := json.Marshal(restored.Nodes[0])
	var port map[string]interface{}
	json.Unmarshal(raw, &port)
	for _, field := range []string{"x", "y", "first"} {
		if _, ok := port[field]; ok {
			t.Fatalf("obsolete API field %s survived", field)
		}
	}
}
