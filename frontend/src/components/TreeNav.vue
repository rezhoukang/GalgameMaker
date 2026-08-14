<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useTreeStore } from '@/store/tree'
import { useCanvasStore } from '@/store/canvas'
import { useAppStore } from '@/store/app'
import AppIcon from './AppIcon.vue'
import type { TreeFolder } from '@/types'

interface FlatItem {
  key: string
  kind: 'folder' | 'canvas'
  id: number
  name: string
  depth: number
  parentId: number | null
  folderId?: number
  nodeCount?: number
  expanded?: boolean
}

const tree = useTreeStore()
const canvas = useCanvasStore()
const app = useAppStore()

// 把树扁平化为带缩进的行：按 parentId 嵌套（顶层文件夹 → 子文件夹 → 画布）
const flatItems = computed<FlatItem[]>(() => {
  const out: FlatItem[] = []
  // 按 parentId 分组（顶层 = null）
  const byParent = new Map<number | null, TreeFolder[]>()
  for (const f of tree.folders) {
    const key = f.parentId ?? null
    const list = byParent.get(key)
    if (list) list.push(f)
    else byParent.set(key, [f])
  }
  const walk = (parents: TreeFolder[], depth: number) => {
    for (const f of parents) {
      const expanded = tree.expanded.has(f.id)
      out.push({
        key: `f-${f.id}`,
        kind: 'folder',
        id: f.id,
        name: f.name,
        depth,
        parentId: f.parentId,
        expanded
      })
      if (expanded) {
        for (const c of f.canvases) {
          out.push({
            key: `c-${c.id}`,
            kind: 'canvas',
            id: c.id,
            name: c.name,
            depth: depth + 1,
            parentId: f.id,
            folderId: f.id,
            nodeCount: c.nodeCount
          })
        }
        const kids = byParent.get(f.id)
        if (kids) walk(kids, depth + 1)
      }
    }
  }
  walk(byParent.get(null) ?? [], 0)
  return out
})

const isSelected = (it: FlatItem) =>
  tree.selected?.kind === it.kind && tree.selected?.id === it.id

function onRowClick(it: FlatItem) {
  if (it.kind === 'folder') {
    tree.toggleExpand(it.id)
    tree.select('folder', it.id)
    canvas.clear() // 选中文件夹：画布区空白
  } else {
    tree.select('canvas', it.id)
    canvas.openCanvas(it.id, it.name)
  }
}

function onRowContext(event: MouseEvent, it: FlatItem) {
  event.preventDefault()
  tree.openMenu({
    x: event.clientX,
    y: event.clientY,
    kind: it.kind,
    id: it.id,
    name: it.name,
    parentId: it.parentId ?? null,
    folderId: it.folderId
  })
}

function onRootContext(event: MouseEvent) {
  event.preventDefault()
  tree.openMenu({ x: event.clientX, y: event.clientY, kind: 'folder', id: 0, name: '', parentId: null })
}

onMounted(() => {
  tree.load()
})
</script>

<template>
  <aside class="nav" :class="{ collapsed: app.navCollapsed }">
    <div class="nav-head" :class="{ collapsed: app.navCollapsed }">
      <span v-if="!app.navCollapsed" class="title">项目文件</span>
      <button
        class="nav-toggle"
        :class="{ collapsed: app.navCollapsed }"
        :title="app.navCollapsed ? '展开导航栏' : '收起导航栏'"
        @click.stop="app.toggleNav()"
      >
        <AppIcon name="back" :size="16" />
      </button>
    </div>

    <div v-if="!app.navCollapsed" class="nav-body" @contextmenu.prevent="onRootContext">
      <div v-if="tree.loading" class="empty">加载中…</div>
      <div v-else-if="flatItems.length === 0" class="empty">
        <p>暂无内容</p>
        <p class="sub">右键此处新建文件夹，<br />右键文件夹可新建画布</p>
      </div>

      <div
        v-for="it in flatItems"
        :key="it.key"
        class="row"
        :class="{ selected: isSelected(it), folder: it.kind === 'folder' }"
        :style="{ paddingLeft: 10 + it.depth * 16 + 'px' }"
        @click="onRowClick(it)"
        @contextmenu.stop.prevent="onRowContext($event, it)"
      >
        <span v-if="it.kind === 'folder'" class="arrow" :class="{ open: it.expanded }">▸</span>
        <span v-else class="arrow placeholder" />
        <span class="ico">
          <AppIcon :name="it.kind === 'folder' ? 'folder' : 'file'" :size="15" />
        </span>
        <span class="name" :title="it.name">{{ it.name }}</span>
        <span v-if="it.kind === 'canvas'" class="count">{{ it.nodeCount }}</span>
      </div>
    </div>

    <!-- 右键菜单 -->
    <Teleport to="body">
      <div v-if="tree.menu" class="ctx" :style="{ left: tree.menu.x + 'px', top: tree.menu.y + 'px' }">
        <template v-if="tree.menu.kind === 'folder' && tree.menu.id !== 0">
          <div class="ctx-title"><AppIcon name="folder" :size="13" /> {{ tree.menu.name }}</div>
          <button @click="tree.openCreate({ kind: 'folder', parentId: tree.menu!.id, folderId: tree.menu!.id, title: '在文件夹下新建子文件夹' })">
            <AppIcon name="plus" :size="13" /> 新建子文件夹
          </button>
          <button @click="tree.openCreate({ kind: 'canvas', folderId: tree.menu!.id, title: '新建画布' })">
            <AppIcon name="plus" :size="13" /> 新建画布
          </button>
          <div class="divider" />
          <button @click="tree.openRename('folder', tree.menu!.id, tree.menu!.name)"><AppIcon name="rename" :size="13" /> 重命名</button>
          <button class="danger" @click="tree.doDelete()"><AppIcon name="trash" :size="13" /> 删除</button>
        </template>

        <template v-else-if="tree.menu.kind === 'canvas'">
          <div class="ctx-title"><AppIcon name="file" :size="13" /> {{ tree.menu.name }}</div>
          <button @click="tree.openRename('canvas', tree.menu!.id, tree.menu!.name)"><AppIcon name="rename" :size="13" /> 重命名</button>
          <button class="danger" @click="tree.doDelete()"><AppIcon name="trash" :size="13" /> 删除</button>
        </template>

        <template v-else>
          <!-- 空白处右键：新建文件夹 -->
          <div class="ctx-title">项目根</div>
          <button @click="tree.openCreate({ kind: 'folder', parentId: null, title: '新建文件夹' })">
            <AppIcon name="plus" :size="13" /> 新建文件夹
          </button>
        </template>
      </div>
    </Teleport>
  </aside>
