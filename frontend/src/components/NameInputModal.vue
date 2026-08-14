<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'

const props = defineProps<{
  title: string
  placeholder?: string
  initial?: string
  confirmText?: string
}>()
const emit = defineEmits<{
  (e: 'confirm', value: string): void
  (e: 'cancel'): void
}>()

const value = ref(props.initial ?? '')
const inputRef = ref<HTMLInputElement | null>(null)

onMounted(() => {
  nextTick(() => {
    inputRef.value?.focus()
    inputRef.value?.select()
  })
})

function confirm() {
  emit('confirm', value.value.trim())
}
</script>

<template>
  <div class="modal-mask" @click.self="emit('cancel')">
    <div class="modal">
      <h3>{{ title }}</h3>
      <div class="field">
        <input
          ref="inputRef"
          v-model="value"
          :placeholder="placeholder ?? '请输入名称'"
          @keyup.enter="confirm"
          @keyup.esc="emit('cancel')"
        />
      </div>
      <div class="row">
        <button class="btn" @click="emit('cancel')">取消</button>
        <button class="btn primary" @click="confirm">{{ confirmText ?? '确定' }}</button>
      </div>
    </div>
  </div>
</template>
