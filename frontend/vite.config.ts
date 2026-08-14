import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import VueDevTools from 'vite-plugin-vue-devtools'

// ============================================================
// Vue DevTools 开关
// - true：开启（默认）
// - false：关闭（插件完全不加载）
// 下方保留「生产自动关」保护：构建生产包时不注入 DevTools
// ============================================================
const DEVTOOLS_ENABLED = true

const enableDevTools =
  DEVTOOLS_ENABLED && process.env.NODE_ENV !== 'production'

export default defineConfig({
  plugins: [
    ...(enableDevTools
      ? [
          VueDevTools({
            launchEditor: 'code', // 点击组件用 VS Code 打开源码
            componentInspector: true // 组件检查器（Ctrl+Shift 定位组件）
          })
        ]
      : []),
    vue()
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    port: 5173,
    // 开发时把 /api 代理到 Go 后端，避免跨域
    proxy: {
      '/api': {
        target: 'http://localhost:8787',
        changeOrigin: true
      }
    }
  }
})
