<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useCanvasStore } from '@/store/canvas'
import { useAppStore } from '@/store/app'
import BallNode from './BallNode.vue'
import AppIcon from './AppIcon.vue'
import type { HtmlScene, NodeItem } from '@/types'
import { BALL_R, NODE_H, NODE_RADIUS, NODE_W, PORT_OFFSET, PORT_ROW_H } from '@/canvasLayout'

const props = defineProps<{ scene: HtmlScene }>()

const store = useCanvasStore()
const app = useAppStore()

const name = ref(stripExt(props.scene.name))
watch(
  () => props.scene.name,
  (v) => (name.value = stripExt(v))
)

/** 名称去掉 .html 后缀显示（节点名即文件夹名） */
function stripExt(n: string): string {
  return n.replace(/\.html?$/i, '')
}

const height = computed(() => store.sceneHeight(props.scene.id))
const selected = computed(() => store.selectedSceneId === props.scene.id)
const isFolded = computed(() => store.isCollapsed(props.scene.id))

// 出端口列：加号按钮放在即将新增出的那个端口位置（最后一个出端口下一行；无端口时在卡片中心）
const outCol = computed(() =>
  store.nodes.filter((n) => n.sceneId === props.scene.id && !store.nodeIsEntry(n.id))
)
const addBtnTop = computed(() => {
  const n = outCol.value.length
  if (n === 0) return NODE_H / 2 // 无出端口：高度居中
  let y = NODE_H / 2 + ((n - 1) - (n - 1) / 2) * PORT_ROW_H + PORT_ROW_H
  return Math.max(y, NODE_H / 2 + 24) // 有出端口：与居中折叠按钮拉开，不挤
})
// 加号球相对节点的定位（与出端口球同一套公式）
const addPos = computed(() => ({
  left: NODE_W + PORT_OFFSET - BALL_R,
  top: addBtnTop.value - BALL_R
}))

// 节点分支色：跟随入端口（父节点分支色）；根节点用默认蓝
const entry = computed(() =>
  store.nodes.find((n) => n.sceneId === props.scene.id && store.nodeIsEntry(n.id))
)
const accent = computed(() =>
  entry.value ? store.nodeColorById(entry.value.id) : '#5b8def'
)

// 端口球在节点内的相对位置（世界坐标 - 节点左上角）
function nodePos(node: NodeItem) {
  const p = store.ballWorld(node)
  return {
    left: p.x - props.scene.x - BALL_R,
    top: p.y - props.scene.y - BALL_R
  }
}

/** 复制节点哈希 */
async function copyHash() {
  try {
    await navigator.clipboard.writeText(props.scene.hash || '')
    app.notify(`节点哈希已复制：${props.scene.hash || '—'}`, 'success')
  } catch {
    app.notify('复制失败', 'error')
  }
}

/** 折叠/展开子树 */
function onToggleFold() {
  store.toggleCollapse(props.scene.id)
}

function onSelect() {
  store.selectScene(props.scene.id)
}

function onEdit() {
  store.editingScene = props.scene.id
}

async function commitName() {
  let v = name.value.trim()
  if (!v) {
    name.value = stripExt(props.scene.name)
    return
  }
  if (v === stripExt(props.scene.name)) {
    name.value = v
    return
  }
  try {
    await store.renameScene(props.scene.id, v)
  } catch (e: any) {
    app.handleApiError(e)
    name.value = stripExt(props.scene.name)
  }
}
</script>

