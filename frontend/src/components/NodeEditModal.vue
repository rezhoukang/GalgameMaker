<script setup lang="ts">
import { computed, ref, onMounted, nextTick, watch } from 'vue'
import { useCanvasStore } from '@/store/canvas'
import type { PortItem } from '@/types'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ node: PortItem }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const store = useCanvasStore()
const nameRef = ref<HTMLInputElement | null>(null)
const scene = computed(() => store.sceneById(props.node.nodeId))
const sceneName = computed(() => stripExt(store.sceneById(props.node.nodeId)?.name || '（未知）'))
// 配对端口（同一哈希的另一端），出/入通用——配对即相连，方向无所谓
const pairInfo = computed(() => store.pairInfo(props.node.id))
// 一对端口配置共同，以出端口（entry=false）为准
const primary = computed(() => {
  if (!props.node.entry) return props.node
  return pairInfo.value?.peer ?? props.node
})
const isPaired = computed(() => !!pairInfo.value)

const name = ref(primary.value.name)
const showOption = ref(primary.value.showOption)
const kind = ref(primary.value.targetKind === 'html' ? 'html' : 'mp4')

watch(
  () => primary.value?.name,
  (v) => (name.value = v ?? '')
)

/** 名称去掉 .html 后缀显示 */
function stripExt(n: string): string {
  return n.replace(/\.html?$/i, '')
}

/** 跳到配对端口所在节点并高亮 */
function findPair() {
  const sc = pairInfo.value?.peerScene
  if (sc?.hash) store.highlightSceneByHash(sc.hash)
  emit('close')
}

// 连接关系（节点—端口—节点）：上一节点/下一节点，通过节点哈希修改
// 输入框预填当前节点哈希，直接改完点「修改」即可
const linkInfo = computed(() => store.linkInfo(props.node.id))
const prevHash = ref(linkInfo.value?.prevScene?.hash ?? '')
const nextHash = ref(linkInfo.value?.nextScene?.hash ?? '')

/** 通过节点哈希修改上一节点（移动出端口） */
async function doRedirectPrev() {
  const h = prevHash.value.trim()
  if (!h) return
  if (await store.redirectNode(props.node.id, h, 'prev')) emit('close')
}

/** 通过节点哈希修改下一节点（移动入端口） */
async function doRedirectNext() {
  const h = nextHash.value.trim()
  if (!h) return
  if (await store.redirectNode(props.node.id, h, 'next')) emit('close')
}

onMounted(() => {
  nextTick(() => nameRef.value?.focus())
})

async function save() {
  const p = primary.value
  const ok = await store.saveNode(p.id, {
    name: name.value.trim() || p.name,
    targetKind: kind.value,
    showOption: showOption.value
  })
  if (ok) emit('close')
}

async function del() {
  if (await store.deleteNode(props.node.id)) emit('close')
}
</script>

