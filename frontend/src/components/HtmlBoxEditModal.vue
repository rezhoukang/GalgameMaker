<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useCanvasStore } from '@/store/canvas'
import { useAppStore } from '@/store/app'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ nodeId: number }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const store = useCanvasStore()
const app = useAppStore()

const scene = computed(() => store.sceneById(props.nodeId))
const outPortCount = computed(() =>
  store.nodes.filter(p => p.nodeId === props.nodeId && !store.nodeIsEntry(p.id)).length
)
const isFirst = computed(() => store.sceneById(props.nodeId)?.first ?? false)
const name = ref(stripExt(scene.value?.name ?? ''))
const nameRef = ref<HTMLInputElement | null>(null)
const videoInput = ref<HTMLInputElement | null>(null)
const htmlInput = ref<HTMLInputElement | null>(null)

/** 名称去掉 .html 后缀显示（节点名即文件夹名） */
function stripExt(n: string): string {
  return n.replace(/\.html?$/i, '')
}

watch(
  () => scene.value?.name,
  (v) => (name.value = stripExt(v ?? ''))
)

onMounted(() => nextTick(() => nameRef.value?.focus()))

async function save() {
  const sc = scene.value
  if (!sc) return
  const v = name.value.trim()
  if (v && v !== stripExt(sc.name)) {
    try {
      if (!await store.renameScene(sc.id, v)) return
    } catch (e: any) {
      app.handleApiError(e)
    }
  }
  emit('close')
}

async function onVideoPick(e: Event) {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  input.value = ''
  const sc = scene.value
  if (!f || !sc) return
  try {
    await store.uploadVideo(sc.id, f)
  } catch (err: any) {
    app.handleApiError(err)
  }
}

async function onHtmlPick(e: Event) {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  input.value = ''
  const sc = scene.value
  if (!f || !sc) return
  try {
    await store.uploadHtml(sc.id, f)
  } catch (err: any) {
    app.handleApiError(err)
  }
}

async function del() {
  const sc = scene.value
  if (sc && await store.deleteScene(sc.id)) emit('close')
}
</script>

<template>
  <div class="modal-mask" @click.self="emit('close')">
    <div class="modal html-edit">
      <h3><AppIcon name="file" :size="17" /> 节点属性</h3>

      <div class="first-wrap">
        <button
          type="button"
          class="first-btn"
          :class="{ active: isFirst }"
          @click="scene && store.setFirstScene(scene.id)"
        >
          <span class="play-icon">
            <svg viewBox="0 0 16 16" width="15" height="15" fill="currentColor"><path d="M4 2 L14 8 L4 14 Z" /></svg>
          </span>
          <span>{{ isFirst ? '第一个播放' : '设为第一个播放' }}</span>
        </button>
        <div class="first-tip">
          每个工程恰好有一个播放起点。根节点不可删除，普通分支按层级排列。
        </div>
      </div>

      <div class="field">
        <label>节点名称</label>
        <input
          ref="nameRef"
          v-model="name"
          placeholder="节点名称"
          @keyup.enter="save"
          @keyup.esc="emit('close')"
        />
      </div>

      <div class="field">
        <label>视频（播放结束后进入选项或自动继续）</label>
        <div class="video-row">
          <input ref="videoInput" type="file" accept=".mp4" style="display:none" @change="onVideoPick" />
          <button class="btn" @click="videoInput?.click()"><AppIcon name="upload" :size="14" /> 插入 MP4</button>
          <span class="video-name">{{ scene?.video ? '已导入 MP4' : '未插入视频' }}</span>
        </div>
      </div>

      <div class="field">
        <label>页面（可选，放到节点文件夹；端口跳转类型选「页面」时使用）</label>
        <div class="video-row">
          <input ref="htmlInput" type="file" accept=".html,.htm" style="display:none" @change="onHtmlPick" />
          <button class="btn" @click="htmlInput?.click()"><AppIcon name="upload" :size="14" /> 导入 HTML</button>
          <span class="video-name">{{ scene?.htmlFile ? '已导入 HTML' : '未导入 HTML' }}</span>
        </div>
      </div>

      <div class="field">
        <label>所属工程 · 层级 {{ scene?.level }} · 出端口 {{ outPortCount }} 个</label>
        <div class="file-row">
          <AppIcon name="file" :size="15" />
          <span class="fname">{{ store.canvasName }}</span>
        </div>
      </div>

      <div class="row">
        <button class="btn danger" :disabled="scene?.isRoot || store.busy" @click="del"><AppIcon name="trash" :size="13" /> {{ scene?.isRoot ? '根节点不可删除' : '删除节点' }}</button>
        <div class="spacer" />
        <button class="btn" @click="emit('close')">取消</button>
        <button class="btn primary" :disabled="store.busy" @click="save">保存</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.file-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  color: var(--text-dim);
  font-size: 13px;
}
.video-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.video-name {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fname {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 18px;
}
.row .spacer {
  flex: 1;
}
.first-wrap {
  position: relative;
  display: inline-block;
  margin-bottom: 16px;
}
.first-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 9px 16px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  color: var(--text-dim);
  font-size: 13px;
  cursor: pointer;
  user-select: none;
  transition: 0.15s;
}
.first-btn:hover {
  border-color: #3b82f6;
  color: #3b82f6;
  background: rgba(59, 130, 246, 0.08);
}
.first-btn .play-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.first-btn.active {
  border-color: #3b82f6;
  background: #3b82f6;
  color: #fff;
  box-shadow: 0 4px 14px rgba(59, 130, 246, 0.35);
}
.first-tip {
  position: absolute;
  left: 0;
  top: calc(100% + 8px);
  z-index: 40;
  width: max-content;
  max-width: 300px;
  padding: 9px 12px;
  background: rgba(18, 20, 28, 0.95);
  color: #fff;
  font-size: 12px;
  line-height: 1.55;
  border-radius: 8px;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.35);
  opacity: 0;
  pointer-events: none;
  transform: translateY(-4px);
  transition: 0.15s;
}
.first-wrap:hover .first-tip {
  opacity: 1;
  transform: translateY(0);
}
</style>