<template>
  <div
    class="mm-node"
    :class="{ selected, flash: store.flashSceneId === scene.id }"
    :style="{
      left: scene.x + 'px',
      top: scene.y + 'px',
      width: NODE_W + 'px',
      height: height + 'px',
      borderRadius: NODE_RADIUS + 'px',
      borderColor: selected ? '#4f6bf6' : 'var(--border)',
      '--port-offset': PORT_OFFSET + 'px'
    }"
  >
    <!-- 分支色条（左侧，跟随父分支颜色） -->
    <div class="mm-stripe" :style="{ background: accent, borderRadius: NODE_RADIUS + 'px 0 0 ' + NODE_RADIUS + 'px' }" />

    <!-- 标题行 -->
    <div class="mm-title" @click.stop="onSelect">
      <span v-if="scene.first" class="first-badge" title="开始节点：导出后从它开始播放">
        <svg viewBox="0 0 16 16" width="10" height="10" fill="currentColor"><path d="M4 2 L14 8 L4 14 Z" /></svg>
      </span>
      <input
        v-model="name"
        spellcheck="false"
        title="节点名称（可编辑）"
        @mousedown.stop
        @blur="commitName"
        @keyup.enter="commitName"
        @keyup.esc="name = scene.name; ($event.target as HTMLInputElement).blur()"
      />
      <button class="mini hash-copy" title="复制节点哈希" @mousedown.stop @click.stop="copyHash">#</button>
      <button class="mini edit" title="编辑节点（导入 HTML / 视频 / 第一个播放）" @mousedown.stop @click.stop="onEdit">
        <AppIcon name="pen" :size="13" />
      </button>
    </div>

    <!-- 折叠/展开子树：骑在节点顶框上（一半在框内一半在框外），每个节点都有 -->
    <button
      class="fold-btn"
      :class="{ folded: isFolded }"
      :title="isFolded ? '展开子树' : '收起子树'"
      @mousedown.stop.prevent
      @click.stop="onToggleFold"
    >
      <span class="bracket">{{ isFolded ? '❯' : '❮' }}</span>
    </button>

    <!-- 右侧端口容器：加号球 = 当作下一个出端口（与出端口同逻辑），点击新增出端口+下一跳 -->
    <BallNode :add-scene-id="scene.id" :pos="addPos" />

    <!-- 入/出端口球：节点直接子级，绝对定位基准与连线锚点（ballWorld）一致 -->
    <BallNode v-for="ball in store.nodes.filter((n) => n.sceneId === scene.id)" :key="ball.id" :node="ball" :pos="nodePos(ball)" />
  </div>
</template>

<style scoped>
/* 思维导图节点：圆角卡片，无左右端口列区域、无方框化装饰 */
.mm-node {
  position: absolute;
  background: var(--bg);
  border: 1px solid var(--border);
  box-shadow: 0 3px 14px rgba(28, 28, 34, 0.08);
  transition: box-shadow 0.15s, border-color 0.15s;
}
.mm-node:hover:not(:has(.ball.add:hover)) {
  box-shadow: 0 8px 26px rgba(28, 28, 34, 0.14);
}
.mm-node.selected {
  box-shadow: 0 0 0 3px rgba(79, 107, 246, 0.16);
}
/* 寻找高亮：橙色脉冲闪烁 */
.mm-node.flash {
  animation: flash-pulse 0.8s ease-in-out 3;
}
@keyframes flash-pulse {
  0%,
  100% {
    box-shadow: 0 0 0 3px rgba(249, 115, 22, 0.18);
  }
  50% {
    box-shadow: 0 0 0 10px rgba(249, 115, 22, 0.5);
  }
}
/* 分支色条 */
.mm-stripe {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 5px;
  pointer-events: none;
}
.mm-title {
  height: 100%;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px 0 14px;
  cursor: default;
}
.first-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #3b82f6;
  color: #fff;
  flex: none;
  box-shadow: 0 2px 6px rgba(59, 130, 246, 0.4);
}
.mm-title input {
  flex: 1;
  min-width: 0;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
  padding: 3px 6px;
  border-radius: 6px;
  outline: none;
}
.mm-title input:hover {
  border-color: var(--border-light);
}
.mm-title input:focus {
  border-color: var(--accent);
  background: var(--bg);
}
.mini {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  border: 1px solid var(--border-light);
  background: var(--bg);
  color: var(--text-dim);
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  opacity: 0;
  transition: all 0.15s;
}
.mm-node:hover .mini {
  opacity: 1;
}
.mini.edit:hover {
  border-color: var(--accent);
  color: var(--accent);
}
/* 折叠/展开按钮：节点框右侧外部、垂直居中（与右侧端口容器水平错开） */
.fold-btn {
  position: absolute;
  left: calc(100% + 13px);
  top: 50%;
  transform: translate(-50%, -50%);
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 1px solid var(--border);
  background: var(--bg);
  color: var(--text-dim);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 8;
  box-shadow: 0 2px 6px rgba(28, 28, 34, 0.12);
  transition: 0.15s;
}
.fold-btn:hover {
  border-color: var(--accent);
  color: var(--accent);
}
.fold-btn .bracket {
  font-size: 13px;
  line-height: 1;
  font-weight: 700;
  transform: translateY(-1px);
}
/* 右侧端口容器已并入 BallNode（加号球与出端口同逻辑渲染，见 BallNode.add-dot） */
</style>
