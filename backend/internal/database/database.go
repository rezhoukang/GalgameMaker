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

	// 自动建表
	if err := db.AutoMigrate(
		&models.Setting{},
		&models.Folder{},
		&models.Canvas{},
		&models.HtmlScene{},
		&models.Node{},
	); err != nil {
		return nil, err
	}

	// 旧版连线表已废弃（端口配对改为「哈希相同即相连」），若残留则清空
	db.Exec("DROP TABLE IF EXISTS connections")

	// 旧版端口哈希是全局唯一索引；新版改为「配对端口共用同一哈希」，需移除唯一约束
	db.Exec("DROP INDEX IF EXISTS idx_nodes_hash")

	// SQLite 开启外键约束
	db.Exec("PRAGMA foreign_keys = ON")

	log.Println("数据库初始化完成:", cfg.DBPath)
	return db, nil
}
