// 画布布局常量（全局唯一来源，禁止散落魔法值）
// 视觉模型：开源思维导图 —— 圆角节点卡片 + 右侧分支，S 形平滑连线，无左右端口列区域

// ---- 节点卡片 ----
export const NODE_W = 200 // 节点卡片宽
export const NODE_H = 44 // 节点基础高（仅标题行）
export const NODE_RADIUS = 12 // 节点圆角半径

// ---- 端口（选项球） ----
// 节点卡片高度固定 NODE_H，端口球在节点边缘以固定行距散开（超出卡片自然延伸）
export const PORT_ROW_H = 26 // 端口竖直间距（散开行距）
export const PORT_PAD = 8 // 端口列上下留白（参与布局占位高度）
export const PORT_OFFSET = 40 // 端口球心到节点边缘的距离（拉开，不贴边）
export const BALL_R = 6 // 端口球半径

// ---- 树形自动布局 ----
export const ROOT_X = 120 // 根节点起始 x
export const ROOT_Y = 320 // 根块垂直中心
export const LEVEL_GAP = 240 // 父子层级水平净间距（左右拉开，不挤）
export const CHILD_GAP = 24 // 兄弟节点竖直净间距

// ---- 连线 ----
export const CURVE_MIN_BEND = 60 // S 曲线最小弯曲段（控制点与端点的最小水平距离）
export const LINE_W = 2 // 连线宽

// ---- 视图 ----
export const ZOOM_MIN = 0.25
export const ZOOM_MAX = 3

// ---- 分支调色板（按同一父节点下出端口顺序取色，连线与端口球跟随） ----
export const MIND_COLORS = [
  '#5b8def', // 蓝
  '#5dc3a2', // 绿
  '#f2a65a', // 橙
  '#e5738e', // 粉
  '#9a7be8', // 紫
  '#4fc3f7', // 天蓝
  '#ffb74d', // 金
  '#81c784' // 草绿
]
export const MAGIC_COLOR = '#ef4444' // 魔法端口专用纯红