</template>

<style scoped>
.nav {
  width: 264px;
  min-width: 264px;
  height: 100%;
  background: var(--bg-soft);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
}
.nav-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
}
.nav-head .title {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-dim);
  letter-spacing: 1px;
}
/* 收起状态：只留一条窄栏 + 展开按钮 */
.nav.collapsed {
  width: 46px;
  min-width: 46px;
}
.nav-head.collapsed {
  justify-content: center;
  padding: 12px 0;
  border-bottom: none;
}
.nav-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--border-light);
  border-radius: 7px;
  background: var(--bg);
  color: var(--text-dim);
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
}
.nav-toggle:hover {
  border-color: var(--accent);
  color: var(--accent);
}
.nav-toggle svg {
  transition: transform 0.2s;
}
.nav-toggle.collapsed svg {
  transform: rotate(180deg); /* back 左箭头 → 右箭头（展开） */
}
.nav-body {
  flex: 1;
  overflow-y: auto;
  padding: 6px 0;
}
.row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  cursor: pointer;
  border-radius: 6px;
  margin: 1px 6px;
  user-select: none;
  white-space: nowrap;
}
.row:hover {
  background: var(--bg);
}
.row.selected {
  background: rgba(255, 184, 108, 0.12);
  outline: 1px solid rgba(255, 184, 108, 0.35);
}
.arrow {
  width: 14px;
  font-size: 11px;
  color: var(--text-faint);
  transition: transform 0.15s;
  flex-shrink: 0;
}
.arrow.open {
  transform: rotate(90deg);
}
.arrow.placeholder {
  visibility: hidden;
}
.ico {
  display: flex;
  color: var(--text-dim);
  flex-shrink: 0;
}
.name {
  flex: 1;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.count {
  font-size: 11px;
  color: var(--text-faint);
  background: var(--bg);
  border-radius: 10px;
  padding: 0 7px;
  flex-shrink: 0;
}
.empty {
  text-align: center;
  color: var(--text-faint);
  padding: 40px 12px;
  font-size: 13px;
  line-height: 1.8;
}
.empty .sub {
  font-size: 12px;
  color: var(--text-faint);
}
.ctx {
  position: fixed;
  z-index: 1500;
  min-width: 180px;
  background: var(--bg-soft);
  border: 1px solid var(--border-light);
  border-radius: 10px;
  padding: 6px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
}
.ctx-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--text-faint);
  padding: 4px 10px 8px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 220px;
}
.ctx button {
  display: flex;
  align-items: center;
  gap: 8px;
  text-align: left;
  padding: 8px 12px;
  border: none;
  background: none;
  color: var(--text);
  font-size: 13px;
  border-radius: 6px;
  cursor: pointer;
}
.ctx button:hover {
  background: var(--bg);
}
.ctx button.danger:hover {
  background: rgba(255, 107, 107, 0.12);
  color: var(--danger);
}
.divider {
  height: 1px;
  background: var(--border);
  margin: 4px 6px;
}
</style>
