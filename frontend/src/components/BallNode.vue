<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useCanvasStore } from '@/store/canvas'
import type { NodeItem } from '@/types'

const props = defineProps<{
  node?: NodeItem
  pos: { left: number; top: number } // 相对节点的定位（球心对齐 ballWorld）
  addSceneId?: number | null // 非空：渲染为「新增出端口」十字球（当作一个出端口）
}>()

const store = useCanvasStore()
const isAdd = computed(() => props.addSceneId != null)
const color = computed(() => (props.node ? store.nodeColorById(props.node.id) : '#4f6bf6'))
const isMagic = computed(() => !!props.node && props.node.magic)
const side = computed(() =>
  props.node && store.nodeIsEntry(props.node.id) ? 'in' : 'out'
)
const dotStyle = computed(() =>
  isMagic.value
    ? {
        background: color.value,
        boxShadow: `0 0 0 1px ${color.value}, 0 0 10px ${color.value}, 0 0 22px ${color.value}`
      }
    : {
        background: color.value,
        boxShadow: `0 0 0 1px ${color.value}, 0 2px 6px rgba(0,0,0,.25)`
      }
)

// 右键菜单（仅加号球）：新增魔法端口 → 自定义弹窗输入目标哈希
const menu = ref(false)
const magicOpen = ref(false)
const magicHash = ref('')
const menuPos = ref({ x: 0, y: 0 })
function onCtx(e: MouseEvent) {
  if (!isAdd.value) return
  e.preventDefault()
  e.stopPropagation()
  menuPos.value = { x: e.clientX, y: e.clientY }
  menu.value = true
}
function closeMenu() {
  menu.value = false
}
function openAddMagic() {
  closeMenu()
  magicHash.value = ''
  magicOpen.value = true
}
async function doAddMagic() {
  const h = magicHash.value.trim()
  if (!h) return
  magicOpen.value = false
  if (props.addSceneId == null) return
  await store.addMagicPort(props.addSceneId, h)
}
onMounted(() => document.addEventListener('mousedown', closeMenu))
onUnmounted(() => document.removeEventListener('mousedown', closeMenu))

function onClick() {
  if (isAdd.value && props.addSceneId != null) {
    store.addNextNode(props.addSceneId)
    return
  }
  if (props.node) store.editingNode = props.node
}
</script>

<template>
  <div
    class="ball"
    :class="[side, { add: isAdd }]"
    :style="{ left: pos.left + 'px', top: pos.top + 'px' }"
    :title="isAdd ? '左键：新增出端口 + 下一跳节点 · 右键：新增魔法端口' : '点击编辑端口（改名 / 魔法连接 / 结局内容）'"
    @mousedown.stop.prevent
    @click.stop="onClick"
    @contextmenu.stop.prevent="onCtx"
  >
    <span v-if="isAdd" class="add-plus">
      <svg viewBox="0 0 24 24" width="30" height="30" fill="currentColor"><path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6z" /></svg>
    </span>
    <span v-else class="dot" :class="{ magic: isMagic }" :style="dotStyle" />
    <span v-if="node" class="name" :class="side">{{ node.name }}</span>
  </div>

  <!-- 右键菜单（加号球）：新增魔法端口 -->
  <Teleport to="body">
    <div
      v-if="menu"
      class="ctx-menu"
      :style="{ left: menuPos.x + 'px', top: menuPos.y + 'px' }"
      @mousedown.stop
      @click.stop
      @contextmenu.prevent.stop
    >
      <button class="ctx-item" @click="openAddMagic">新增魔法端口</button>
    </div>

    <!-- 魔法端口弹窗：输入目标节点哈希 -->
    <div v-if="magicOpen" class="modal-mask" @click.self="magicOpen = false">
      <div class="modal">
        <h3>新增魔法端口</h3>
        <div class="field">
          <label>粘贴目标节点哈希（出端口与目标入端口对应、不连线，导出生效）</label>
          <input
            v-model="magicHash"
            placeholder="目标节点哈希（节点标题 # 可复制）"
            autofocus
            @keyup.enter="doAddMagic"
          />
        </div>
        <div class="row">
          <button class="btn" @click="magicOpen = false">取消</button>
          <button class="btn primary" @click="doAddMagic">连接</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
/* 端口球：由节点绝对定位（球心与 ballWorld 一致，连线锚点精确） */
.ball {
  position: absolute;
  width: 12px;
  height: 12px;
  cursor: pointer;
  user-select: none;
  z-index: 5;
}
/* 扩大点击热区（视觉不变，便于点中） */
.ball::after {
  content: '';
  position: absolute;
  inset: -8px;
  z-index: 0;
  pointer-events: auto;
}
.dot {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid var(--ball-ring);
  z-index: 1;
}
/* 新增出端口十字球：直接显示十字（无圆点背景、无圆圈），hover 高亮放大 */
.add-plus {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  color: var(--accent);
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 0;
  transition: transform 0.15s;
}
.ball.add:hover .add-plus {
  transform: translate(-50%, -50%) scale(1.3);
}
.ball.add::after {
  width: 18px;
  height: 18px;
}
/* 右键菜单 */
.ctx-menu {
  position: fixed;
  z-index: 9999;
  min-width: 140px;
  background: var(--bg);
  border: 1px solid var(--border-light);
  border-radius: 10px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.35);
  padding: 6px;
}
.ctx-item {
  display: block;
  width: 100%;
  text-align: left;
  padding: 8px 12px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text);
  font-size: 13px;
  cursor: pointer;
}
.ctx-item:hover {
  background: var(--accent-soft);
  color: var(--accent);
}
/* 魔法端口：纯红实心（外圈同色、无白边）+ 荧光增强 */
.dot.magic {
  border-color: #ef4444;
}
.ball:hover .dot.magic {
  box-shadow: 0 0 0 1px #ef4444, 0 0 14px #ef4444, 0 0 30px rgba(239, 68, 68, 0.7) !important;
}
.name {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  pointer-events: none;
  line-height: 1.2;
  padding: 2px 8px;
  border-radius: 7px;
  background: var(--ball-name-bg);
  border: 1px solid var(--border-light);
  box-shadow: 0 2px 10px rgba(28, 28, 34, 0.28);
  opacity: 0; /* 默认隐藏，hover 时显示，避免遮挡/被线条干扰 */
  transition: opacity 0.12s ease;
  z-index: 10; /* 浮在连线上方，名字不被线条盖住 */
}
/* 出端口（右侧）：名字在球右侧 */
.ball.out .name {
  left: 14px;
}
/* 入端口（左侧）：名字在球左侧，向右对齐 */
.ball.in .name {
  right: 14px;
  text-align: right;
}
/* hover 显示端口名：加背景更清晰（颜色保持稳定，不做强调色切换，避免闪烁） */
.ball:hover .name {
  opacity: 1;
}
</style>
