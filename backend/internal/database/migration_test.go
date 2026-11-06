package database

import (
	"galgame-maker/internal/config"
	"galgame-maker/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
)

func legacyFixture(t *testing.T, broken bool) (*gorm.DB, *config.Config) {
	t.Helper()
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "legacy.db")}
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&models.Config{}, &models.Canvas{}, &models.HtmlScene{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&models.Canvas{ID: 1, Name: "legacy", Path: t.TempDir()})
	db.Create(&models.HtmlScene{ID: 10, CanvasID: 1, Hash: "root", Name: "root"})
	db.Create(&models.HtmlScene{ID: 20, CanvasID: 1, Hash: "child", Name: "child"})
	db.Exec("CREATE TABLE nodes (id INTEGER PRIMARY KEY, hash TEXT, name TEXT, scene_id INTEGER, entry BOOLEAN, magic BOOLEAN, target_kind TEXT, ending_content TEXT)")
	db.Exec("INSERT INTO nodes VALUES (1,'pair','choose',10,0,0,'mp4','')")
	if !broken {
		db.Exec("INSERT INTO nodes VALUES (2,'pair','choose',20,1,0,'mp4','')")
	}
	return db, cfg
}
func TestLegacyMigrationIsLosslessAndIdempotent(t *testing.T) {
	_, cfg := legacyFixture(t, false)
	db, err := Init(cfg)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	var ports []models.Node
	db.Find(&ports)
	if len(ports) != 1 || ports[0].ExportHash != "child" || ports[0].Name != "choose" || !ports[0].ShowOption {
		t.Fatalf("bad conversion: %+v", ports)
	}
	var legacyCount int64
	db.Table("nodes").Count(&legacyCount)
	if legacyCount != 2 {
		t.Fatal("legacy backup was modified")
	}
	var root, child models.HtmlScene
	db.First(&root, 10)
	db.First(&child, 20)
	if !root.IsRoot || !root.IsFirst || root.Level != 1 || child.Level != 2 {
		t.Fatal("root/depth migration failed")
	}
	if err := migratePorts(db); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&models.Node{}).Count(&count)
	if count != 1 {
		t.Fatal("migration ran twice")
	}
}
func TestInvalidLegacyPairDoesNotPartiallyMigrate(t *testing.T) {
	db, cfg := legacyFixture(t, true)
	if _, err := Init(cfg); err == nil {
		t.Fatal("broken legacy pair accepted")
	}
	var root models.HtmlScene
	db.First(&root, 10)
	if root.IsRoot {
		t.Fatal("partial scene changes committed")
	}
	var count int64
	db.Model(&models.Node{}).Count(&count)
	if count != 0 {
		t.Fatal("partial ports committed")
	}
}
