<script setup lang="ts">
import { useAppStore } from '@/store/app'
import { useTreeStore } from '@/store/tree'
import { useCanvasStore } from '@/store/canvas'
import { api } from '@/api/client'
import AppIcon from './AppIcon.vue'

const app = useAppStore()
const tree = useTreeStore()
const canvas = useCanvasStore()

const shortcuts = [
  { label: '框选节点', keys: '按住 Q + 左键拖动' },
  { label: '撤回', keys: 'Ctrl + Z' },
  { label: '重做（恢复撤回）', keys: 'Ctrl + Shift + Z / Ctrl + Y' },
  { label: '复制节点内容属性', keys: 'Ctrl + C' },
  { label: '粘贴节点内容属性', keys: 'Ctrl + V' },
  { label: '适应画布', keys: 'Ctrl + 0' },
  { label: '编辑选中节点', keys: 'F2' },
  { label: '删除选中节点', keys: 'Delete' },
  { label: '关闭属性窗口', keys: 'Esc' },
  { label: '平移画布', keys: '空格 + 拖动 / 中键拖动' },
  { label: '平移画布', keys: '拖动空白 / Ctrl + 拖动' },
  { label: '缩放画布', keys: '鼠标滚轮' }
]

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
  if (canvas.busy) return
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
    <button class="btn" :disabled="!canvas.isCanvasView || canvas.busy" @click="canvas.variablesOpen = true">剧情变量</button>
    <div class="shortcut-help">
      <button type="button" class="icon-btn help-btn" aria-label="快捷键帮助" aria-describedby="shortcut-guide">?</button>
      <div id="shortcut-guide" class="shortcut-guide" role="tooltip">
        <h2>快捷键与画布操作</h2>
        <dl>
          <div v-for="item in shortcuts" :key="item.keys" class="shortcut-row">
            <dt>{{ item.label }}</dt>
            <dd><kbd>{{ item.keys }}</kbd></dd>
          </div>
        </dl>
        <p>复制、粘贴前先选中节点；输入框内保留文字编辑快捷键。</p>
      </div>
    </div>
    <button class="icon-btn" :title="app.dark ? '切换白天模式' : '切换黑夜模式'" @click="app.toggleDark()">
      <AppIcon :name="app.dark ? 'sun' : 'moon'" />
    </button>
    <button class="btn reset" title="清空所有数据（数据库 + 存储目录），重新开始" @click="onReset">
      <AppIcon name="close" :size="13" />
      初始化
    </button>
    <button class="btn export" :disabled="canvas.busy" title="导出为单页播放器 + 资源目录（先自动检测问题）" @click="onExport">
      <AppIcon name="export" :size="15" />
      导出 Galgame Output
    </button>
  </header>
</template>

<style scoped>
.shortcut-help { position: relative; flex: none; padding: 10px 0; margin: -10px 0; z-index: 100; }
.help-btn { font-size: 18px; font-weight: 600; }
.shortcut-help .help-btn:hover { transform: none; }
.help-btn:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
.shortcut-guide { display: none; position: absolute; top: 100%; right: 0; width: min(420px, calc(100vw - 32px)); padding: 16px; border: 1px solid var(--border); border-radius: 12px; background: var(--bg); color: var(--text); box-shadow: 0 10px 32px rgba(28, 28, 34, 0.14); max-height: calc(100dvh - 68px); overflow-y: auto; }
.shortcut-help:hover .shortcut-guide, .shortcut-help:has(.help-btn:focus-visible) .shortcut-guide { display: block; }
.shortcut-guide h2 { margin: 0 0 10px; font-size: 14px; }
.shortcut-guide dl { margin: 0; }
.shortcut-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 7px 0; font-size: 12px; }
.shortcut-row dt { flex: none; color: var(--text-dim); }
.shortcut-row dd { margin: 0; text-align: right; }
.shortcut-row kbd { font-family: inherit; font-size: 11px; line-height: 1.6; }
.shortcut-guide p { border-top: 1px solid var(--border); padding-top: 12px; margin: 10px 0 0; color: var(--text-dim); font-size: 12px; line-height: 1.6; }
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

