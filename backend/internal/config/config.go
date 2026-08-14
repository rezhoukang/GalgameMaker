// Package config 负责加载后端运行配置（数据库路径、端口、存储目录等）。
package config

import (
	"os"
	"path/filepath"
)

// Config 后端基础配置。
// 所有数据位置固定在后端运行目录下（暂时锁死，需要灵活时再放开）。
type Config struct {
	// DBPath SQLite 数据库文件路径（固定目录 Index SQLite）。
	DBPath string
	// Port 后端监听端口。
	Port string
	// StorageRoot 本地存储根目录（文件夹映射到该目录下，固定 Galgame Makerope）。
	StorageRoot string
	// OutputDir 导出目录（固定 Galgame Output）。
	OutputDir string
}

// Load 返回固定配置：存储 Galgame Makerope、数据库 Index SQLite、导出 Galgame Output。
func Load() *Config {
	// 以后端运行目录为基准（air / go run 均在 backend 目录下启动）
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	return &Config{
		DBPath:      filepath.Join(wd, "Index SQLite", "galgame.db"),
		Port:        "8787",
		StorageRoot: filepath.Join(wd, "Galgame Makerope"),
		OutputDir:   filepath.Join(wd, "Galgame Output"),
	}
}
