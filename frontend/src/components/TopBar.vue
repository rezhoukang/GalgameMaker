<script setup lang="ts">
import { useAppStore } from '@/store/app'
import { useTreeStore } from '@/store/tree'
import { useCanvasStore } from '@/store/canvas'
import { api } from '@/api/client'
import AppIcon from './AppIcon.vue'

const app = useAppStore()
const tree = useTreeStore()
const canvas = useCanvasStore()

async function onReset() {
  if (!confirm('确定初始化？\n将清空数据库与存储目录中的全部数据，不可恢复！')) return
  try {
    await api.reset()
    await tree.load()
    canvas.clear()
    app.notify('已初始化，可以重新开始', 'success')
  } catch (e: any) {
    app.handleApiError(e)
  }
}

/** 导出前检测：空端口/双角色端口/跳转资源缺失，有问题则列出并中止 */
async function onExport() {
  if (canvas.canvasId == null) {
    app.notify('请先在左侧选择要导出的画布', 'error')
    return
  }
  try {
    const issues = await api.checkCanvas(canvas.canvasId)
    if (issues && issues.length > 0) {
      const list = issues.map((i) => `【${i.sceneName}】${i.nodeName}：${i.problem}`).join('\n')
      confirm(`导出检测发现 ${issues.length} 个问题：\n\n${list}\n\n请先修复后再导出。`)
      return
    }
  } catch (e: any) {
    app.handleApiError(e)
    return
  }
  await app.exportAll(canvas.canvasId)
}
</script>

<template>
  <header class="topbar">
    <button class="icon-btn" title="设置" @click="app.openSettings()">
      <AppIcon name="gear" />
    </button>
    <div class="brand">
      <h1>Galgame 制作器</h1>
      <span class="storage" :title="app.settings?.storagePath">
        存储：{{ app.settings?.storagePath }}
      </span>
    </div>
    <div class="spacer" />
    <button class="icon-btn" :title="app.dark ? '切换白天模式' : '切换黑夜模式'" @click="app.toggleDark()">
      <AppIcon :name="app.dark ? 'sun' : 'moon'" />
    </button>
    <button class="btn reset" title="清空所有数据（数据库 + 存储目录），重新开始" @click="onReset">
      <AppIcon name="close" :size="13" />
      初始化
    </button>
    <button class="btn export" title="导出为单页播放器 + 资源目录（先自动检测问题）" @click="onExport">
      <AppIcon name="export" :size="15" />
      导出 Galgame Output
    </button>
  </header>
</template>

<style scoped>
.topbar {
  height: 52px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 16px;
  background: var(--bg-soft);
  border-bottom: 1px solid var(--border);
  user-select: none;
}
.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: var(--bg);
  border: 1px solid var(--border-light);
  border-radius: 8px;
  color: var(--text-dim);
  transition: all 0.15s;
}
.icon-btn:hover {
  border-color: var(--accent);
  color: var(--accent);
  transform: rotate(20deg);
}
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}
.brand h1 {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 1px;
}
.storage {
  font-size: 12px;
  color: var(--text-faint);
  background: var(--bg);
  border: 1px solid var(--border);
  padding: 3px 10px;
  border-radius: 20px;
  max-width: 420px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.spacer {
  flex: 1;
}
.export {
  font-size: 13px;
  font-weight: 600;
  color: var(--accent);
  border-color: var(--accent);
}
.export:hover {
  background: var(--accent-soft);
}
.reset {
  font-size: 13px;
  color: var(--danger);
  border-color: var(--danger);
}
.reset:hover {
  background: rgba(255, 107, 107, 0.08);
  color: var(--danger);
}
</style>

