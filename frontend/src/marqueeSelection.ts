import { NODE_H, NODE_W } from './canvasLayout.ts'

export type Point = { x: number; y: number }

export function selectionRect(start: Point, end: Point) {
  return { left: Math.min(start.x, end.x), top: Math.min(start.y, end.y), width: Math.abs(end.x - start.x), height: Math.abs(end.y - start.y) }
}

/** 框选使用视口坐标，命中可见节点的卡片区域，不包含外伸端口。 */
export function marqueeNodeIds(nodes: { id: number; x: number; y: number }[], start: Point, end: Point, view: Point & { zoom: number }): number[] {
  if (Math.abs(end.x - start.x) < 3 && Math.abs(end.y - start.y) < 3) return []
  const rect = selectionRect(start, end)
  const left = (rect.left - view.x) / view.zoom
  const top = (rect.top - view.y) / view.zoom
  const right = left + rect.width / view.zoom
  const bottom = top + rect.height / view.zoom
  return nodes.filter(n => n.x <= right && n.x + NODE_W >= left && n.y <= bottom && n.y + NODE_H >= top).map(n => n.id)
}
