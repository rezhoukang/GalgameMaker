<script setup lang="ts">
import { computed, onMounted, onUnmounted, provide, ref, watch } from 'vue'
import { useCanvasStore } from '@/store/canvas'
import { useAppStore } from '@/store/app'
import HtmlBox from './HtmlBox.vue'
import AppIcon from './AppIcon.vue'
import type { HtmlScene, NodeItem } from '@/types'
import {
  BALL_R,
  CURVE_MIN_BEND,
  LINE_W,
  NODE_W,
  PORT_OFFSET,
  ZOOM_MAX,
  ZOOM_MIN
} from '@/canvasLayout'

const store = useCanvasStore()
const app = useAppStore()

const viewportRef = ref<HTMLDivElement | null>(null)

// 屏幕坐标 → 画布世界坐标（无界坐标系）
function screenToWorld(clientX: number, clientY: number) {
  const rect = viewportRef.value!.getBoundingClientRect()
  const sx = clientX - rect.left
  const sy = clientY - rect.top
  return {
    wx: (sx - store.view.x) / store.view.zoom,
    wy: (sy - store.view.y) / store.view.zoom
  }
}
provide('toWorld', (x: number, y: number) => screenToWorld(x, y))

// ---------- 视图：平移（Ctrl 拖拽）/ 缩放（Ctrl 滚轮） ----------
function onViewDown(e: MouseEvent) {
  if (e.button !== 0) return // 右键放行（浏览器右键菜单/手势）
  if (e.ctrlKey) {
    store.beginPan(e.clientX, e.clientY)
    e.preventDefault()
  }
}
function onWheel(e: WheelEvent) {
  if (!e.ctrlKey) return
  const rect = viewportRef.value!.getBoundingClientRect()
  const sx = e.clientX - rect.left
  const sy = e.clientY - rect.top
  const factor = e.deltaY < 0 ? 1.1 : 1 / 1.1
  store.zoomAt(sx, sy, factor)
}

// 用 window 统一收口 mousemove / mouseup，保证松开鼠标必然结束拖拽
function onWindowMove(e: MouseEvent) {
  const it = store.interaction
  if (!it) return
  if (it.type === 'pan') store.movePan(e.clientX, e.clientY)
}
function onWindowUp() {
  const it = store.interaction
  if (!it) return
  if (it.type === 'pan') store.endPan()
}
onMounted(() => {
  window.addEventListener('mousemove', onWindowMove)
  window.addEventListener('mouseup', onWindowUp)
})
onUnmounted(() => {
  window.removeEventListener('mousemove', onWindowMove)
  window.removeEventListener('mouseup', onWindowUp)
})

// 打开画布后把内容居中显示
function fitToContent() {
  if (!viewportRef.value) return
  let minX = Infinity,
    minY = Infinity,
    maxX = -Infinity,
    maxY = -Infinity
  for (const sc of visibleScenes.value) {
    minX = Math.min(minX, sc.x - PORT_OFFSET - 80) // 含左侧端口名空间
    minY = Math.min(minY, sc.y)
    maxX = Math.max(maxX, sc.x + NODE_W + PORT_OFFSET + 80)
    maxY = Math.max(maxY, sc.y + store.sceneHeight(sc.id))
  }
  const vw = viewportRef.value.clientWidth
  const vh = viewportRef.value.clientHeight
  if (!isFinite(minX)) {
    store.view = { x: 0, y: 0, zoom: 1 }
    return
  }
  const pad = 100
  const bw = maxX - minX + pad * 2
  const bh = maxY - minY + pad * 2
  const z = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, Math.min(vw / bw, vh / bh, 1)))
  store.view.zoom = z
  store.view.x = (vw - (maxX - minX) * z) / 2 - minX * z
  store.view.y = (vh - (maxY - minY) * z) / 2 - minY * z
}
// 只在切换画布时适配一次视图；之后的增删/折叠等操作一律不动缩放（缩放由用户 Ctrl 滚轮控制）
watch(
  () => store.canvasId,
  () => setTimeout(fitToContent, 0)
)

// 思维导图 S 曲线：水平出发、水平进入，中间平滑拐弯（无箭头）
function mindPath(x1: number, y1: number, x2: number, y2: number) {
  const dx = x2 - x1
  const bend = Math.max(CURVE_MIN_BEND, dx * 0.5)
  const cx = x1 + bend
  const ex = x2 - BALL_R // 线头贴端口球边缘
  const sx = x1 + BALL_R
  return `M ${sx} ${y1} C ${cx} ${y1}, ${ex - bend} ${y2}, ${ex} ${y2}`
}