<template>
  <div class="modal-mask" @click.self="emit('close')">
    <div class="modal node-edit">
      <h3><AppIcon name="file" :size="17" /> 端口 <span class="hash">#{{ node.hash }}</span></h3>

      <div v-if="isPaired" class="hint-line">
        这里编辑同一条连接的选项、目标和播放方式。
      </div>

      <div class="field">
        <label>选项内容</label>
        <input
          ref="nameRef"
          v-model="name"
          placeholder="视频结束后显示的选项文案"
          @keyup.enter="save"
          @keyup.esc="emit('close')"
        />
      </div>

      <div class="field">
        <label>所属节点</label>
        <div class="file-row">
          <AppIcon name="file" :size="15" />
          <span class="fname">{{ sceneName }}</span>
        </div>
      </div>

      <div class="field">
        <label><input v-model="showOption" type="checkbox" /> 视频结束后显示此选项</label>
        <p class="hint-line">取消勾选表示自动继续；该节点必须恰好有一个出口。</p>
      </div>

      <div class="field">
        <label>跳转目标类型（一对端口共享，导出时跳向目标节点的资源）</label>
        <div class="kind-row">
          <label class="kind-opt">
            <input v-model="kind" type="radio" value="mp4" />
            🎬 视频（MP4）
          </label>
          <label class="kind-opt">
            <input v-model="kind" type="radio" value="html" />
            📄 页面（HTML）
          </label>
        </div>
      </div>

      <div class="field">
        <label>节点哈希：{{ scene?.hash || '—' }} · 端点哈希：{{ node.hash }}</label>
      </div>

      <div class="field">
        <label>连接另一端</label>
        <div v-if="pairInfo" class="out-info">
          <span class="out-name">{{ pairInfo.peer?.name || '?' }}</span>
          <span class="out-scene">↔ {{ stripExt(pairInfo.peerScene?.name || '?') }}</span>
          <span v-if="pairInfo.peer?.magic" class="magic-badge">魔法端口（画布不画线）</span>
          <button class="btn" title="跳到配对端口所在节点并高亮" @click="findPair">寻找</button>
        </div>
        <div v-else class="out-empty">目标节点不存在</div>
      </div>

      <div class="field">
        <label>连接关系（节点—端口—节点，通过节点哈希修改）</label>
        <div class="link-row">
          <span class="link-label">上一节点</span>
          <span class="link-name">{{ stripExt(linkInfo?.prevScene?.name || '?') }}</span>
          <input v-model="prevHash" placeholder="粘贴节点哈希" @keyup.enter="doRedirectPrev" />
          <button class="btn" @click="doRedirectPrev">修改</button>
        </div>
        <div class="link-row">
          <span class="link-label">下一节点</span>
          <span class="link-name">{{ stripExt(linkInfo?.nextScene?.name || '?') }}</span>
          <input v-model="nextHash" placeholder="粘贴节点哈希" @keyup.enter="doRedirectNext" />
          <button class="btn" @click="doRedirectNext">修改</button>
        </div>
      </div>

      <div class="row">
        <button class="btn danger" @click="del"><AppIcon name="trash" :size="13" /> 删除端口</button>
        <div class="spacer" />
        <button class="btn" @click="emit('close')">取消</button>
        <button class="btn primary" :disabled="store.busy" @click="save">保存</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hash {
  color: var(--text-faint);
  font-size: 12px;
  font-family: monospace;
}
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
.fname {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.hint-line {
  font-size: 12px;
  color: var(--text-dim);
  line-height: 1.6;
}
.hint-line strong {
  color: var(--accent);
}
.kind-row {
  display: flex;
  gap: 10px;
}
.kind-opt {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  padding: 9px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  color: var(--text);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}
.kind-opt:hover {
  border-color: var(--accent);
}
.kind-opt input {
  accent-color: var(--accent);
}
.field textarea {
  width: 100%;
  box-sizing: border-box;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  color: var(--text);
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
  font-family: inherit;
}
.field textarea:focus {
  outline: none;
  border-color: var(--accent);
}
/* 连接关系行：上一/下一节点 + 哈希修改 */
.link-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  margin-bottom: 6px;
}
.link-label {
  font-size: 12px;
  color: var(--text-dim);
  flex-shrink: 0;
  width: 56px;
}
.link-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  min-width: 72px;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.link-row input {
  flex: 1;
  min-width: 0;
  padding: 6px 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg);
  color: var(--text);
  font-size: 12px;
  font-family: monospace;
}
.link-row input:focus {
  outline: none;
  border-color: var(--accent);
}
.out-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
}
.out-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.out-scene {
  font-size: 12px;
  color: var(--text-dim);
}
.magic-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 6px;
  background: rgba(139, 92, 246, 0.15);
  color: #8b5cf6;
  font-weight: 600;
}
.in-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.in-item {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
}
.out-empty {
  font-size: 12px;
  color: var(--text-faint);
}
.row .spacer {
  flex: 1;
}
</style>

