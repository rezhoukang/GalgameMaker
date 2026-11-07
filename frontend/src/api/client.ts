// 轻量 API 客户端（fetch 封装）
import type {
  Settings,
  TreeFolder,
  Canvas,
  CanvasView,
  Node,
  PortItem,
  ExportResult,
  ItemKind
} from '@/types'

export class ApiError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.code = code
  }
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const isForm = options?.body instanceof FormData
  const res = await fetch(url, {
    ...options,
    headers: isForm
      ? undefined
      : { 'Content-Type': 'application/json', ...((options?.headers as Record<string, string>) || {}) }
  })
  let json: any
  try {
    json = await res.json()
  } catch {
    throw new ApiError(res.status, `请求失败（HTTP ${res.status}）`)
  }
  if (json.code !== 0) {
    throw new ApiError(json.code ?? res.status, json.message || '请求失败')
  }
  return json.data as T
}

export const api = {
  // 设置（只读）
  getSettings: () => request<Settings>('/api/settings'),

  // 树：文件夹 → 画布
  getTree: () => request<TreeFolder[]>('/api/tree'),
  createFolder: (parentId: number | null, name: string) =>
    request<{ id: number }>('/api/folders', {
      method: 'POST',
      body: JSON.stringify({ parentId, name })
    }),
  createCanvas: (folderId: number, name: string) =>
    request<Canvas>('/api/canvases', {
      method: 'POST',
      body: JSON.stringify({ folderId, name })
    }),
  renameItem: (kind: ItemKind, id: number, newName: string) =>
    request<void>('/api/items/rename', {
      method: 'PATCH',
      body: JSON.stringify({ kind, id, newName })
    }),
  deleteItem: (kind: ItemKind, id: number) =>
    request<void>(`/api/items?kind=${kind}&id=${id}`, { method: 'DELETE' }),

  // 画布视图
  getCanvas: (id: number) => request<CanvasView>(`/api/canvas/${id}`),
  // 画布检测：空端口 / 双角色端口 / 跳转资源缺失
  checkCanvas: (id: number) =>
    request<{ sceneName: string; nodeName: string; problem: string }[]>(`/api/canvas/${id}/check`),

  // HTML 框（节点）
  updateScene: (id: number, data: { name?: string; x?: number; y?: number }) =>
    request<Node>(`/api/scenes/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteScene: (id: number) => request<void>(`/api/scenes/${id}`, { method: 'DELETE' }),
  // 上传 HTML 框关联的 mp4 视频
  uploadSceneVideo: (id: number, file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<Node>(`/api/scenes/${id}/video`, { method: 'POST', body: fd })
  },
  // 导入/覆盖节点的 HTML 内容（文件名不变）
  uploadSceneHtml: (id: number, file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<Node>(`/api/scenes/${id}/html`, { method: 'PUT', body: fd })
  },
  // 从节点的右侧加号创建下一跳：新增出端口 + 右侧空白节点（带入端口），两端共用同一哈希
  createNextNode: (nodeId: number) =>
    request<{ scene: Node; outNode: PortItem; entryNode: null }>(`/api/scenes/${nodeId}/next`, {
      method: 'POST'
    }),
  // 新建魔法出端口并连向指定哈希的节点（不画线、导出生效）
  createMagicPort: (nodeId: number, hash: string) =>
    request<{ outNode: PortItem; entryNode: null }>(`/api/scenes/${nodeId}/magic-port`, {
      method: 'POST',
      body: JSON.stringify({ hash })
    }),
  // 设为当前工程的播放起点
  setFirstScene: (id: number) => request<Node>(`/api/scenes/${id}/first`, { method: 'POST' }),

  // 出端口，入端口只是前端展示
  updateNode: (
    id: number,
    data: { name?: string; isShow?: boolean }
  ) => request<PortItem>(`/api/nodes/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteNode: (id: number) => request<void>(`/api/nodes/${id}`, { method: 'DELETE' }),
  // 断开魔法连接；普通连接需移动或删除分支
  breakNode: (id: number) => request<void>(`/api/nodes/${id}/break`, { method: 'DELETE' }),
  // 重定向魔法端口一端到指定节点（按节点哈希）：side=prev 改上一节点 / next 改下一节点
  redirectNode: (id: number, hash: string, side: 'prev' | 'next') =>
    request<void>(`/api/nodes/${id}/redirect`, {
      method: 'PUT',
      body: JSON.stringify({ hash, side })
    }),

  restoreCanvas: (id: number, expected: string, snapshot: CanvasView) =>
    request<CanvasView>(`/api/canvas/${id}/restore`, {method: 'PUT', body: JSON.stringify({expected, snapshot})}),
  copyProperties: (id: number, sourceId: number, snapshot: CanvasView) =>
    request<void>(`/api/scenes/${id}/copy-properties`, {method: 'POST', body: JSON.stringify({sourceId, snapshot})}),

  // 导出
  // 导出指定画布为单页播放器 + 资源目录
  exportAll: (canvasId: number) =>
    request<ExportResult>('/api/export', { method: 'POST', body: JSON.stringify({ canvasId }) }),

  // 初始化（清空数据库与存储目录）
  reset: () => request<void>('/api/reset', { method: 'POST' })
}
