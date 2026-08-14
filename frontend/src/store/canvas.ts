// 画布状态：画布 = 节点（HtmlScene）+ 端口（Node）。
// 端口配对即「哈希相同」：一对出/入端口共用同一哈希，哈希相同即相连（无独立连线表）。
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import { useAppStore } from './app'
import { useTreeStore } from './tree'
import type { HtmlScene, NodeItem } from '@/types'
import {
  MAGIC_COLOR,
  MIND_COLORS,
  NODE_H,
  NODE_W,
  PORT_OFFSET,
  PORT_ROW_H,
  ZOOM_MAX,
  ZOOM_MIN
} from '@/canvasLayout'
import { computeMindLayout, sceneDisplayHeight } from '@/mindLayout'

export interface ViewState {
  x: number
  y: number
  zoom: number
}

type Interaction = { type: 'pan'; lastX: number; lastY: number } | null

function clamp(v: number, min: number, max: number) {
  return Math.min(max, Math.max(min, v))
}

/** 节点卡片宽（固定，思维导图不提供拉伸） */
function sceneWidth(scene: HtmlScene): number {
  return scene.width > 0 ? scene.width : NODE_W
}

/** 返回与 nodeId 哈希相同的配对端口（另一端）；没有则返回 null */
function pairOf(nodes: NodeItem[], nodeId: number): NodeItem | null {
  const n = nodes.find((x) => x.id === nodeId)
  if (!n) return null
  return nodes.find((x) => x.id !== nodeId && x.hash === n.hash) ?? null
}

/** 端口颜色：魔法纯红；出端口按同框出端口顺序取分支调色板；入端口颜色跟随配对出端口 */
function portColor(nodes: NodeItem[], nodeId: number): string {
  const node = nodes.find((n) => n.id === nodeId)
  if (!node) return MIND_COLORS[0]
  if (node.magic) return MAGIC_COLOR
  // 入端口：颜色 = 配对出端口（同哈希的 !entry 端口）的颜色
  if (node.entry) {
    const pair = nodes.find((n) => n.id !== nodeId && n.hash === node.hash && !n.entry)
    if (pair) return portColor(nodes, pair.id)
    return MIND_COLORS[0]
  }
  // 出端口：按同框出端口顺序取色
  const peers = nodes.filter((n) => n.sceneId === node.sceneId && !n.entry)
  const idx = Math.max(0, peers.findIndex((n) => n.id === nodeId))
  return MIND_COLORS[idx % MIND_COLORS.length]
}

