<script setup lang="ts">
import { useAppStore } from '@/store/app'
import AppIcon from './AppIcon.vue'

const app = useAppStore()
</script>

<template>
  <div class="modal-mask" @click.self="app.closeSettings()">
    <div class="modal">
      <h3><AppIcon name="gear" :size="18" /> 设置</h3>

      <div class="field">
        <label>本地存储目录（导航栏文件夹映射到该目录下）</label>
        <div class="path-box">{{ app.settings?.storagePath }}</div>
      </div>

      <div class="field">
        <label>导出目录（每个节点导出为独立 HTML 文件）</label>
        <div class="path-box">{{ app.settings?.outputDir }}</div>
      </div>

      <p class="note">
        存储与导出目录固定在后端目录下，无需手动配置。
      </p>

      <div class="row">
        <button class="btn primary" @click="app.closeSettings()">知道了</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.note {
  font-size: 12px;
  color: var(--text-dim);
  background: var(--bg-soft);
  border: 1px dashed var(--border-light);
  border-radius: 8px;
  padding: 10px 12px;
  line-height: 1.6;
}
</style>

