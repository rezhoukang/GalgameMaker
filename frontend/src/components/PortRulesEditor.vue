<script setup lang="ts">
import { computed } from 'vue'
import { useCanvasStore } from '@/store/canvas'
import type { PortRules } from '@/types'
import AppIcon from './AppIcon.vue'
const rules = defineModel<PortRules>({required: true})
const store = useCanvasStore()
const operators = ['>=','<=','==','!=','>','<']
const unused = computed(() => store.variables.filter(v => !rules.value.effects.some(e => e.variableId === v.id)))
function addCondition() { if (store.variables.length) rules.value.conditions.push({variableId: store.variables[0].id, operator: '>=', value: 1}) }
function addEffect() { if (unused.value.length) rules.value.effects.push({variableId: unused.value[0].id, set: 0, delta: 1}) }
</script>
<template>
  <section class="rules-section">
    <div class="rules-heading"><h4>出现条件</h4><button class="text-btn" @click="store.variablesOpen = true">管理变量</button></div>
    <p class="hint">无条件时默认允许；条件不满足时隐藏，并且不会自动跳转。</p>
    <select v-model="rules.conditionMode" aria-label="条件组合方式" :disabled="store.busy"><option value="all">全部条件满足</option><option value="any">任一条件满足</option></select>
    <div v-for="(c,i) in rules.conditions" :key="i" class="rule-row condition-row">
      <select v-model.number="c.variableId" :aria-label="`条件 ${i+1} 变量`" :disabled="store.busy"><option v-for="v in store.variables" :key="v.id" :value="v.id">{{ v.name }}</option></select>
      <select v-model="c.operator" :aria-label="`条件 ${i+1} 比较方式`" :disabled="store.busy"><option v-for="op in operators" :key="op" :value="op">{{ op }}</option></select>
      <input v-model.number="c.value" type="number" step="1" :aria-label="`条件 ${i+1} 数值`" :disabled="store.busy" />
      <button class="remove" :aria-label="`删除条件 ${i+1}`" :disabled="store.busy" @click="rules.conditions.splice(i,1)"><AppIcon name="close" :size="14" /></button>
    </div>
    <button class="btn add-rule" :disabled="store.busy || !store.variables.length" @click="addCondition">＋ 添加条件</button>
    <h4 class="effects-title">经过端口后的变化</h4>
    <p class="hint">增减：正数增加，负数减少；设值：直接改为填写的数值，填 0 即清空。未配置的变量保持不变。</p>
    <div v-for="(e,i) in rules.effects" :key="i" class="rule-row effect-row">
      <select v-model.number="e.variableId" :aria-label="`变化 ${i+1} 变量`" :disabled="store.busy"><option v-for="v in store.variables" :key="v.id" :value="v.id" :disabled="rules.effects.some(other => other !== e && other.variableId === v.id)">{{ v.name }}</option></select>
      <select v-model.number="e.set" :aria-label="`变化 ${i+1} 操作`" :disabled="store.busy"><option :value="0">增减</option><option :value="1">设值</option></select>
      <input v-model.number="e.delta" type="number" step="1" :aria-label="`变化 ${i+1} ${e.set === 1 ? '目标值' : '增减量'}`" :disabled="store.busy" />
      <button class="remove" :aria-label="`删除变化 ${i+1}`" :disabled="store.busy" @click="rules.effects.splice(i,1)"><AppIcon name="close" :size="14" /></button>
    </div>
    <button class="btn add-rule" :disabled="store.busy || !unused.length" @click="addEffect">＋ 添加变量变化</button>
    <p v-if="!store.variables.length" class="hint">请先在“管理变量”中添加工程变量。</p>
  </section>
</template>
<style scoped>
.rules-section { border: 1px solid var(--border); border-radius: 10px; padding: 14px; }
.rules-heading { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
h4 { font-size: 13px; margin: 0; }
.text-btn { background: transparent; border: 0; color: var(--accent); font-size: 12px; }
.hint { font-size: 12px; color: var(--text-dim); line-height: 1.6; margin: 8px 0; }
select { min-width: 0; max-width: 100%; height: 36px; padding: 5px 8px; border: 1px solid var(--border); border-radius: 6px; background: var(--bg); color: var(--text); font-size: 12px; }
.rule-row { display: grid; align-items: center; gap: 6px; margin-top: 8px; }
.condition-row { grid-template-columns: minmax(80px,1fr) 60px 90px 24px; }
.effect-row { grid-template-columns: minmax(80px,1fr) 72px 90px 24px; }
.rule-row input { min-width: 0; height: 36px; padding: 5px 8px; }
.remove { display: grid; place-items: center; border: 0; background: transparent; color: var(--text-dim); height: 30px; }
.add-rule { margin-top: 10px; font-size: 12px; }
.effects-title { margin-top: 20px; }
@media(max-width:480px) { .condition-row { grid-template-columns: minmax(60px,1fr) 52px 65px 20px; } .effect-row { grid-template-columns: minmax(60px,1fr) 60px 65px 20px; } }
</style>