export const useCanvasStore = defineStore('canvas', {
  state: () => ({
    canvasId: null as number | null,
    canvasName: '',
    scenes: [] as HtmlScene[],
    nodes: [] as NodeItem[],
    selectedSceneId: null as number | null,
    view: { x: 0, y: 0, zoom: 1 } as ViewState,
    interaction: null as Interaction,
    editingNode: null as NodeItem | null,
    editingScene: null as number | null,
    flashSceneId: null as number | null,
    collapsedSceneIds: [] as number[]
  }),

  getters: {
    nodeById: (s) => (id: number) => s.nodes.find((n) => n.id === id),
    sceneById: (s) => (id: number) => s.scenes.find((sc) => sc.id === id),

    /** 端口颜色：魔法纯红；出端口按同框出端口顺序取分支调色板；入端口颜色跟随配对出端口 */
    nodeColorById: (s) => (nodeId: number) => portColor(s.nodes, nodeId),
    isCanvasView: (s) => s.canvasId != null,

    /** 节点是否已折叠子树 */
    isCollapsed: (s) => (sceneId: number) => s.collapsedSceneIds.includes(sceneId),

    /** 节点卡片渲染高度：固定 NODE_H（布局占位见 mindLayout.sceneDisplayHeight） */
    sceneHeight: (s) => (_sceneId: number) => NODE_H,

    /** 节点是否为入端口（entry=true）→ 显示在左侧；否则右侧出端口 */
    nodeIsEntry: (s) => (nodeId: number) => !!s.nodes.find((n) => n.id === nodeId)?.entry,

    /** 按节点哈希查找 HTML 框 */
    sceneByHash: (s) => (hash: string) => {
      const h = (hash || '').trim().toLowerCase()
      if (!h) return undefined
      return s.scenes.find((sc) => (sc.hash || '').toLowerCase() === h)
    },

    /** 端口连接去向（仅出端口）：配对入端口 + 目标节点；无配对返回 null */
    outConn: (s) => (nodeId: number) => {
      const node = s.nodes.find((n) => n.id === nodeId)
      if (!node || node.entry) return null // 入端口无「连接去向」
      const toNode = pairOf(s.nodes, nodeId)
      if (!toNode) return null
      const toScene = s.scenes.find((sc) => sc.id === toNode.sceneId)
      return { toNode, toScene }
    },

    /** 被谁连入（仅入端口）：配对出端口 + 来源节点 */
    inConns: (s) => (nodeId: number) => {
      const node = s.nodes.find((n) => n.id === nodeId)
      if (!node || !node.entry) return [] // 出端口无「被谁连入」
      const fromNode = pairOf(s.nodes, nodeId)
      if (!fromNode) return []
      const fromScene = s.scenes.find((sc) => sc.id === fromNode.sceneId)
      return [{ fromNode, fromScene }]
    },

    /** 端口配对信息（出/入通用）：同一哈希的另一端 + 所属节点；无配对返回 null */
    pairInfo: (s) => (nodeId: number) => {
      const peer = pairOf(s.nodes, nodeId)
      if (!peer) return null
      const peerScene = s.scenes.find((sc) => sc.id === peer.sceneId)
      return { peer, peerScene }
    },

    /** 端口的连接关系（节点—端口—节点）：上一节点（出端口所在）/ 下一节点（入端口所在） */
    linkInfo: (s) => (nodeId: number) => {
      const node = s.nodes.find((n) => n.id === nodeId)
      if (!node) return null
      const peer = pairOf(s.nodes, nodeId)
      if (!peer) return null
      const outNode = node.entry ? peer : node
      const inNode = node.entry ? node : peer
      const prevScene = s.scenes.find((sc) => sc.id === outNode.sceneId)
      const nextScene = s.scenes.find((sc) => sc.id === inNode.sceneId)
      return { prevScene, nextScene }
    },

    /** 端口在画布上的世界坐标：入端口贴节点左边缘、出端口贴右边缘，以固定行距散开（卡片高度不变） */
    ballWorld: (s) => (node: NodeItem) => {
      const scene = s.scenes.find((sc) => sc.id === node.sceneId)
      if (!scene) return { x: node.x, y: node.y }
      const mine = s.nodes.filter((n) => n.sceneId === scene.id)
      const isEntry = node.entry
      const col = isEntry ? mine.filter((n) => n.entry) : mine.filter((n) => !n.entry)
      const idx = Math.max(0, col.findIndex((n) => n.id === node.id))
      const x = isEntry ? scene.x - PORT_OFFSET : scene.x + sceneWidth(scene) + PORT_OFFSET
      const y = scene.y + NODE_H / 2 + (idx - (col.length - 1) / 2) * PORT_ROW_H // 以卡片中心发散
      return { x, y }
    }
  },

  actions: {
    clear() {
      this.canvasId = null
      this.canvasName = ''
      this.scenes = []
      this.nodes = []
      this.selectedSceneId = null
      this.editingNode = null
      this.editingScene = null
      this.flashSceneId = null
      this.collapsedSceneIds = []
      this.interaction = null
      this.view = { x: 0, y: 0, zoom: 1 }
    },

    async openCanvas(id: number, name: string) {
      const app = useAppStore()
      try {
        const view = await api.getCanvas(id)
        this.canvasId = id
        this.canvasName = name
        this.scenes = view.scenes
        this.nodes = view.nodes
        this.syncSceneCounts()
        this.selectedSceneId = null
        this.editingNode = null
        this.editingScene = null
        this.interaction = null
        this.collapsedSceneIds = []
        await this.relayout()
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    /** 思维导图自动布局：重算所有节点坐标（结构变化/打开画布时调用），差分后持久化 */
    async relayout() {
      const pos = computeMindLayout(
        this.scenes,
        this.nodes,
        new Set(this.collapsedSceneIds)
      )
      let changed = false
      for (const sc of this.scenes) {
        const p = pos.get(sc.id)
        if (!p) continue
        if (Math.round(sc.x) !== Math.round(p.x) || Math.round(sc.y) !== Math.round(p.y)) {
          sc.x = p.x
          sc.y = p.y
          changed = true
        }
      }
      if (!changed) return
      try {
        for (const sc of this.scenes) {
          await api.updateScene(sc.id, { x: Math.round(sc.x), y: Math.round(sc.y) })
        }
      } catch (e: any) {
        useAppStore().handleApiError(e)
      }
    },

    selectScene(id: number) {
      this.selectedSceneId = id
    },

    // ---------- HTML 框操作 ----------

    async renameScene(id: number, newName: string) {
      if (typeof id !== 'number' || Number.isNaN(id)) return // 防御：id 非法不发请求
      const app = useAppStore()
      try {
        const updated = await api.updateScene(id, { name: newName })
        const i = this.scenes.findIndex((sc) => sc.id === id)
        if (i >= 0) this.scenes[i] = updated
        useTreeStore().load()
        app.notify('已重命名', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    async deleteScene(id: number) {
      const app = useAppStore()
      const sc = this.sceneById(id)
      if (!sc) return
      if (!confirm(`确定删除 HTML 框「${sc.name}」？\n其上的所有节点与连线将一并删除！`)) return
      try {
        await api.deleteScene(id)
        await this.refreshCanvas() // 后端会级联删配对端口，拉最新数据同步
        if (this.selectedSceneId === id) this.selectedSceneId = null
        this.syncSceneCounts()
        useTreeStore().load()
        await this.relayout()
        app.notify('节点已删除', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    /** 重新从后端拉取画布数据（删除级联后同步前后端） */
    async refreshCanvas() {
      if (this.canvasId == null) return
      const view = await api.getCanvas(this.canvasId)
      this.scenes = view.scenes
      this.nodes = view.nodes
      this.syncSceneCounts()
    },

    /** 从节点的右侧加号创建下一跳：出端口 + 右侧空白节点（带入端口）+ 连线 */
    async addNextNode(sceneId: number) {
      const app = useAppStore()
      try {
        const res = await api.createNextNode(sceneId)
        if (res?.scene) this.scenes.push(res.scene)
        if (res?.outNode) this.nodes.push(res.outNode)
        if (res?.entryNode) this.nodes.push(res.entryNode)
        this.syncSceneCounts()
        useTreeStore().load()
        await this.relayout()
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    /** 为 HTML 框插入关联视频 */
    async uploadVideo(id: number, file: File) {
      const app = useAppStore()
      try {
        const updated = await api.uploadSceneVideo(id, file)
        const i = this.scenes.findIndex((sc) => sc.id === id)
        if (i >= 0) this.scenes[i] = updated
        app.notify('已插入视频', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    /** 导入/覆盖节点的 HTML 内容 */
    async uploadHtml(id: number, file: File) {
      const app = useAppStore()
      try {
        const updated = await api.uploadSceneHtml(id, file)
        const i = this.scenes.findIndex((sc) => sc.id === id)
        if (i >= 0) this.scenes[i] = updated
        app.notify('已导入 HTML', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    /** 把 HTML 框设为全局唯一的「开始」区块 */
    async setFirstScene(id: number) {
      const app = useAppStore()
      try {
        const updated = await api.setFirstScene(id)
        this.scenes = this.scenes.map((s) => (s.id === id ? updated : { ...s, first: false }))
        app.notify('已设为开始区块', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    /** 定位视图到某个节点并选中（魔法跳跃） */
    jumpToScene(sceneId: number) {
      const sc = this.sceneById(sceneId)
      if (!sc) return
      this.selectedSceneId = sceneId
      this.editingNode = null
      this.editingScene = null
      const w = sceneWidth(sc)
      const cx = sc.x + w / 2
      const cy = sc.y + this.sceneHeight(sceneId) / 2
      this.view.x = window.innerWidth / 2 - cx * this.view.zoom
      this.view.y = window.innerHeight / 2 - cy * this.view.zoom
    },

    /** 按节点哈希跳跃（魔法跳跃）；找不到返回 false */
    jumpToSceneByHash(hash: string): boolean {
      const sc = this.sceneByHash(hash)
      if (!sc) return false
      this.jumpToScene(sc.id)
      return true
    },

    /** 本地重算节点端口统计字段（与后端一致），端口增删后调用 */
    syncSceneCounts() {
      for (const sc of this.scenes) {
        const mine = this.nodes.filter((n) => n.sceneId === sc.id)
        let out = 0
        let inc = 0
        let magic = false
        for (const n of mine) {
          if (n.magic) magic = true
          if (!pairOf(this.nodes, n.id)) continue
          if (n.entry) inc++
          else out++
        }
        sc.outCount = out
        sc.inCount = inc
        sc.hasMagic = magic
      }
    },

    /** 新建魔法端口：右键加号球 → 输入目标节点哈希，出端口与目标入端口对应、不连线（hidden） */
    async addMagicPort(sceneId: number, hash: string) {
      const app = useAppStore()
      try {
        const res = await api.createMagicPort(sceneId, hash)
        if (res?.outNode) this.nodes.push(res.outNode)
        if (res?.entryNode) this.nodes.push(res.entryNode)
        this.syncSceneCounts()
        app.notify('已新增魔法端口（画布不画线，导出生效）', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    /** 寻找：定位到指定哈希的节点（HTML 框）并高亮闪烁 */
    highlightSceneByHash(hash: string): boolean {
      const sc = this.sceneByHash(hash)
      if (!sc) return false
      this.jumpToScene(sc.id)
      this.flashSceneId = sc.id
      window.setTimeout(() => {
        if (this.flashSceneId === sc.id) this.flashSceneId = null
      }, 2400)
      return true
    },

    // ---------- 节点操作 ----------

    async renameNode(id: number, name: string) {
      const app = useAppStore()
      try {
        const updated = await api.updateNode(id, { name })
        const i = this.nodes.findIndex((n) => n.id === id)
        if (i >= 0) this.nodes[i] = updated
        this.editingNode = null
        useTreeStore().load()
        app.notify('已保存', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    /** 保存端口名称 + 结局内容 + 跳转类型 */
    async saveNode(id: number, data: { name: string; endingContent: string; targetKind?: string }) {
      const app = useAppStore()
      try {
        const updated = await api.updateNode(id, {
          name: data.name,
          endingContent: data.endingContent,
          targetKind: data.targetKind
        })
        const i = this.nodes.findIndex((n) => n.id === id)
        if (i >= 0) this.nodes[i] = updated
        // 后端已把配对端口同步（syncPortPair），前端也同步配对端口，避免端口球标签要刷新才更新
        const peer = pairOf(this.nodes, id)
        if (peer) {
          const j = this.nodes.findIndex((n) => n.id === peer.id)
          if (j >= 0) {
            this.nodes[j] = {
              ...this.nodes[j],
              name: updated.name,
              endingContent: updated.endingContent,
              targetKind: updated.targetKind
            }
          }
        }
        this.editingNode = null
        useTreeStore().load()
        app.notify('已保存', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    async deleteNode(id: number) {
      const app = useAppStore()
      const node = this.nodeById(id)
      if (!node) return
      const isEntry = !!node.entry
      const tip = isEntry
        ? '确定删除该入端口？其对应的出端口也将一并删除。'
        : '确定删除该出端口？其下（目标下一跳节点）的所有节点和端口将一并删除！'
      if (!confirm(`确定删除端口「${node.name}」？\n${tip}`)) return
      try {
        await api.deleteNode(id)
        await this.refreshCanvas() // 配对端口/子树由后端级联删除，拉最新数据同步
        if (this.selectedSceneId != null && !this.sceneById(this.selectedSceneId)) this.selectedSceneId = null
        if (this.editingNode?.id === id) this.editingNode = null
        this.syncSceneCounts()
        useTreeStore().load()
        await this.relayout()
        app.notify('端口已删除', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    /** 断开端口连接：删除当前端口与配对端口（保留节点，不删子树） */
    async breakNode(id: number) {
      const app = useAppStore()
      try {
        const peer = pairOf(this.nodes, id)
        await api.breakNode(id)
        this.nodes = this.nodes.filter((n) => n.id !== id && n.id !== peer?.id)
        if (this.editingNode?.id === id) this.editingNode = null
        this.syncSceneCounts()
        useTreeStore().load()
        app.notify('已断开', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    /** 重定向端口一端到指定节点（按哈希）：side=prev 改上一节点 / next 改下一节点 */
    async redirectNode(id: number, hash: string, side: 'prev' | 'next') {
      const app = useAppStore()
      try {
        await api.redirectNode(id, hash, side)
        await this.refreshCanvas()
        useTreeStore().load()
        await this.relayout()
        app.notify(side === 'prev' ? '已修改上一节点' : '已修改下一节点', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
    },

    // ---------- 视图：缩放 / 平移 ----------

    zoomAt(sx: number, sy: number, factor: number) {
      const z = clamp(this.view.zoom * factor, ZOOM_MIN, ZOOM_MAX)
      const wx = (sx - this.view.x) / this.view.zoom
      const wy = (sy - this.view.y) / this.view.zoom
      this.view.zoom = z
      this.view.x = sx - wx * z
      this.view.y = sy - wy * z
    },

    beginPan(clientX: number, clientY: number) {
      this.interaction = { type: 'pan', lastX: clientX, lastY: clientY }
    },
    movePan(clientX: number, clientY: number) {
      const it = this.interaction
      if (!it || it.type !== 'pan') return
      this.view.x += clientX - it.lastX
      this.view.y += clientY - it.lastY
      it.lastX = clientX
      it.lastY = clientY
    },
    endPan() {
      if (this.interaction?.type === 'pan') this.interaction = null
    },

    /** 折叠/展开某节点的子树：只隐藏/显示，不重排坐标（避免收起时节点位移） */
    toggleCollapse(sceneId: number) {
      const i = this.collapsedSceneIds.indexOf(sceneId)
      if (i >= 0) this.collapsedSceneIds.splice(i, 1)
      else this.collapsedSceneIds.push(sceneId)
    }
  }
})
