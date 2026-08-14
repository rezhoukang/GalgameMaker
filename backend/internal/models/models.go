// Package models 定义数据实体：设置、文件夹、画布、HTML 框、节点、连线。
package models

import "time"

// Setting 键值对设置，用于存储本地存储目录等。
type Setting struct {
	Key   string `gorm:"primaryKey" json:"key"`
	Value string `json:"value"`
}

// Folder 文件夹，映射到本地文件系统的真实目录。
type Folder struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ParentID  *uint     `json:"parentId"`
	Name      string    `json:"name"`
	Path      string    `json:"path"` // 本地文件系统真实路径
	CreatedAt time.Time `json:"createdAt"`
}

// Canvas 画布（树里的「文件」），隶属于某个文件夹。
// 画布内可添加多个 HTML 框（制作者做好的 HTML），每个 HTML 框上再有节点。
type Canvas struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FolderID  uint      `json:"folderId"`
	Name      string    `json:"name"` // 画布名称
	Path      string    `json:"path"` // 画布真实目录（存放 HTML 文件）
	CreatedAt time.Time `json:"createdAt"`
}

// HtmlScene HTML 框：制作者做好的一个 HTML 文件，位于画布上，内含多个节点。
// 哈希（Hash）作为节点的固定标识，节点 = 节点哈希 + 端口哈希，用于魔法跳跃。
type HtmlScene struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CanvasID  uint      `json:"canvasId"`
	Hash      string    `gorm:"uniqueIndex;size:16" json:"hash"` // 节点哈希（全局唯一）
	Name      string    `json:"name"`                            // HTML 文件名（含 .html）
	X         float64   `json:"x"`                               // 画布坐标
	Y         float64   `json:"y"`
	Width     float64   `json:"width"`    // 卡片宽度（0 表示用前端默认）
	Video     string    `json:"video"`    // 关联的 mp4 文件名（空表示无视频）
	IsFirst   bool      `json:"first"`    // 是否「开始」区块（全局唯一，导出入口）
	OutCount  int       `json:"outCount"` // 普通出端口数（出边数）
	InCount   int       `json:"inCount"`  // 普通入端口数（入边数）
	HasMagic  bool      `json:"hasMagic"` // 是否有魔法端口（0/1）
	CreatedAt time.Time `json:"createdAt"`
}

// Node 端口：属于某个 HTML 框（SceneID），坐标相对框内。
// 端口是故事线的跳转锚点。一对「出端口 + 入端口」共用同一个哈希（Hash）：
// 两个端口哈希相同即相连（不再需要独立的连线表，哈希即连接）。
//   - Entry = true  → 入端口（被进入/接收端，渲染在节点左侧）
//   - Entry = false → 出端口（选项端，渲染在节点右侧）
//   - Magic = true  → 魔法端口对（画布不画线，导出生效）
// TargetKind 只对出端口有意义：跳转到目标节点文件夹里的资源类型（mp4=视频 / html=页面）。
type Node struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Hash          string    `gorm:"size:32;index" json:"hash"` // 配对端口共用（成对唯一，不再全局唯一）
	Name          string    `json:"name"`
	SceneID       uint      `json:"sceneId"` // 所属 HTML 框
	X             float64   `json:"x"`       // 框内相对坐标
	Y             float64   `json:"y"`
	IsFirst       bool      `json:"first"`                                // 是否「开始」节点（全局唯一）
	TargetKind    string    `gorm:"size:8;default:mp4" json:"targetKind"` // 出端口跳转类型：mp4 / html
	EndingContent string    `json:"endingContent"`                        // 结局内容（末节点播完黑屏显示）
	Entry         bool      `json:"entry"`                                // 是否入端口
	Magic         bool      `json:"magic"`                                // 是否魔法端口对
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
