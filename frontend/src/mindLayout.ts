// 思维导图自动布局（纯函数，XMind 风格）：
// - 普通连线构成树：父节点在左，子节点竖直排列在右
// - 父节点垂直居中于其子树块；多根节点竖直排列
// - 魔法连线（hidden）不参与布局
import { CHILD_GAP, LEVEL_GAP, NODE_H, NODE_W, PORT_PAD, PORT_ROW_H, ROOT_X, ROOT_Y } from '@/canvasLayout'
import type { HtmlScene, NodeItem } from '@/types'

/** 布局占位高度：卡片固定 NODE_H，但按端口散开列高预留空间（避免子节点与端口重叠） */
export function sceneDisplayHeight(sceneId: number, nodes: NodeItem[]): number {
  let inCount = 0
  let outCount = 0
  for (const n of nodes) {
    if (n.sceneId !== sceneId) continue
    if (n.entry) inCount++
    else outCount++
  }
  const portH = Math.max(inCount, outCount) * PORT_ROW_H + PORT_PAD * 2
  return Math.max(NODE_H, portH)
}

/** 计算所有节点的思维导图坐标；collapsedIds 为折叠的节点（其子树不参与布局）；返回 Map<sceneId, {x,y}> */
export function computeMindLayout(
  scenes: HtmlScene[],
  nodes: NodeItem[],
  collapsedIds: Set<number>
): Map<number, { x: number; y: number }> {
  // 建树：出端口（!entry，非魔法）→ 配对入端口所在场景；折叠节点视为叶子
  const children = new Map<number, number[]>()
  const parentOf = new Map<number, number>()
  for (const n of nodes) {
    if (n.entry || n.magic) continue // 只从普通出端口出发（魔法端口不参与布局）
    const pair = nodes.find((x) => x.id !== n.id && x.hash === n.hash)
    if (!pair) continue // 无配对（空端口）不参与
    const a = n.sceneId
    const b = pair.sceneId
    if (a === b) continue
    if (collapsedIds.has(a)) continue // 父已折叠：子树不参与布局
    if (!parentOf.has(b)) parentOf.set(b, a) // 树：子节点只有一个父
    const list = children.get(a)
    if (list) list.push(b)
    else children.set(a, [b])
  }

  // 子树高度（含自身端口扩展高度）
  const hCache = new Map<number, number>()
  const subtreeH = (id: number): number => {
    const hit = hCache.get(id)
    if (hit != null) return hit
    const kids = children.get(id) || []
    let h = sceneDisplayHeight(id, nodes)
    if (kids.length > 0) {
      const sum = kids.reduce((s, k) => s + subtreeH(k), 0)
      h = Math.max(h, sum + CHILD_GAP * (kids.length - 1))
    }
    hCache.set(id, h)
    return h
  }

  const pos = new Map<number, { x: number; y: number }>()

  /** 递归放置：节点自身中心对齐 centerY，子节点块围绕 centerY 竖直排布 */
  const place = (id: number, x: number, centerY: number) => {
    const selfH = sceneDisplayHeight(id, nodes)
    pos.set(id, { x, y: centerY - selfH / 2 })
    const kids = children.get(id) || []
    if (kids.length === 0) return
    const total =
      kids.reduce((s, k) => s + subtreeH(k), 0) + CHILD_GAP * (kids.length - 1)
    let top = centerY - total / 2
    for (const k of kids) {
      const kh = subtreeH(k)
      place(k, x + NODE_W + LEVEL_GAP, top + kh / 2)
      top += kh + CHILD_GAP
    }
  }

  // 根列表（无父），保持 scenes 顺序；多根竖直排列
  const roots = scenes.map((s) => s.id).filter((id) => !parentOf.has(id))
  if (roots.length === 0) return pos
  const rootsH =
    roots.reduce((s, r) => s + subtreeH(r), 0) + CHILD_GAP * (roots.length - 1)
  let top = ROOT_Y - rootsH / 2
  for (const r of roots) {
    place(r, ROOT_X, top + subtreeH(r) / 2)
    top += subtreeH(r) + CHILD_GAP
  }
  return pos
}
