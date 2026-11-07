<script setup lang="ts">
import { vBackdropClose } from '@/backdropClose'
import { ref } from 'vue'
import { useCanvasStore } from '@/store/canvas'
import { useAppStore } from '@/store/app'
import type { StoryVariable } from '@/types'
import AppIcon from './AppIcon.vue'
const emit = defineEmits<{ (e: 'close'): void }>()
const store = useCanvasStore()
const app = useAppStore()
const rows = ref<StoryVariable[]>(store.variables.map(v => ({...v})))
function add() { rows.value.push({id: 0, projectId: store.canvasId!, name: '', initialValue: 0}) }
function used(id: number) { return id !== 0 && store.nodes.some(p => !p.entry && (p.conditions.some(c => c.variableId === id) || p.effects.some(e => e.variableId === id))) }
async function save() {
  const names = rows.value.map(v => v.name.trim())
  if (names.some(n => !n) || new Set(names).size !== names.length) { app.notify('变量名称不能为空且不可重复', 'error'); return }
  if (rows.value.some(v => !Number.isSafeInteger(v.initialValue))) { app.notify('初始值必须是安全整数', 'error'); return }
  if (await store.saveVariables(rows.value.map(v => ({...v, name: v.name.trim()})))) emit('close')
}
</script>
<template>
  <div class="modal-mask variables-mask" v-backdrop-close="() => emit('close')">
    <section class="modal variables-modal" role="dialog" aria-modal="true" aria-labelledby="variables-title">
      <h3 id="variables-title">剧情变量</h3>
      <div class="variable-list">
        <div v-for="(v, i) in rows" :key="i" class="variable-row">
          <label>名称<input v-model="v.name" :aria-label="`变量 ${i + 1} 名称`" placeholder="例如：勇气" :disabled="store.busy" /></label>
          <label>初始值<input v-model.number="v.initialValue" type="number" step="1" :aria-label="`变量 ${i + 1} 初始值`" :disabled="store.busy" /></label>
          <button class="remove" :aria-label="`删除变量 ${i + 1}`" :title="used(v.id) ? '被端口使用，先移除相关条件和效果' : '删除变量'" :disabled="store.busy || used(v.id)" @click="rows.splice(i, 1)"><AppIcon name="trash" :size="16" /></button>
        </div>
        <p v-if="!rows.length" class="empty">暂无变量，添加后即可在端口设置中使用。</p>
      </div>
      <button class="btn add-variable" :disabled="store.busy" @click="add">＋ 添加变量</button>
      <p class="hint footnote">需要清空变量时，在对应端口选择“设值”并填 0。正在被端口使用的变量不可删除。修改剧情配置后重新导出，旧版玩家存档需重新开始。</p>
      <footer class="row"><button class="btn" @click="emit('close')">取消</button><button class="btn primary" :disabled="store.busy" @click="save">保存变量</button></footer>
    </section>
  </div>
</template>
<style scoped>
.variables-mask { z-index: 1100; }
.variables-modal { width: min(720px, calc(100vw - 32px)); height: min(620px, calc(100dvh - 32px)); display: flex; flex-direction: column; overflow: hidden; }
.variables-modal > :not(.variable-list) { flex-shrink: 0; }
.add-variable { align-self: flex-start; }
.hint { color: var(--text-dim); font-size: 12px; line-height: 1.7; }
.variable-list { flex: 1; min-height: 0; overflow-y: auto; margin: 0 0 16px; padding-right: 6px; scrollbar-gutter: stable; }
.variable-row { display: grid; grid-template-columns: minmax(120px,1fr) 110px 32px; align-items: end; gap: 12px; padding: 12px 0; border-bottom: 1px solid var(--border); }
.variable-row label { font-size: 12px; margin: 0; }
.variable-row input { margin-top: 5px; }
.remove { height: 38px; display: grid; place-items: center; background: transparent; border: 0; color: var(--text-dim); }
.remove:disabled { opacity: .35; cursor: not-allowed; }
.empty { padding: 24px 0; color: var(--text-dim); font-size: 13px; }
.footnote { margin-top: 14px; }
@media(max-width:600px) { .variable-row { grid-template-columns: minmax(0,1fr) 80px 28px; gap: 8px; } }
</style>
