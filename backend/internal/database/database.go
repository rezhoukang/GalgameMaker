// Package database 初始化 SQLite 数据库连接并自动建表。
package database

import (
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"galgame-maker/internal/config"
	"galgame-maker/internal/models"
)

// Init 打开数据库、自动迁移表结构。
func Init(cfg *config.Config) (*gorm.DB, error) {
	// 确保数据库所在目录存在
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	ready := false
	defer func() {
		if !ready {
			if conn, err := db.DB(); err == nil {
				_ = conn.Close()
			}
		}
	}()

	// Rename before AutoMigrate so the signing key and migration markers are preserved.
	if err := migrateConfig(db); err != nil {
		return nil, err
	}

	// 自动建表
	if err := db.AutoMigrate(
		&models.Config{},
		&models.Folder{},
		&models.Canvas{},
		&models.HtmlScene{},
		&models.Node{},
	); err != nil {
		return nil, err
	}

	if err := migratePorts(db); err != nil {
		return nil, err
	}
	if err := removeLegacyPortColumns(db); err != nil {
		return nil, err
	}

	// SQLite 开启外键约束
	db.Exec("PRAGMA foreign_keys = ON")

	log.Println("数据库初始化完成:", cfg.DBPath)
	ready = true
	return db, nil
}
