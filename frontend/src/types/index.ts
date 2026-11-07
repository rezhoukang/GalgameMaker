// 与后端 API 对应的数据类型定义

export interface Settings {
  storagePath: string // 素材存储根目录的本地完整路径
  configured: boolean // 存储配置是否可用；当前路径固定，后端返回 true
  outputDir: string // 导出文件的本地根目录
}

// 树结构：文件夹 → 画布
export interface TreeCanvas {
  id: number // 工程 ID，对应 canvas.id
  name: string // 工程显示名称
  nodeCount: number // 工程内剧情节点数量，由后端查询统计
}

export interface TreeFolder {
  id: number // 文件夹 ID，对应 folders.id
  parentId: number | null // 父文件夹 ID；null 表示导航根目录下的文件夹
  name: string // 文件夹显示名称
  canvases: TreeCanvas[] // 直接属于该文件夹的工程列表
}

// 画布（树里的「文件」）
export interface Canvas {
  id: number // 工程 ID，对应 canvas.id
  folderId: number // 所属文件夹 ID，对应 folders.id
  name: string // 工程显示名称
}

// 剧情节点，对应数据库 nodes 表，可关联视频或 HTML 内容。
export interface Node {
  level: number // 普通树层级，根节点为 1，不计魔法连接
  assetDir: string // 相对工程目录的素材目录，可通过复制属性共享引用
  htmlFile: string // HTML 文件相对素材目录的路径；空串表示未导入
  id: number // 剧情节点 ID，对应 nodes.id
  canvasId: number // 所属工程 ID，对应 canvas.id
  hash: string // 节点全局唯一标识，用于连接目标、定位和导出路由
  name: string // 节点显示名称，与素材文件名独立
  x: number // 节点横坐标，前端自动布局重新计算
  y: number // 节点纵坐标，前端自动布局重新计算
  video: string // 视频相对素材目录的路径；空串表示未指定视频文件
  first: boolean // 是否为所属工程的播放起点，不是全局唯一
}

// 端口视图：后端保存出端口，前端派生入端口。
// 后端仅返回出端口；前端按 exportHash 生成负 ID 的入端口。
export interface PortItem {
  exportHash: string // 目标剧情节点 hash，对应 nodes.hash
  showOption: boolean // 是否显示选项；隐藏时要求源节点恰好一个出口，播放结束自动跳转
  sortOrder: number // 源节点内的出端口排序值，后端按此值及 ID 升序返回
  id: number // 出端口 ID 为正数；派生入端口使用对应负 ID
  hash: string // 出端口唯一标识；派生入端口沿用，用于配对和颜色同步
  name: string // 选项显示文案，对应数据库 option_content
  nodeId: number // 当前端点所在的节点 ID；出端口对应源节点，派生入端口对应目标节点
  targetKind: string // 目标内容类型：mp4 播放视频，html 打开 HTML 页面
  entry: boolean // true 为前端派生入端口，false 为后端保存的出端口
  magic: boolean // 是否为魔法连接；不画线，不参与普通树层级和子树归属
}

// 画布完整视图
export interface CanvasView {
  canvasId: number // 当前工程 ID
  token: string // 后端生成的快照签名，用于校验撤回内容和拒绝过期覆盖
  scenes: Node[] // 剧情节点列表，对应 nodes 表
  nodes: PortItem[] // 出端口列表，对应 ports 表；接口字段名沿用 nodes，入端口由前端派生
}

// 导出结果
export interface ExportResult {
  outputDir: string // 本次导出结果所在的本地完整目录路径
  fileCount: number // 导出文件总数，包含播放器入口和素材
  files: string[] // 导出文件名列表；素材位于导出目录的 dist 子目录
}

export type ItemKind = 'folder' | 'canvas'
