// 剧情节点与出端口来自后端；入端口仅在 applySnapshot 中推导。
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import { useAppStore } from './app'
import { useTreeStore } from './tree'
import type { Node, PortItem, CanvasView } from '@/types'
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
import { computeMindLayout } from '@/mindLayout'
import { treeVisibility } from '@/treeVisibility'

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

/** 返回与 nodeId 哈希相同的配对端口（另一端）；没有则返回 null */
function pairOf(nodes: PortItem[], nodeId: number): PortItem | null {
  const n = nodes.find((x) => x.id === nodeId)
  if (!n) return null
  return nodes.find((x) => x.id !== nodeId && x.hash === n.hash) ?? null
}

/** 端口颜色：魔法纯红；出端口按同框出端口顺序取分支调色板；入端口颜色跟随配对出端口 */
function portColor(nodes: PortItem[], nodeId: number): string {
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
  const peers = nodes.filter((n) => n.nodeId === node.nodeId && !n.entry)
  const idx = Math.max(0, peers.findIndex((n) => n.id === nodeId))
  return MIND_COLORS[idx % MIND_COLORS.length]
}

export const useCanvasStore = defineStore('canvas', {
  state: () => ({
    snapshot: null as CanvasView | null,
    past: [] as CanvasView[],
    future: [] as CanvasView[],
    busy: false,
    cacheWarning: false,
    clipboard: null as { sourceId: number; snapshot: CanvasView } | null,

    canvasId: null as number | null,
    canvasName: '',
    scenes: [] as Node[],
    nodes: [] as PortItem[],
    selectedSceneId: null as number | null,
    view: { x: 0, y: 0, zoom: 1 } as ViewState,
    interaction: null as Interaction,
    editingNode: null as PortItem | null,
    editingScene: null as number | null,
    flashSceneId: null as number | null,
    collapsedSceneIds: [] as number[],
    compactSceneIds: [] as number[]
  }),

  getters: {
    treeVisibility: (s) => treeVisibility(s.scenes, s.nodes, s.collapsedSceneIds, s.compactSceneIds),
    canUndo: (s) => !s.busy && s.past.length > 0,
    canRedo: (s) => !s.busy && s.future.length > 0,
    nodeById: (s) => (id: number) => s.nodes.find((n) => n.id === id),
    sceneById: (s) => (id: number) => s.scenes.find((sc) => sc.id === id),

    /** 端口颜色：魔法纯红；出端口按同框出端口顺序取分支调色板；入端口颜色跟随配对出端口 */
    nodeColorById: (s) => (nodeId: number) => portColor(s.nodes, nodeId),
    isCanvasView: (s) => s.canvasId != null,

    foldState: (s) => (nodeId: number) => s.collapsedSceneIds.includes(nodeId) ? 'collapsed' : s.compactSceneIds.includes(nodeId) ? 'compact' : 'expanded',

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
      const toScene = s.scenes.find((sc) => sc.id === toNode.nodeId)
      return { toNode, toScene }
    },

    /** 被谁连入（仅入端口）：配对出端口 + 来源节点 */
    inConns: (s) => (nodeId: number) => {
      const node = s.nodes.find((n) => n.id === nodeId)
      if (!node || !node.entry) return [] // 出端口无「被谁连入」
      const fromNode = pairOf(s.nodes, nodeId)
      if (!fromNode) return []
      const fromScene = s.scenes.find((sc) => sc.id === fromNode.nodeId)
      return [{ fromNode, fromScene }]
    },

    /** 端口配对信息（出/入通用）：同一哈希的另一端 + 所属节点；无配对返回 null */
    pairInfo: (s) => (nodeId: number) => {
      const peer = pairOf(s.nodes, nodeId)
      if (!peer) return null
      const peerScene = s.scenes.find((sc) => sc.id === peer.nodeId)
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
      const prevScene = s.scenes.find((sc) => sc.id === outNode.nodeId)
      const nextScene = s.scenes.find((sc) => sc.id === inNode.nodeId)
      return { prevScene, nextScene }
    },

    /** 端口在画布上的世界坐标：入端口贴节点左边缘、出端口贴右边缘，以固定行距散开（卡片高度不变） */
    ballWorld(): (node: PortItem) => { x: number; y: number } {
      const s = this
      return (node: PortItem) => {
        const scene = s.scenes.find((sc) => sc.id === node.nodeId)
        if (!scene) return { x: 0, y: 0 }
        const mine = s.nodes.filter((n) => n.nodeId === scene.id)
        const isEntry = node.entry
        const col = isEntry ? mine.filter((n) => n.entry) : mine.filter((n) => !n.entry)
        const compact = !isEntry && s.treeVisibility.compactIds.has(scene.id)
        const idx = compact ? 0 : Math.max(0, col.findIndex((n) => n.id === node.id))
        const count = compact ? 1 : col.length
        const x = isEntry ? scene.x - PORT_OFFSET : scene.x + NODE_W + PORT_OFFSET
        const y = scene.y + NODE_H / 2 + (idx - (count - 1) / 2) * PORT_ROW_H // 以卡片中心发散
        return { x, y }
      }
    }
  },

  actions: {
    clear() {
      this.saveCache()
      this.canvasId = null; this.canvasName = ''; this.scenes = []; this.nodes = []
      this.snapshot = null; this.past = []; this.future = []
      this.selectedSceneId = null; this.editingNode = null; this.editingScene = null
      this.collapsedSceneIds = []; this.interaction = null; this.view = { x: 0, y: 0, zoom: 1 }
      this.compactSceneIds = []
    },

    applySnapshot(view: CanvasView) {
      this.snapshot = JSON.parse(JSON.stringify(view))
      this.scenes = view.scenes.map(s => ({...s, x: 0, y: 0}))
      const byHash = new Map(this.scenes.map(s => [s.hash, s.id]))
      this.nodes = view.nodes.flatMap(p => {
        const out = {...p, entry: false}
        const targetId = byHash.get(p.exportHash)
        return targetId == null ? [out] : [out, {...out, id: -p.id, nodeId: targetId, entry: true}]
      })
      if (this.selectedSceneId != null && !this.sceneById(this.selectedSceneId)) this.selectedSceneId = null
      this.collapsedSceneIds = this.collapsedSceneIds.filter(id => this.sceneById(id))
      this.compactSceneIds = this.compactSceneIds.filter(id => this.sceneById(id))
      this.relayout()
    },

    saveCache() {
      if (!this.canvasId || !this.snapshot) return
      try {
        localStorage.setItem(`galgame-history-v1-${this.canvasId}`, JSON.stringify({
          token: this.snapshot.token, past: this.past, future: this.future,
          view: this.view, collapsed: this.collapsedSceneIds, compact: this.compactSceneIds
        }))
      } catch {
        if (!this.cacheWarning) { this.cacheWarning = true; useAppStore().notify('本地缓存空间不足，本次撤回仍可用，刷新后历史可能丢失', 'error') }
      }
    },

    async openCanvas(id: number, name: string) {
      if (this.busy) return
      this.busy = true
      this.saveCache()
      try {
        const view = await api.getCanvas(id)
        this.canvasId = id; this.canvasName = name
        this.past = []; this.future = []; this.collapsedSceneIds = []
        this.compactSceneIds = []
        this.editingNode = null; this.editingScene = null; this.selectedSceneId = null
        this.view = { x: 0, y: 0, zoom: 1 }
        try {
          const cache = JSON.parse(localStorage.getItem(`galgame-history-v1-${id}`) || 'null')
          if (cache?.token === view.token) {
            if (Array.isArray(cache.past)) this.past = cache.past.slice(-50)
            if (Array.isArray(cache.future)) this.future = cache.future.slice(-50)
            if (cache.view && [cache.view.x, cache.view.y, cache.view.zoom].every(Number.isFinite)) this.view = {...cache.view, zoom: clamp(cache.view.zoom, ZOOM_MIN, ZOOM_MAX)}
            if (Array.isArray(cache.collapsed)) this.collapsedSceneIds = cache.collapsed
            if (Array.isArray(cache.compact)) this.compactSceneIds = cache.compact
          }
        } catch { /* Ignore malformed browser cache; server data remains authoritative. */ }
        this.applySnapshot(view)
      } catch (e: any) { useAppStore().handleApiError(e) }
      finally { this.busy = false }
    },

    relayout() {
      const { visibleIds, compactIds } = this.treeVisibility
      const positions = computeMindLayout(this.scenes.filter(s => visibleIds.has(s.id)), this.nodes, compactIds)
      for (const sc of this.scenes) { const p = positions.get(sc.id); if (p) { sc.x = p.x; sc.y = p.y } }
    },

    async edit(action: () => Promise<unknown>, message = ''): Promise<boolean> {
      if (this.busy || this.canvasId == null) return false
      this.busy = true
      const id = this.canvasId
      let before: CanvasView | null = null
      let recorded = false
      try {
        before = await api.getCanvas(id)
        if (before.token !== this.snapshot?.token) { this.past = []; this.future = [] }
        await action()
        const after = await api.getCanvas(id)
        if (before.token !== after.token) { this.past.push(before); this.past = this.past.slice(-50); this.future = []; recorded = true }
        this.applySnapshot(after); this.saveCache()
        await useTreeStore().load()
        if (message) useAppStore().notify(message, 'success')
        return true
      } catch (e: any) {
        useAppStore().handleApiError(e)
        try {
          const after = await api.getCanvas(id)
          if (before && before.token !== after.token && !recorded) {
            this.past.push(before); this.past = this.past.slice(-50); this.future = []
          }
          this.applySnapshot(after); this.saveCache()
        } catch { /* Keep existing view if disconnected. */ }
        return false
      } finally { this.busy = false }
    },

    async travel(redo: boolean) {
      if (this.busy || !this.canvasId || !this.snapshot) return
      const stack = redo ? this.future : this.past
      const target = stack[stack.length - 1]
      if (!target) return
      this.busy = true
      try {
        const current = this.snapshot
        const restored = await api.restoreCanvas(this.canvasId, current.token, target)
        stack.pop(); (redo ? this.past : this.future).push(current)
        this.editingNode = null; this.editingScene = null
        this.applySnapshot(restored); this.saveCache()
        await useTreeStore().load()
        useAppStore().notify(redo ? '已重做' : '已撤回', 'success')
      } catch (e: any) { useAppStore().handleApiError(e) }
      finally { this.busy = false }
    },
    undo() { return this.travel(false) },
    redo() { return this.travel(true) },
    selectScene(id: number) { this.selectedSceneId = id },
    renameScene(id: number, name: string) { return this.edit(() => api.updateScene(id, {name}), '已重命名') },
    async deleteScene(id: number) {
      const sc = this.sceneById(id)
      if (!sc) return false
      if (sc.level === 1) { useAppStore().notify('根节点不可删除', 'error'); return false }
      if (!confirm(`删除「${sc.name}」及其普通子树？指向它们的魔法连接也会移除，可撤回。`)) return false
      return this.edit(() => api.deleteScene(id), '已删除分支，可撤回')
    },
    async refreshCanvas() { if (this.canvasId) this.applySnapshot(await api.getCanvas(this.canvasId)) },
    addNextNode(id: number) { return this.edit(() => api.createNextNode(id)) },
    uploadVideo(id: number, file: File) { return this.edit(() => api.uploadSceneVideo(id, file), '已导入视频') },
    uploadHtml(id: number, file: File) { return this.edit(() => api.uploadSceneHtml(id, file), '已导入 HTML') },
    setFirstScene(id: number) { return this.edit(() => api.setFirstScene(id), '已设为播放起点') },
    copyProperties() {
      if (this.selectedSceneId == null || !this.snapshot) return
      this.clipboard = {sourceId: this.selectedSceneId, snapshot: JSON.parse(JSON.stringify(this.snapshot))}
      useAppStore().notify('已复制节点内容属性', 'success')
    },
    pasteProperties() {
      const clip = this.clipboard, id = this.selectedSceneId
      if (!clip || id == null || clip.snapshot.projectId !== this.canvasId) return
      return this.edit(() => api.copyProperties(id, clip.sourceId, clip.snapshot), '已粘贴节点内容属性')
    },

    /** 定位视图到某个节点并选中（魔法跳跃） */
    jumpToScene(nodeId: number) {
      const sc = this.sceneById(nodeId)
      if (!sc) return
      this.selectedSceneId = nodeId
      this.collapsedSceneIds = []
      this.compactSceneIds = []
      this.relayout()
      this.editingNode = null
      this.editingScene = null
      const cx = sc.x + NODE_W / 2
      const cy = sc.y + this.sceneHeight(nodeId) / 2
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

    addMagicPort(id: number, hash: string) { return this.edit(() => api.createMagicPort(id, hash), '已新增魔法端口') },

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

    renameNode(id: number, name: string) { return this.edit(() => api.updateNode(Math.abs(id), {name}), '已保存') },
    async saveNode(id: number, data: {name: string; isShow?: boolean}) {
      const ok = await this.edit(() => api.updateNode(Math.abs(id), data), '已保存')
      if (ok) this.editingNode = null
      return ok
    },
    async deleteNode(id: number) {
      const p = this.nodeById(id)
      if (!p) return false
      const tip = p.magic ? '删除这条魔法连接？目标节点保留。' : '删除这条普通连接及其目标子树？可撤回。'
      if (!confirm(tip)) return false
      const ok = await this.edit(() => api.deleteNode(Math.abs(id)), '已删除，可撤回')
      if (ok) this.editingNode = null
      return ok
    },
    breakNode(id: number) { return this.edit(() => api.breakNode(Math.abs(id)), '已断开') },
    redirectNode(id: number, hash: string, side: 'prev' | 'next') {
      return this.edit(() => api.redirectNode(Math.abs(id), hash, side), '已修改连接')
    },

    // ---------- 视图：缩放 / 平移 ----------

    zoomAt(sx: number, sy: number, factor: number) {
      const z = clamp(this.view.zoom * factor, ZOOM_MIN, ZOOM_MAX)
      const wx = (sx - this.view.x) / this.view.zoom
      const wy = (sy - this.view.y) / this.view.zoom
      this.view.zoom = z
      this.view.x = sx - wx * z
      this.view.y = sy - wy * z
      this.saveCache()
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
      this.saveCache()
    },

    /** 重排后保持操作节点在屏幕上的位置，缩放不变。 */
    relayoutAt(nodeId: number, before: { x: number; y: number } | undefined) {
      this.relayout()
      const after = this.sceneById(nodeId)
      if (before && after) {
        this.view.x += (before.x - after.x) * this.view.zoom
        this.view.y += (before.y - after.y) * this.view.zoom
      }
      this.saveCache()
    },

    /** 展开 → 紧凑（减号）→ 收起 → 展开；每次按可见内容重排。 */
    toggleCollapse(nodeId: number) {
      const sc = this.sceneById(nodeId)
      const before = sc ? { x: sc.x, y: sc.y } : undefined
      const state = this.foldState(nodeId)
      this.collapsedSceneIds = this.collapsedSceneIds.filter(id => id !== nodeId)
      this.compactSceneIds = this.compactSceneIds.filter(id => id !== nodeId)
      if (state === 'expanded') this.compactSceneIds.push(nodeId)
      else if (state === 'compact') this.collapsedSceneIds.push(nodeId)
      this.relayoutAt(nodeId, before)
    }
  }
})
