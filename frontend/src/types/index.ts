// 与后端 API 对应的数据类型定义

export interface Settings {
  storagePath: string
  configured: boolean
  outputDir: string
}

// 树结构：文件夹 → 画布
export interface TreeCanvas {
  id: number
  name: string
  nodeCount: number
}

export interface TreeFolder {
  id: number
  parentId: number | null
  name: string
  canvases: TreeCanvas[]
}

// 画布（树里的「文件」）
export interface Canvas {
  id: number
  folderId: number
  name: string
}

// HTML 框：制作者做好的一个 HTML，位于画布上，内含多个节点
export interface HtmlScene {
  level: number
  portNum: number
  isRoot: boolean
  assetDir: string
  htmlFile: string
  id: number
  canvasId: number
  hash: string // 节点哈希（全局唯一，魔法跳跃用）
  name: string // HTML 文件名
  x: number // 画布坐标
  y: number
  width: number // 卡片宽度（0 表示用默认）
  video: string // 关联 mp4 文件名（空表示无视频）
  first: boolean // 是否「开始」区块（全局唯一，导出入口）
  outCount: number // 普通出端口数（出边数）
  inCount: number // 普通入端口数（入边数）
  hasMagic: boolean // 是否有魔法端口（0/1）
}

// 节点（端口）：属于某个 HTML 框，坐标相对框内。
// 后端仅返回出端口；前端按 exportHash 生成负 ID 的入端口。
export interface NodeItem {
  exportHash: string
  showOption: boolean
  sortOrder: number
  id: number
  hash: string // 出端口标识；生成的入端口沿用此标识
  name: string
  sceneId: number
  targetKind: string // 出端口跳转类型：mp4 / html
  endingContent: string // 结局内容（末节点播完黑屏居中显示）
  entry: boolean // 是否入端口（true=左侧接收端；false=右侧出端口）
  magic: boolean // 是否魔法端口对（画布不画线，导出生效）
}

// 画布完整视图
export interface CanvasView {
  canvasId: number
  token: string
  scenes: HtmlScene[]
  nodes: NodeItem[]
}

// 导出结果
export interface ExportResult {
  outputDir: string
  fileCount: number
  files: string[]
}

export type ItemKind = 'folder' | 'canvas'
