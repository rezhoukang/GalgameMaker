package database

import (
	"galgame-maker/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
)

func configDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "config.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	t.Cleanup(func() { conn.Close() })
	if err := db.Exec("CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	return db
}
func TestConfigRenamePreservesValuesAndIsIdempotent(t *testing.T) {
	db := configDB(t)
	items := []models.Config{{Key: "history.signing.key", Value: "test-history-key"}, {Key: "schema.outgoing.v1", Value: "1"}}
	if err := db.Table("settings").Create(&items).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := migrateConfig(db); err != nil {
			t.Fatal(err)
		}
	}
	if db.Migrator().HasTable("settings") || !db.Migrator().HasTable("config") {
		t.Fatal("table was not renamed")
	}
	for _, want := range items {
		var got models.Config
		if err := db.First(&got, "key = ?", want.Key).Error; err != nil || got.Value != want.Value {
			t.Fatal("configuration value changed")
		}
	}
}
func TestConfigMergeAndConflictRollback(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		name := "merge"
		if conflict {
			name = "conflict"
		}
		t.Run(name, func(t *testing.T) {
			db := configDB(t)
			if err := db.AutoMigrate(&models.Config{}); err != nil {
				t.Fatal(err)
			}
			old := []models.Config{{Key: "history.signing.key", Value: "old-key"}, {Key: "schema.outgoing.v1", Value: "1"}}
			if err := db.Table("settings").Create(&old).Error; err != nil {
				t.Fatal(err)
			}
			value := "old-key"
			if conflict {
				value = "different-key"
			}
			if err := db.Create(&models.Config{Key: "history.signing.key", Value: value}).Error; err != nil {
				t.Fatal(err)
			}
			err := migrateConfig(db)
			if conflict {
				if err == nil || !db.Migrator().HasTable("settings") {
					t.Fatal("conflicting values were overwritten")
				}
				var count int64
				db.Model(&models.Config{}).Count(&count)
				if count != 1 {
					t.Fatal("partial merge was committed")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				var count int64
				db.Model(&models.Config{}).Count(&count)
				if count != 2 || db.Migrator().HasTable("settings") {
					t.Fatal("merge lost keys or left old table")
				}
			}
		})
	}
}
