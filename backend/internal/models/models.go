// Package models 定义配置、文件夹、工程、剧情节点和出端口。
package models

import (
	"gorm.io/gorm"
	"time"
)

// Config 保存应用配置和撤回签名密钥。
type Config struct {
	Key   string `gorm:"primaryKey;comment:配置项唯一键，例如 history.signing.key" json:"key"` // 配置项唯一键，例如 history.signing.key
	Value string `gorm:"comment:配置项内容，按字符串保存" json:"value"`                           // 配置项内容，按字符串保存
}

func (Config) TableName() string { return "config" }

// Folder 文件夹，映射到本地文件系统的真实目录。
type Folder struct {
	DeleteAt  gorm.DeletedAt `gorm:"column:delete_at;index;comment:软删除时间；NULL 表示正常，删除时保留目录和素材" json:"-"` // 软删除时间；NULL 表示正常，删除时保留目录和素材
	ID        uint           `gorm:"primaryKey;comment:文件夹自增主键" json:"id"`                               // 文件夹自增主键
	ParentID  *uint          `gorm:"comment:父文件夹 ID，关联 folders.id；NULL 表示导航根目录下的文件夹" json:"parentId"`    // 父文件夹 ID，关联 folders.id；NULL 表示导航根目录下的文件夹
	Name      string         `gorm:"not null;default:'未命名';comment:文件夹名称，也是本地目录名称" json:"name"`          // 文件夹名称，也是本地目录名称
	Path      string         `gorm:"comment:文件夹在本机文件系统中的完整路径" json:"path"`                               // 文件夹在本机文件系统中的完整路径
	CreatedAt time.Time      `gorm:"comment:文件夹创建时间，由 GORM 自动填写" json:"createdAt"`                       // 文件夹创建时间，由 GORM 自动填写
}

// Canvas 画布（树里的「文件」），隶属于某个文件夹。
// 工程包含剧情节点，每个剧情节点可关联视频或 HTML 内容。
type Canvas struct {
	DeleteAt  gorm.DeletedAt `gorm:"column:delete_at;index;comment:工程软删除时间；NULL 表示正常，删除时保留节点、端口和素材" json:"-"` // 工程软删除时间；NULL 表示正常，删除时保留节点、端口和素材
	ID        uint           `gorm:"primaryKey;comment:工程自增主键" json:"id"`                                     // 工程自增主键
	FolderID  uint           `gorm:"comment:所属文件夹 ID，关联 folders.id，由业务代码校验" json:"folderId"`                  // 所属文件夹 ID，关联 folders.id，由业务代码校验
	Name      string         `gorm:"not null;default:'未命名';comment:工程显示名称，导航栏将工程显示为文件" json:"name"`           // 工程显示名称，导航栏将工程显示为文件
	Path      string         `gorm:"comment:工程素材的本地完整目录路径" json:"path"`                                       // 工程素材的本地完整目录路径
	CreatedAt time.Time      `gorm:"comment:工程创建时间，由 GORM 自动填写" json:"createdAt"`                             // 工程创建时间，由 GORM 自动填写
}

