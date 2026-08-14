// 树状导航状态：文件夹 → 画布；右键菜单、新建/重命名/删除
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import { useAppStore } from './app'
import { useCanvasStore } from './canvas'
import type { ItemKind, TreeFolder } from '@/types'

export interface Selected {
  kind: ItemKind
  id: number
}

export interface MenuState {
  x: number
  y: number
  kind: ItemKind
  id: number
  name: string
  parentId: number | null
  folderId?: number
}

export interface CreateDialog {
  kind: 'folder' | 'canvas'
  title: string
  placeholder: string
  parentId: number | null // 新建文件夹的父级
  folderId: number | null // 新建画布所属文件夹
}

export interface RenameDialog {
  kind: ItemKind
  id: number
  name: string
}

export const useTreeStore = defineStore('tree', {
  state: () => ({
    folders: [] as TreeFolder[],
    loading: false,
    selected: null as Selected | null,
    menu: null as MenuState | null,
    expanded: new Set<number>(),
    createDialog: null as CreateDialog | null,
    renameDialog: null as RenameDialog | null
  }),

  actions: {
    async load() {
      this.loading = true
      try {
        this.folders = await api.getTree()
      } catch (e: any) {
        useAppStore().handleApiError(e)
      } finally {
        this.loading = false
      }
    },

    select(kind: ItemKind, id: number) {
      this.selected = { kind, id }
    },

    openMenu(m: MenuState) {
      this.menu = m
    },
    closeMenu() {
      this.menu = null
    },

    toggleExpand(folderId: number) {
      if (this.expanded.has(folderId)) this.expanded.delete(folderId)
      else this.expanded.add(folderId)
    },

    openCreate(d: Partial<CreateDialog> & { kind: 'folder' | 'canvas' }) {
      this.createDialog = {
        parentId: null,
        folderId: null,
        title: '新建',
        placeholder: '请输入名称',
        ...d
      }
    },

    async doCreate(name: string) {
      const d = this.createDialog
      if (!d) return
      const app = useAppStore()
      try {
        if (d.kind === 'folder') {
          await api.createFolder(d.parentId, name)
        } else {
          if (d.folderId == null) throw new Error('请先选中一个文件夹')
          await api.createCanvas(d.folderId, name)
        }
        if (d.parentId != null) this.expanded.add(d.parentId)
        if (d.folderId != null) this.expanded.add(d.folderId)
        await this.load()
        app.notify('已创建', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
      this.createDialog = null
    },

    openRename(kind: ItemKind, id: number, name: string) {
      this.renameDialog = { kind, id, name }
    },

    async doRename(newName: string) {
      const d = this.renameDialog
      if (!d) return
      const app = useAppStore()
      try {
        await api.renameItem(d.kind, d.id, newName)
        await this.load()
        app.notify('已重命名', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
      this.renameDialog = null
    },

    async doDelete() {
      const m = this.menu
      if (!m) return
      const app = useAppStore()
      const kindName = m.kind === 'folder' ? '文件夹' : '画布'
      const tip =
        m.kind === 'folder'
          ? '该操作会同步删除本地目录与其中所有画布，不可恢复！'
          : '该画布中的所有 HTML 框与节点将一并删除！'
      if (!confirm(`确定删除${kindName}「${m.name}」？\n${tip}`)) {
        this.closeMenu()
        return
      }
      try {
        await api.deleteItem(m.kind, m.id)
        // 若删除的是当前打开的画布或其上级文件夹，清空视图
        const canvas = useCanvasStore()
        if (m.kind === 'folder') canvas.clear()
        if (m.kind === 'canvas' && canvas.canvasId === m.id) canvas.clear()
        if (this.selected && this.selected.kind === m.kind && this.selected.id === m.id) {
          this.selected = null
        }
        await this.load()
        app.notify('已删除', 'success')
      } catch (e: any) {
        app.handleApiError(e)
      }
      this.closeMenu()
    }
  }
})
