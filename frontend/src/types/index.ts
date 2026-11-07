// 与后端 API 对应的数据类型定义

export interface Settings {
  storagePath: string // 素材存储根目录的本地完整路径
  configured: boolean // 存储配置是否可用；当前路径固定，后端返回 true
  outputDir: string // 导出文件的本地根目录
}

// 树结构：文件夹 → 画布
export interface TreeCanvas {
  id: number // 工程 ID，对应 projects.id
  name: string // 工程显示名称
  nodeCount: number // 工程内剧情节点数量，由后端查询统计
}

export interface TreeFolder {
  id: number // 文件夹 ID，对应 folders.id
  parentId: number | null // 父文件夹 ID；null 表示导航根目录下的文件夹
  name: string // 文件夹显示名称
  canvases: TreeCanvas[] // 直接属于该文件夹的工程列表
}

// 工程（树里的「文件」），对应 projects 表
export interface Project {
  id: number // 工程 ID，对应 projects.id
  folderId: number // 所属文件夹 ID，对应 folders.id
  name: string // 工程显示名称
}

// 剧情节点，对应数据库 nodes 表，可关联视频或 HTML 内容。
export interface NodeData {
  contentPath: string // 节点内容入口相对于工程目录的路径（当前为 HTML 页面）；空串表示未导入内容
  level: number // 普通树层级，根节点为 1，不计魔法连接
  id: number // 剧情节点 ID，对应 nodes.id
  projectId: number // 所属工程 ID，对应 projects.id
  hash: string // 节点全局唯一标识，用于连接目标、定位和导出路由
  name: string // 节点显示名称，与素材文件名独立
  first: boolean // 是否为所属工程的播放起点，不是全局唯一
}

// 仅用于画布渲染的节点；坐标不进入接口、数据库或撤回快照。
export interface Node extends NodeData {
  x: number // 前端自动计算的横坐标
  y: number // 前端自动计算的纵坐标
}

// 端口视图：后端保存出端口，前端派生入端口。
// 后端仅返回出端口；前端按 exportHash 生成负 ID 的入端口。
export interface PortItem {
  conditionMode: 'all' | 'any' // 条件组合方式
  resetLoop: boolean // 经过时先重置本轮变量，再执行增减
  conditions: PortCondition[] // 端口出现条件
  effects: PortEffect[] // 经过端口的变量变化
  exportHash: string // 目标剧情节点 hash，对应 nodes.hash
  isShow: boolean // 是否显示选项；单出口隐藏时自动跳转，多出口隐藏时不展示此项
  sortOrder: number // 源节点内的出端口排序值，后端按此值及 ID 升序返回
  id: number // 出端口 ID 为正数；派生入端口使用对应负 ID
  hash: string // 出端口唯一标识；派生入端口沿用，用于配对和颜色同步
  name: string // 选项显示文案，对应数据库 option_content
  nodeId: number // 当前端点所在的节点 ID；出端口对应源节点，派生入端口对应目标节点
  entry: boolean // true 为前端派生入端口，false 为后端保存的出端口
  magic: boolean // 是否为魔法连接；不画线，不参与普通树层级和子树归属
}

// 画布完整视图
export interface CanvasView {
  variables: StoryVariable[] // 工程变量定义，参与撤回
  projectId: number // 当前工程 ID
  token: string // 后端生成的快照签名，用于校验撤回内容和拒绝过期覆盖
  scenes: NodeData[] // 剧情节点列表，对应 nodes 表
  nodes: PortItem[] // 出端口列表，对应 ports 表；接口字段名沿用 nodes，入端口由前端派生
}

// 导出结果
export interface ExportResult {
  outputDir: string // 本次导出结果所在的本地完整目录路径
  fileCount: number // 导出文件总数，包含播放器入口和素材
  files: string[] // 导出文件名列表；素材位于导出目录的 dist 子目录
}

export type ItemKind = 'folder' | 'canvas'

export interface StoryVariable {
  id: number // 变量 ID，新增草稿使用 0
  projectId: number // 所属工程
  name: string // 工程内唯一名称
  initialValue: number // 新游戏初始整数值
  resetOnLoop: boolean // 开始新一轮是否恢复初始值
}
export interface PortCondition {
  variableId: number // 检查的变量
  operator: '>=' | '<=' | '==' | '!=' | '>' | '<' // 比较运算
  value: number // 比较阈值
}
export interface PortEffect {
  variableId: number // 修改的变量
  delta: number // 正数增加，负数减少
}
export interface PortRules {
  conditionMode: 'all' | 'any' // 多条条件的组合方式
  resetLoop: boolean // 开始新一轮
  conditions: PortCondition[] // 无条件表示默认允许
  effects: PortEffect[] // 同一变量最多一条变化
}
