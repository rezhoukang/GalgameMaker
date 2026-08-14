package service

import (
	"os"

	"gorm.io/gorm"

	"galgame-maker/internal/config"
)

// SettingsService 本地存储配置：路径固定在后端目录下，无需用户选择。
type SettingsService struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewSettingsService 创建设置服务。
func NewSettingsService(db *gorm.DB, cfg *config.Config) *SettingsService {
	return &SettingsService{db: db, cfg: cfg}
}

// GetStoragePath 返回本地存储根目录（backend/storage）。
func (s *SettingsService) GetStoragePath() string {
	return s.cfg.StorageRoot
}

// GetOutputDir 返回导出目录（backend/Galgame Output）。
func (s *SettingsService) GetOutputDir() string {
	return s.cfg.OutputDir
}

// EnsureStorageConfigured 确保存储与导出目录已创建。
func (s *SettingsService) EnsureStorageConfigured() error {
	if err := os.MkdirAll(s.cfg.StorageRoot, 0o755); err != nil {
		return err
	}
	return os.MkdirAll(s.cfg.OutputDir, 0o755)
}

// SettingsDTO 设置接口返回结构（路径固定，只读展示）。
type SettingsDTO struct {
	StoragePath string `json:"storagePath"`
	Configured  bool   `json:"configured"`
	OutputDir   string `json:"outputDir"`
}

// GetSettings 返回当前设置。
func (s *SettingsService) GetSettings() SettingsDTO {
	return SettingsDTO{
		StoragePath: s.cfg.StorageRoot,
		Configured:  true,
		OutputDir:   s.cfg.OutputDir,
	}
}