// ---------- 跨节点连线 ----------
interface Line {
  id: number
  name: string
  d: string
  color: string
}
// 可见场景：从根 BFS，折叠节点的子树不渲染（边 = 出端口 → 配对入端口所在场景）
const visibleScenes = computed<HtmlScene[]>(() => {
  const parentOf = new Map<number, number>()
  const kidsOf = new Map<number, number[]>()
  for (const n of store.nodes) {
    if (n.entry || n.magic) continue
    const pair = store.nodes.find((x) => x.id !== n.id && x.hash === n.hash)
    if (!pair || pair.sceneId === n.sceneId) continue
    if (!parentOf.has(pair.sceneId)) parentOf.set(pair.sceneId, n.sceneId)
    const list = kidsOf.get(n.sceneId)
    if (list) list.push(pair.sceneId)
    else kidsOf.set(n.sceneId, [pair.sceneId])
  }
  const visible: HtmlScene[] = []
  const seen = new Set<number>()
  const queue = store.scenes.filter((sc) => !parentOf.has(sc.id)).map((sc) => sc.id)
  while (queue.length > 0) {
    const id = queue.shift()!
    if (seen.has(id)) continue
    seen.add(id)
    const sc = store.sceneById(id)
    if (sc) visible.push(sc)
    if (store.isCollapsed(id)) continue
    for (const k of kidsOf.get(id) || []) queue.push(k)
  }
  return visible
})
const visibleIds = computed(() => new Set(visibleScenes.value.map((s) => s.id)))

const lines = computed<Line[]>(() => {
  const out: Line[] = []
  const vis = visibleIds.value
  for (const n of store.nodes) {
    if (n.entry || n.magic) continue // 魔法端口：画布不画线（导出生效）
    const pair = store.nodes.find((x) => x.id !== n.id && x.hash === n.hash)
    if (!pair) continue
    if (!vis.has(n.sceneId) || !vis.has(pair.sceneId)) continue // 折叠隐藏的子树不画线
    const p1 = store.ballWorld(n)
    const p2 = store.ballWorld(pair)
    out.push({
      id: n.id,
      name: n.name,
      d: mindPath(p1.x, p1.y, p2.x, p2.y),
      color: store.nodeColorById(n.id) // 线色跟随出端口分支色
    })
  }
  return out
})

const worldStyle = computed(() => ({
  transform: `translate(${store.view.x}px, ${store.view.y}px) scale(${store.view.zoom})`
}))

const hintText = 'Ctrl 拖拽平移 · Ctrl 滚轮缩放'
</script>

<template>
  <div class="canvas-area">
    <!-- 工具栏：仅画布视图使用 -->
    <div class="toolbar">
      <template v-if="store.isCanvasView">
        <span class="crumb">
          <AppIcon name="file" :size="15" />
          {{ store.canvasName }}
        </span>
        <span class="hint">{{ hintText }}</span>
        <div class="spacer" />
        <span class="zoom">{{ Math.round(store.view.zoom * 100) }}%</span>
      </template>
      <template v-else>
        <span class="hint">在左侧选择一个画布</span>
        <div class="spacer" />
      </template>
    </div>

    <!-- 无界画布视口 -->
    <div
      ref="viewportRef"
      class="viewport"
      :class="{ panning: store.interaction?.type === 'pan' }"
      @mousedown="onViewDown"
      @wheel.prevent="onWheel"
    >
      <!-- 未选择画布：空白 -->
      <div v-if="!store.isCanvasView" class="placeholder">
        <p class="ph-sub">请在左侧选择一个画布</p>
      </div>

      <!-- 画布内容 -->
      <template v-else>
        <div class="world" :style="worldStyle">
          <!-- HTML 框（内含端口；折叠隐藏子树不渲染） -->
          <HtmlBox v-for="scene in visibleScenes" :key="scene.id" :scene="scene" />

          <!-- 跨节点连线 SVG（思维导图 S 曲线，无箭头，线色跟随分支） -->
          <svg class="lines" width="20000" height="20000">
            <path
              v-for="line in lines"
              :key="line.id"
              :d="line.d"
              class="line"
              fill="none"
              :style="{ stroke: line.color, strokeWidth: LINE_W }"
            >
              <title>{{ line.name }}</title>
            </path>
          </svg>
        </div>

      </template>
    </div>
  </div>
</template>

<style scoped>
.canvas-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-width: 0;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 10px 16px;
  background: var(--bg-soft);
  border-bottom: 1px solid var(--border);
}
.crumb {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 700;
  color: var(--text);
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.hint {
  font-size: 12px;
  color: var(--text-faint);
}
.btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: var(--bg);
  color: var(--text);
  font-size: 13px;
  transition: all 0.15s;
  white-space: nowrap;
}
.btn:hover {
  border-color: var(--accent);
  color: var(--accent);
}
.btn.plus {
  color: var(--accent);
  border-color: var(--accent-soft);
}
.spacer {
  flex: 1;
}
.zoom {
  font-size: 12px;
  color: var(--text-faint);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* 无界画布视口 */
.viewport {
  flex: 1;
  overflow: hidden;
  background: var(--bg-soft);
  position: relative;
  background-image: radial-gradient(circle, var(--canvas-dot) 1px, transparent 1px);
  background-size: 28px 28px;
}
.viewport.panning {
  cursor: grabbing;
}
.world {
  position: absolute;
  top: 0;
  left: 0;
  transform-origin: 0 0;
}
.lines {
  position: absolute;
  top: 0;
  left: 0;
  overflow: visible;
  pointer-events: none;
}
.line {
  opacity: 0.85;
  stroke-linecap: round;
}

/* 空白/占位提示 */
.placeholder {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--text-faint);
  text-align: center;
}
.ph-icon {
  opacity: 0.4;
}
.ph-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-dim);
}
.ph-sub {
  font-size: 13px;
  line-height: 2;
  color: var(--text-faint);
}
</style>
