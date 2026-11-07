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
const outPorts = computed(() => store.nodes.filter(p => p.nodeId === props.nodeId && !p.entry))
const visibility = ref<Record<number, boolean>>({})
const optionNames = ref<Record<number, string>>({})
watch(outPorts, ports => {
  visibility.value = Object.fromEntries(ports.map(p => [p.id, visibility.value[p.id] ?? p.isShow]))
  optionNames.value = Object.fromEntries(ports.map(p => [p.id, optionNames.value[p.id] ?? p.name]))
}, { immediate: true })
const shownCount = computed(() => outPorts.value.filter(p => visibility.value[p.id]).length)
function setAllVisible(value: boolean) {
  for (const port of outPorts.value) visibility.value[port.id] = value
}
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
  if (outPorts.value.some(p => !optionNames.value[p.id]?.trim())) {
    app.notify('选项内容不能为空', 'error')
    return
  }
  const ports = outPorts.value.filter(p => visibility.value[p.id] !== p.isShow || optionNames.value[p.id].trim() !== p.name)
    .map(p => ({ id: p.id, name: optionNames.value[p.id].trim(), isShow: visibility.value[p.id] }))
  if ((v && v !== stripExt(sc.name)) || ports.length) {
    if (!await store.saveSceneProperties(sc.id, v && v !== stripExt(sc.name) ? v : sc.name, ports)) return
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
    <div class="modal html-edit" role="dialog" aria-modal="true" aria-labelledby="scene-dialog-title">
      <h3 id="scene-dialog-title"><AppIcon name="file" :size="17" /> 节点属性</h3>
      <div class="properties-columns">
        <section class="node-properties">

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
            <label>节点内容</label>
            <div class="video-row">
              <input ref="videoInput" type="file" accept=".mp4" style="display:none" @change="onVideoPick" />
              <input ref="htmlInput" type="file" accept=".html,.htm" style="display:none" @change="onHtmlPick" />
              <button class="btn" :disabled="store.busy" @click="videoInput?.click()"><AppIcon name="upload" :size="14" /> 导入 MP4</button>
              <button class="btn" :disabled="store.busy" @click="htmlInput?.click()"><AppIcon name="upload" :size="14" /> 导入 HTML</button>
              <span class="video-name">{{ scene?.contentPath ? '已导入内容' : '未导入内容' }}</span>
            </div>
            <p class="content-tip">重新导入会替换当前内容，可撤回。视频结束后显示选项；纯页面点击“继续”。</p>
            <p class="content-tip">HTML 请使用资源已内嵌的单文件，或引用在线资源。</p>
          </div>

          <div class="field">
            <label>层级 {{ scene?.level }} · 出端口 {{ outPorts.length }} 个</label>
          </div>
        </section>
        <section class="ports-panel" aria-labelledby="ports-panel-title">
          <header class="ports-header">
            <h4 id="ports-panel-title">全部出端口 <span>{{ outPorts.length }}</span></h4>
            <span class="ports-summary">显示 {{ shownCount }} / {{ outPorts.length }}</span>
          </header>
          <div class="ports-actions">
            <button type="button" class="btn" :disabled="store.busy || shownCount === outPorts.length" @click="setAllVisible(true)">全部显示</button>
            <button type="button" class="btn" :disabled="store.busy || shownCount === 0" @click="setAllVisible(false)">全部隐藏</button>
          </div>
          <div class="ports-list" aria-label="出端口列表">
            <div v-for="(port, index) in outPorts" :key="port.id" class="port-row" :class="{ 'option-hidden': !visibility[port.id] }">
              <span class="port-index">{{ index + 1 }}</span>
              <div class="port-content">
                <input v-model="optionNames[port.id]" class="port-option" :aria-label="`选项 ${index + 1} 内容`"
                  placeholder="选项内容" :disabled="store.busy" @keyup.enter="save" />
                <span v-if="port.magic" class="port-kind">魔法</span>
              </div>
              <button type="button" class="visibility-toggle" :class="{ hidden: !visibility[port.id] }"
                :aria-label="`${visibility[port.id] ? '隐藏' : '显示'}选项 ${index + 1}`"
                :title="visibility[port.id] ? '隐藏选项' : '显示选项'" :aria-pressed="visibility[port.id]"
                :disabled="store.busy" @click="visibility[port.id] = !visibility[port.id]">
                <AppIcon :name="visibility[port.id] ? 'eye' : 'eye-off'" :size="18" />
              </button>
            </div>
            <p v-if="!outPorts.length" class="ports-empty">暂无出端口</p>
          </div>
        </section>
      </div>

      <div class="row">
        <button class="btn danger" :disabled="(scene?.level === 1) || store.busy" @click="del"><AppIcon name="trash" :size="13" /> {{ (scene?.level === 1) ? '根节点不可删除' : '删除节点' }}</button>
        <div class="spacer" />
        <button class="btn" @click="emit('close')">取消</button>
        <button class="btn primary" :disabled="store.busy" @click="save">保存</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.html-edit { width: min(900px, calc(100vw - 32px)); max-height: calc(100dvh - 32px); overflow-y: auto; }
.properties-columns { display: grid; grid-template-columns: minmax(0, 1.15fr) minmax(0, 1fr); gap: 24px; }
.node-properties { min-width: 0; }
.ports-panel { height: 380px; min-width: 0; display: flex; flex-direction: column; border: 1px solid var(--border); border-radius: 12px; overflow: hidden; }
.ports-header { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 14px 16px 10px; }
.ports-header h4 { font-size: 14px; margin: 0; }
.ports-header h4 span { color: var(--text-dim); margin-left: 5px; font-weight: 400; }
.ports-summary, .port-kind { color: var(--text-dim); font-size: 12px; }
.ports-actions { display: flex; gap: 8px; padding: 0 16px 12px; border-bottom: 1px solid var(--border); }
.ports-actions .btn { padding: 5px 10px; font-size: 12px; }
.ports-list { flex: 1; min-height: 0; overflow-y: auto; overscroll-behavior: contain; padding: 4px 12px; }
.port-row { display: flex; align-items: center; gap: 10px; padding: 10px 4px; border-bottom: 1px solid var(--border-light); }
.port-row:last-child { border-bottom: 0; }
.port-index { width: 20px; flex: none; font-size: 12px; color: var(--text-dim); text-align: center; }
.port-content { flex: 1; min-width: 0; display: flex; align-items: center; gap: 8px; }
.port-content .port-option { flex: 1; width: 0; min-width: 0; font-size: 13px; line-height: 1.5; padding: 5px 8px; }
.port-kind { flex: none; white-space: nowrap; }
.option-hidden .port-option { color: var(--text-dim); }
.visibility-toggle { flex: none; width: 30px; height: 30px; display: grid; place-items: center; border: 0; border-radius: 6px; background: transparent; color: var(--text-dim); }
.visibility-toggle:hover, .visibility-toggle:focus-visible { background: var(--accent-soft); color: var(--accent); }
.visibility-toggle:focus-visible { outline: 2px solid var(--accent); outline-offset: 1px; }
.visibility-toggle.hidden { color: var(--text-faint); }
.ports-empty { text-align: center; color: var(--text-dim); font-size: 13px; padding: 32px 0; }
@media (max-width: 700px) { .properties-columns { grid-template-columns: minmax(0, 1fr); gap: 12px; } .ports-panel { height: 300px; } .video-row { flex-wrap: wrap; } }
.content-tip { font-size: 12px; color: var(--text-dim); line-height: 1.6; margin-top: 8px; }
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
