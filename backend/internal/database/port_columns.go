package database

import (
	"galgame-maker/internal/models"
	"gorm.io/gorm"
)

// Native DROP COLUMN preserves indexes, triggers and the AUTOINCREMENT high-water
// mark. This SQLite driver's Migrator.DropColumn rebuilds the table and loses them.
// DDL is limited to these fixed, obsolete columns; all changes share a transaction.
func removeLegacyPortColumns(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		columns, err := tx.Migrator().ColumnTypes(&models.Node{})
		if err != nil {
			return err
		}
		present := make(map[string]bool, len(columns))
		for _, column := range columns {
			present[column.Name()] = true
		}
		for _, column := range []string{"x", "y", "is_first"} {
			if present[column] {
				if err := tx.Exec("ALTER TABLE ports DROP COLUMN " + column).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
