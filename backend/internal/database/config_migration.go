package database

import (
	"fmt"

	"galgame-maker/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// migrateConfig preserves all values, particularly the key used by saved undo history.
func migrateConfig(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if !tx.Migrator().HasTable("settings") {
			return nil
		}
		if !tx.Migrator().HasTable("config") {
			return tx.Migrator().RenameTable("settings", "config")
		}
		var old, current []models.Config
		if err := tx.Table("settings").Find(&old).Error; err != nil {
			return err
		}
		if err := tx.Find(&current).Error; err != nil {
			return err
		}
		values := map[string]string{}
		for _, item := range current {
			values[item.Key] = item.Value
		}
		for _, item := range old {
			if value, exists := values[item.Key]; exists && value != item.Value {
				return fmt.Errorf("settings 和 config 中的配置项 %q 冲突，迁移已取消", item.Key)
			}
		}
		if len(old) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&old).Error; err != nil {
				return err
			}
		}
		return tx.Migrator().DropTable("settings")
	})
}
