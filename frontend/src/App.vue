<script setup lang="ts">
import { onMounted } from 'vue'
import TopBar from './components/TopBar.vue'
import TreeNav from './components/TreeNav.vue'
import CanvasArea from './components/CanvasArea.vue'
import SettingsModal from './components/SettingsModal.vue'
import NodeEditModal from './components/NodeEditModal.vue'
import HtmlBoxEditModal from './components/HtmlBoxEditModal.vue'
import NameInputModal from './components/NameInputModal.vue'
import { useAppStore } from '@/store/app'
import { useTreeStore } from '@/store/tree'
import { useCanvasStore } from '@/store/canvas'

const app = useAppStore()
const tree = useTreeStore()
const canvas = useCanvasStore()

onMounted(() => {
  document.body.classList.toggle('dark', app.dark)
  app.loadSettings()
  tree.load()
})

// 点击画布外任意处关闭右键菜单
function onBackgroundClick() {
  tree.closeMenu()
}
</script>

<template>
  <div class="layout" @click="onBackgroundClick">
    <TopBar />
    <div class="main">
      <TreeNav />
      <CanvasArea />
    </div>

    <!-- 设置弹窗 -->
    <SettingsModal v-if="app.settingsOpen" />

    <!-- 节点编辑弹窗 -->
    <NodeEditModal v-if="canvas.editingNode" :node="canvas.editingNode" @close="canvas.editingNode = null" />

    <!-- HTML 框编辑弹窗 -->
    <HtmlBoxEditModal
      v-if="canvas.editingScene != null"
      :scene-id="canvas.editingScene"
      @close="canvas.editingScene = null"
    />

    <!-- 新建弹窗 -->
    <NameInputModal
      v-if="tree.createDialog"
      :title="tree.createDialog.title"
      :placeholder="tree.createDialog.placeholder"
      @confirm="tree.doCreate"
      @cancel="tree.createDialog = null"
    />

    <!-- 重命名弹窗 -->
    <NameInputModal
      v-if="tree.renameDialog"
      :title="'重命名'"
      :initial="tree.renameDialog.name"
      confirm-text="保存"
      @confirm="tree.doRename"
      @cancel="tree.renameDialog = null"
    />

    <!-- 全局提示 -->
    <Teleport to="body">
      <div v-if="app.toast" class="toast" :class="app.toast.type">{{ app.toast.text }}</div>
    </Teleport>
  </div>
</template>

<style scoped>
.layout {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.main {
  flex: 1;
  display: flex;
  min-height: 0;
}
</style>
