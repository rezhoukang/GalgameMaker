// 应用级状态：设置（只读）、设置弹窗、全局提示、导出
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import type { Settings, ExportResult } from '@/types'

let toastTimer: ReturnType<typeof setTimeout> | null = null

export const useAppStore = defineStore('app', {
  state: () => ({
    settings: null as Settings | null,
    settingsOpen: false,
    exportResult: null as ExportResult | null,
    dark: localStorage.getItem('galgame-dark') === '1',
    navCollapsed: localStorage.getItem('galgame-nav') === '1',
    toast: null as { text: string; type: 'info' | 'error' | 'success' } | null
  }),

  actions: {
    /** 切换黑夜模式：挂到 body.dark + 持久化 */
    toggleDark() {
      this.dark = !this.dark
      document.body.classList.toggle('dark', this.dark)
      localStorage.setItem('galgame-dark', this.dark ? '1' : '0')
    },

    /** 切换左侧导航栏收起/展开 */
    toggleNav() {
      this.navCollapsed = !this.navCollapsed
      localStorage.setItem('galgame-nav', this.navCollapsed ? '1' : '0')
    },

    async loadSettings() {
      try {
        this.settings = await api.getSettings()
      } catch (e: any) {
        this.notify(e?.message || String(e), 'error')
      }
    },

    openSettings() {
      this.settingsOpen = true
    },
    closeSettings() {
      this.settingsOpen = false
    },

    /** 统一处理 API 错误 */
    handleApiError(e: any, notify = true) {
      const msg = e && e.message ? e.message : String(e)
      if (notify) this.notify(msg, 'error')
    },

    notify(text: string, type: 'info' | 'error' | 'success' = 'info') {
      if (toastTimer) clearTimeout(toastTimer)
      this.toast = { text, type }
      toastTimer = setTimeout(() => {
        this.toast = null
      }, 3200)
    },

    async exportAll(canvasId: number) {
      try {
        this.exportResult = await api.exportAll(canvasId)
        this.notify(
          `已导出（单页播放器 + ${this.exportResult.fileCount} 个资源文件）→ ${this.exportResult.outputDir}`,
          'success'
        )
      } catch (e: any) {
        this.handleApiError(e)
      }
    }
  }
})