// Node 剧情节点，对应 nodes 表，可关联视频或 HTML 内容。
type Node struct {
	HTMLPath  string    `gorm:"not null;default:'';comment:节点 HTML 页面相对于工程目录的路径；空串表示未导入内容；复制与撤回可共享内容版本" json:"htmlPath"` // 节点 HTML 页面相对于工程目录的路径；空串表示未导入内容；复制与撤回可共享内容版本
	ID        uint      `gorm:"primaryKey;comment:剧情节点自增主键" json:"id"`                                                   // 剧情节点自增主键
	CanvasID  uint      `gorm:"comment:所属工程 ID，关联 canvas.id，由业务代码校验" json:"canvasId"`                                    // 所属工程 ID，关联 canvas.id，由业务代码校验
	Hash      string    `gorm:"uniqueIndex;size:16;not null;comment:节点的全局唯一标识，供端口跳转、哈希定位及导出路由使用" json:"hash"`            // 节点的全局唯一标识，供端口跳转、哈希定位及导出路由使用
	Name      string    `gorm:"not null;default:'未命名';comment:节点显示名称，与素材目录名称独立" json:"name"`                             // 节点显示名称，与素材目录名称独立
	X         float64   `gorm:"comment:节点横坐标；当前前端自动布局会重新计算，不用于保存自由拖动位置" json:"x"`                                        // 节点横坐标；当前前端自动布局会重新计算，不用于保存自由拖动位置
	Y         float64   `gorm:"comment:节点纵坐标；当前前端自动布局会重新计算，不用于保存自由拖动位置" json:"y"`                                        // 节点纵坐标；当前前端自动布局会重新计算，不用于保存自由拖动位置
	IsFirst   bool      `gorm:"comment:是否为本工程播放起点；业务逻辑保证工程内恰好一个，可与根节点不同" json:"first"`                                   // 是否为本工程播放起点；业务逻辑保证工程内恰好一个，可与根节点不同
	CreatedAt time.Time `gorm:"comment:节点创建时间，由 GORM 自动填写" json:"createdAt"`                                             // 节点创建时间，由 GORM 自动填写
	Level     int       `gorm:"not null;default:1;comment:普通连接树中的层级，每个工程恰好一个层级为 1 的根节点且不可删除；结构变化时重新计算" json:"level"`     // 普通连接树中的层级，每个工程恰好一个层级为 1 的根节点且不可删除；结构变化时重新计算
}

// Port 出端口，对应 ports 表；入端口仅由前端派生，不单独保存。
type Port struct {
	ID         uint      `gorm:"primaryKey;comment:出端口自增主键；前端派生的入端口使用对应负 ID" json:"id"`                                        // 出端口自增主键；前端派生的入端口使用对应负 ID
	Hash       string    `gorm:"size:32;uniqueIndex;not null;comment:出端口全局唯一标识；前端派生入端口沿用该值，用于配对和颜色同步" json:"hash"`             // 出端口全局唯一标识；前端派生入端口沿用该值，用于配对和颜色同步
	Name       string    `gorm:"column:option_content;not null;default:'选项';comment:选项显示文案，数据库列名为 option_content" json:"name"` // 选项显示文案，数据库列名为 option_content
	NodeID     uint      `gorm:"not null;index;comment:所属源节点 ID，关联 nodes.id" json:"nodeId"`                                    // 所属源节点 ID，关联 nodes.id
	Entry      bool      `gorm:"-" json:"entry"`                                                                               // 仅供 API 和前端区分方向，不入库；后端出端口为 false，前端派生入端口为 true
	Magic      bool      `gorm:"not null;default:false;comment:是否为魔法出端口；true 不画连接线，也不参与普通树层级和子树归属" json:"magic"`               // 是否为魔法出端口；true 不画连接线，也不参与普通树层级和子树归属
	CreatedAt  time.Time `gorm:"comment:端口创建时间，由 GORM 自动填写" json:"createdAt"`                                                  // 端口创建时间，由 GORM 自动填写
	UpdatedAt  time.Time `gorm:"comment:端口最后更新时间，由 GORM 在更新记录时维护" json:"updatedAt"`                                            // 端口最后更新时间，由 GORM 在更新记录时维护
	ExportHash string    `gorm:"not null;index;comment:目标剧情节点的 hash，关联 nodes.hash；目标必须属于同一工程" json:"exportHash"`               // 目标剧情节点的 hash，关联 nodes.hash；目标必须属于同一工程
	ShowOption bool      `gorm:"not null;default:true;comment:是否显示选项；false 仅在源节点恰好一个出口时用于自动跳转" json:"showOption"`              // 是否显示选项；false 仅在源节点恰好一个出口时用于自动跳转
	SortOrder  int       `gorm:"not null;default:0;comment:源节点内的出端口排序值，升序排列，相同值按端口 ID 排序" json:"sortOrder"`                    // 源节点内的出端口排序值，升序排列，相同值按端口 ID 排序
}

// TableName 指定出端口表名。
func (Port) TableName() string { return "ports" }
