<script setup lang="ts">
import { computed, ref, onMounted, nextTick, watch } from 'vue'
import { useCanvasStore } from '@/store/canvas'
import type { PortItem } from '@/types'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ node: PortItem }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const store = useCanvasStore()
const nameRef = ref<HTMLInputElement | null>(null)
// 配对端口（同一哈希的另一端），出/入通用——配对即相连，方向无所谓
const pairInfo = computed(() => store.pairInfo(props.node.id))
// 一对端口配置共同，以出端口（entry=false）为准
const primary = computed(() => {
  if (!props.node.entry) return props.node
  return pairInfo.value?.peer ?? props.node
})

// 从出端口方向展示所属节点和下一端点，点击入端口时也保持一致。
const sourceScene = computed(() => store.sceneById(primary.value.nodeId))
const targetScene = computed(() => store.scenes.find(sc => sc.hash === primary.value.exportHash))

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

/** 定位连接两端的节点，并关闭设置弹窗。 */
function locateNode(hash?: string) {
  if (hash && store.highlightSceneByHash(hash)) emit('close')
}

// 魔法端口保留连接修改能力，普通端口只展示节点信息。
const prevHash = ref(sourceScene.value?.hash ?? '')
const nextHash = ref(primary.value.exportHash ?? '')

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

        <section class="endpoint-card" aria-labelledby="source-node-title">
          <div class="endpoint-heading">
            <div class="endpoint-details">
              <h4 id="source-node-title">所属节点</h4>
              <div class="endpoint-name" :title="sourceScene?.name"><AppIcon name="file" :size="14" /><span>{{ stripExt(sourceScene?.name || '节点不存在') }}</span></div>
            </div>
            <button class="btn" aria-label="定位所属节点" :disabled="!sourceScene" @click="locateNode(sourceScene?.hash)">定位</button>
          </div>
          <div class="meta-row"><span class="info-label">节点哈希</span><code>{{ sourceScene?.hash || '—' }}</code></div>
          <div v-if="primary.magic" class="link-controls">
            <input v-model="prevHash" aria-label="所属节点哈希" placeholder="粘贴节点哈希" @keyup.enter="doRedirectPrev" />
            <button class="btn" :disabled="store.busy" @click="doRedirectPrev">修改</button>
          </div>
        </section>

        <section class="endpoint-card" aria-labelledby="target-node-title">
          <div class="endpoint-heading">
            <div class="endpoint-details">
              <h4 id="target-node-title">下一端点 <span v-if="primary.magic" class="magic-badge">魔法连接</span></h4>
              <div class="endpoint-name" :title="targetScene?.name"><AppIcon name="file" :size="14" /><span>{{ stripExt(targetScene?.name || '节点不存在') }}</span></div>
            </div>
            <button class="btn" aria-label="定位下一端点" :disabled="!targetScene" @click="locateNode(targetScene?.hash)">定位</button>
          </div>
          <div class="meta-row"><span class="info-label">节点哈希</span><code>{{ primary.exportHash || '—' }}</code></div>
          <div v-if="primary.magic" class="link-controls">
            <input v-model="nextHash" aria-label="下一端点哈希" placeholder="粘贴节点哈希" @keyup.enter="doRedirectNext" />
            <button class="btn" :disabled="store.busy" @click="doRedirectNext">修改</button>
          </div>
        </section>

        <div class="meta-row port-meta"><span class="info-label">端口哈希</span><code>{{ primary.hash }}</code></div>
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
.endpoint-card { padding: 14px; border: 1px solid var(--border); border-radius: 10px; }
.endpoint-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 12px; }
.endpoint-details { flex: 1; min-width: 0; }
.endpoint-details h4 { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; font-size: 12px; font-weight: 400; color: var(--text-dim); }
.endpoint-name { display: flex; align-items: center; gap: 6px; min-width: 0; font-size: 13px; font-weight: 600; }
.endpoint-name svg, .endpoint-heading > .btn { flex: none; }
.endpoint-name span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.meta-row { display: flex; align-items: baseline; gap: 14px; min-width: 0; }
.info-label { flex: 0 0 58px; font-size: 12px; color: var(--text-dim); }
.meta-row code { font-family: monospace; font-size: 12px; color: var(--text-dim); overflow-wrap: anywhere; min-width: 0; user-select: text; }
.port-meta { padding: 0 2px; }
.link-controls { display: flex; align-items: center; gap: 8px; margin-top: 12px; }
.node-edit .link-controls input { height: 34px; flex: 1; font-family: monospace; font-size: 12px; }
.link-controls .btn { flex: none; height: 34px; }
.magic-badge { font-size: 11px; padding: 3px 6px; border-radius: 5px; background: var(--accent-soft); color: var(--accent); }
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
