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

const name = ref(primary.value.name)
const isShow = ref(primary.value.isShow)

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

// 仅魔法端口允许通过节点哈希修改连接两端
// 输入框预填当前节点哈希，直接改完点「修改」即可
const linkInfo = computed(() => store.linkInfo(props.node.id))
const prevHash = ref(linkInfo.value?.prevScene?.hash ?? '')
const nextHash = ref(linkInfo.value?.nextScene?.hash ?? '')

/** 修改魔法端口所属节点 */
async function doRedirectPrev() {
  if (!primary.value.magic) return
  const h = prevHash.value.trim()
  if (!h) return
  if (await store.redirectNode(props.node.id, h, 'prev')) emit('close')
}

/** 修改魔法端口跳转目标 */
async function doRedirectNext() {
  if (!primary.value.magic) return
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
    isShow: isShow.value
  })
  if (ok) emit('close')
}

async function del() {
  if (await store.deleteNode(props.node.id)) emit('close')
}
</script>

<template>
  <div class="modal-mask" @click.self="emit('close')">
    <div class="modal node-edit" role="dialog" aria-modal="true" aria-labelledby="port-dialog-title">
      <header class="dialog-header">
        <div class="heading-row">
          <h3 id="port-dialog-title"><AppIcon name="file" :size="18" /> 端口设置</h3>
          <button class="close-btn" aria-label="关闭端口设置" @click="emit('close')"><AppIcon name="close" :size="16" /></button>
        </div>
        <p class="hint-line">{{ primary.magic ? '设置选项内容、播放方式和魔法连接目标。' : '设置选项内容和播放方式。' }}</p>
      </header>

      <div class="dialog-body">
        <div class="field">
          <label for="port-option-content">选项内容</label>
          <div class="option-input">
            <input
              id="port-option-content"
              ref="nameRef"
              v-model="name"
              placeholder="内容结束后显示的选项文案"
              @keyup.enter="save"
              @keyup.esc="emit('close')"
            />
            <button
              type="button"
              class="visibility-toggle"
              :class="{ hidden: !isShow }"
              :aria-label="isShow ? '隐藏选项' : '显示选项'"
              :title="isShow ? '隐藏选项' : '显示选项'"
              :aria-pressed="isShow"
              :disabled="store.busy"
              @click="isShow = !isShow"
            >
              <AppIcon :name="isShow ? 'eye' : 'close'" :size="18" />
            </button>
          </div>
        </div>

        <div class="node-info">
          <div class="owner-row">
            <span class="info-label">所属节点</span>
            <span class="owner-name" :title="sceneName"><AppIcon name="file" :size="14" /> {{ sceneName }}</span>
          </div>
          <div class="meta-row"><span class="info-label">节点哈希</span><code>{{ scene?.hash || '—' }}</code></div>
          <div class="meta-row"><span class="info-label">端口哈希</span><code>{{ node.hash }}</code></div>
        </div>

        <section class="connection-section">
          <h4>连接关系</h4>
          <p v-if="primary.magic" class="hint-line">通过节点哈希修改魔法连接的起点或终点。</p>
          <div v-if="primary.magic" class="link-row">
            <div class="link-heading"><span class="link-label">上一节点</span><span class="link-name" :title="linkInfo?.prevScene?.name">{{ stripExt(linkInfo?.prevScene?.name || '?') }}</span></div>
            <div class="link-controls">
              <input v-model="prevHash" aria-label="上一节点哈希" placeholder="粘贴节点哈希" @keyup.enter="doRedirectPrev" />
              <button class="btn" :disabled="store.busy" @click="doRedirectPrev">修改</button>
            </div>
          </div>
          <div v-if="primary.magic" class="link-row">
            <div class="link-heading"><span class="link-label">下一节点</span><span class="link-name" :title="linkInfo?.nextScene?.name">{{ stripExt(linkInfo?.nextScene?.name || '?') }}</span></div>
            <div class="link-controls">
              <input v-model="nextHash" aria-label="下一节点哈希" placeholder="粘贴节点哈希" @keyup.enter="doRedirectNext" />
              <button class="btn" :disabled="store.busy" @click="doRedirectNext">修改</button>
            </div>
          </div>
          <div v-if="pairInfo" class="peer-row">
            <span class="peer-name" :title="pairInfo.peerScene?.name">另一端：{{ stripExt(pairInfo.peerScene?.name || '?') }}</span>
            <span v-if="pairInfo.peer?.magic" class="magic-badge">魔法端口</span>
            <button class="btn" title="跳到配对端口所在节点并高亮" @click="findPair">定位节点</button>
          </div>
          <p v-else class="hint-line missing-target">目标节点不存在</p>
        </section>
      </div>

      <footer class="dialog-footer">
        <button class="btn danger" :disabled="store.busy" @click="del"><AppIcon name="trash" :size="13" /> 删除端口</button>
        <div class="footer-actions">
          <button class="btn" @click="emit('close')">取消</button>
          <button class="btn primary" :disabled="store.busy" @click="save">保存</button>
        </div>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.node-edit {
  width: min(520px, calc(100vw - 32px));
  max-height: calc(100dvh - 32px);
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
}
.dialog-header { padding: 16px 24px 12px; border-bottom: 1px solid var(--border); }
.heading-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 6px; }
.node-edit h3 { margin: 0; }
.close-btn {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  flex: none;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--text-dim);
}
.close-btn:hover { background: var(--bg-soft); color: var(--text); }
.dialog-body { padding: 16px 24px; overflow-y: auto; min-height: 0; display: flex; flex-direction: column; gap: 14px; }
.node-edit .field { margin: 0; }
.node-edit .field > label { font-size: 13px; font-weight: 600; color: var(--text); margin-bottom: 8px; }
.node-edit input { min-width: 0; height: 38px; }
.hint-line { font-size: 12px; color: var(--text-dim); line-height: 1.6; }
.option-input { position: relative; }
.node-edit .option-input input { padding-right: 46px; }
.visibility-toggle {
  position: absolute;
  right: 5px;
  top: 50%;
  transform: translateY(-50%);
  width: 30px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--text-dim);
}
.visibility-toggle:hover, .visibility-toggle:focus-visible { background: var(--accent-soft); color: var(--accent); }
.visibility-toggle:focus-visible { outline: 2px solid var(--accent); outline-offset: 1px; }
.visibility-toggle.hidden { color: var(--text-faint); }
.node-info { display: grid; gap: 6px; padding-bottom: 14px; border-bottom: 1px solid var(--border); }
.owner-row, .meta-row { display: flex; align-items: center; gap: 14px; min-width: 0; }
.info-label { flex: 0 0 58px; font-size: 12px; color: var(--text-dim); }
.owner-name { display: flex; align-items: center; gap: 6px; font-size: 13px; min-width: 0; overflow-wrap: anywhere; }
.owner-name svg { flex: none; }
.meta-row code { font-family: monospace; font-size: 12px; color: var(--text-dim); overflow-wrap: anywhere; min-width: 0; }
.connection-section h4 { font-size: 13px; font-weight: 600; margin-bottom: 4px; }
.connection-section > .hint-line { margin-bottom: 8px; }
.link-row { padding: 8px 12px; border: 1px solid var(--border); border-radius: 10px; margin-top: 8px; }
.link-heading { display: flex; align-items: center; gap: 12px; margin-bottom: 6px; min-width: 0; }
.link-label { flex: none; font-size: 12px; color: var(--text-dim); }
.link-name { font-size: 13px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.link-controls { display: flex; align-items: center; gap: 8px; }
.node-edit .link-controls input { height: 34px; flex: 1; font-family: monospace; font-size: 12px; }
.link-controls .btn { flex: none; height: 34px; }
.peer-row { display: flex; align-items: center; gap: 8px; margin-top: 12px; }
.peer-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; color: var(--text-dim); }
.peer-row .btn { flex: none; }
.magic-badge { flex: none; font-size: 11px; padding: 3px 6px; border-radius: 5px; background: var(--accent-soft); color: var(--accent); }
.missing-target { margin-top: 12px; }
.dialog-header, .dialog-footer { flex: none; }
.dialog-footer { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 12px 24px; border-top: 1px solid var(--border); }
.footer-actions { display: flex; gap: 8px; }
.dialog-footer .btn { min-height: 34px; }
.dialog-footer .danger { color: var(--danger); }
@media (max-width: 480px) {
  .dialog-header, .dialog-body { padding-left: 16px; padding-right: 16px; }
  .dialog-footer { padding: 12px 16px; gap: 8px; }
  .dialog-footer .btn { padding: 6px 10px; }
}
</style>
