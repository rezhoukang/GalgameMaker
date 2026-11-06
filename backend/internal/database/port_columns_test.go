package database

import (
	"galgame-maker/internal/config"
	"galgame-maker/internal/models"
	"path/filepath"
	"testing"
)

func TestPortColumnCleanupPreservesRecordsIndexesAndSequence(t *testing.T) {
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "ports.db")}
	db, err := Init(cfg)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	t.Cleanup(func() { conn.Close() })
	for _, q := range []string{
		"ALTER TABLE ports ADD COLUMN x REAL",
		"ALTER TABLE ports ADD COLUMN y REAL",
		"ALTER TABLE ports ADD COLUMN is_first NUMERIC",
		"CREATE INDEX port_sort_test ON ports(sort_order)",
		"INSERT INTO ports(id,hash,option_content,scene_id,export_hash,show_option) VALUES(5,'keep','choice',1,'target',false)",
		"INSERT INTO ports(id,hash,scene_id,export_hash) VALUES(100,'deleted',1,'target')",
		"DELETE FROM ports WHERE id = 100",
	} {
		if err := db.Exec(q).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := removeLegacyPortColumns(db); err != nil {
			t.Fatal(err)
		}
	}
	columns, err := db.Migrator().ColumnTypes(&models.Node{})
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range columns {
		if column.Name() == "x" || column.Name() == "y" || column.Name() == "is_first" {
			t.Fatalf("column %s survived", column.Name())
		}
	}
	for _, idx := range []string{"idx_ports_hash", "idx_ports_scene_id", "idx_ports_export_hash", "port_sort_test"} {
		if !db.Migrator().HasIndex(&models.Node{}, idx) {
			t.Fatalf("index %s lost", idx)
		}
	}
	var keep models.Node
	if err := db.First(&keep, 5).Error; err != nil || keep.Name != "choice" || keep.ShowOption {
		t.Fatal("data changed")
	}
	next := models.Node{Hash: "next", SceneID: 1, ExportHash: "target"}
	if err := db.Create(&next).Error; err != nil || next.ID <= 100 {
		t.Fatal("autoincrement could reuse a history ID")
	}
	if !db.Migrator().HasColumn(&models.HtmlScene{}, "is_first") {
		t.Fatal("story start field removed")
	}
}

func TestPortColumnCleanupRollsBackWhenColumnIsReferenced(t *testing.T) {
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "ports.db")}
	db, err := Init(cfg)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	t.Cleanup(func() { conn.Close() })
	for _, q := range []string{"ALTER TABLE ports ADD COLUMN x REAL", "ALTER TABLE ports ADD COLUMN y REAL", "CREATE INDEX port_y_test ON ports(y)"} {
		if err := db.Exec(q).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := removeLegacyPortColumns(db); err == nil {
		t.Fatal("dependent index was silently removed")
	}
	if !db.Migrator().HasColumn(&models.Node{}, "x") || !db.Migrator().HasIndex(&models.Node{}, "port_y_test") {
		t.Fatal("cleanup was partially committed")
	}
}
