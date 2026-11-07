// Package models 定义配置、文件夹、工程、剧情节点和出端口。
package models

import (
	"gorm.io/gorm"
	"time"
)

// Config 保存应用配置和撤回签名密钥。
type Config struct {
	Key   string `gorm:"primaryKey" json:"key"`
	Value string `json:"value"`
}

func (Config) TableName() string { return "config" }

// Folder 文件夹，映射到本地文件系统的真实目录。
type Folder struct {
	DeleteAt  gorm.DeletedAt `gorm:"column:delete_at;index" json:"-"`
	ID        uint           `gorm:"primaryKey" json:"id"`
	ParentID  *uint          `json:"parentId"`
	Name      string         `gorm:"not null;default:'未命名'" json:"name"`
	Path      string         `json:"path"` // 本地文件系统真实路径
	CreatedAt time.Time      `json:"createdAt"`
}

// Canvas 画布（树里的「文件」），隶属于某个文件夹。
// 工程包含剧情节点，每个剧情节点可关联视频或 HTML 内容。
type Canvas struct {
	DeleteAt  gorm.DeletedAt `gorm:"column:delete_at;index" json:"-"`
	ID        uint           `gorm:"primaryKey" json:"id"`
	FolderID  uint           `json:"folderId"`
	Name      string         `gorm:"not null;default:'未命名'" json:"name"` // 画布名称
	Path      string         `json:"path"`                               // 画布真实目录（存放 HTML 文件）
	CreatedAt time.Time      `json:"createdAt"`
}

// Node is a story node. Media references use a stable directory independent of its name.
type Node struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CanvasID  uint      `json:"canvasId"`
	Hash      string    `gorm:"uniqueIndex;size:16;not null" json:"hash"` // 节点哈希（全局唯一）
	Name      string    `gorm:"not null;default:'未命名'" json:"name"`       // 显示名称，与素材目录无关
	X         float64   `json:"x"`                                        // 画布坐标
	Y         float64   `json:"y"`
	Video     string    `gorm:"not null;default:''" json:"video"` // 关联的 mp4 文件名（空表示无视频）
	IsFirst   bool      `json:"first"`                            // 是否「开始」区块（工程内唯一，导出入口）
	OutCount  int       `json:"outCount"`                         // 普通出端口数（出边数）
	InCount   int       `json:"inCount"`                          // 普通入端口数（入边数）
	HasMagic  bool      `json:"hasMagic"`                         // 是否有魔法端口（0/1）
	CreatedAt time.Time `json:"createdAt"`
	Level     int       `gorm:"not null;default:1" json:"level"`
	PortNum   int       `gorm:"not null;default:0" json:"portNum"`
	IsRoot    bool      `gorm:"not null;default:false" json:"isRoot"`
	AssetDir  string    `gorm:"not null;default:''" json:"assetDir"`
	HTMLFile  string    `gorm:"not null;default:''" json:"htmlFile"`
}

// Port stores one outgoing port in ports. The incoming endpoint is derived by the frontend.
type Port struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Hash          string    `gorm:"size:32;uniqueIndex;not null" json:"hash"` // 出端口的稳定唯一标识
	Name          string    `gorm:"column:option_content;not null;default:'选项'" json:"name"`
	SceneID       uint      `gorm:"not null;index" json:"sceneId"`                 // 所属剧情节点
	TargetKind    string    `gorm:"size:8;not null;default:mp4" json:"targetKind"` // 出端口跳转类型：mp4 / html
	EndingContent string    `gorm:"not null;default:''" json:"endingContent"`      // 结局内容（末节点播完黑屏显示）
	Entry         bool      `gorm:"-" json:"entry"`                                // 仅用于前端生成的入端口，不写数据库
	Magic         bool      `gorm:"not null;default:false" json:"magic"`           // 是否魔法端口对
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ExportHash    string    `gorm:"not null;index" json:"exportHash"`
	ShowOption    bool      `gorm:"not null;default:true" json:"showOption"`
	SortOrder     int       `gorm:"not null;default:0" json:"sortOrder"`
}

// Port is one outgoing port.
func (Port) TableName() string { return "ports" }
