import type { Node, PortItem } from './types'

// 三态：展开全部；紧凑保留下一层；收起隐藏全部后代。
export function treeVisibility(
  scenes: Pick<Node, 'id' | 'hash'>[],
  ports: Pick<PortItem, 'nodeId' | 'exportHash' | 'entry' | 'magic'>[],
  collapsedIds: readonly number[],
  compactNodeIds: readonly number[] = []
) {
  const byHash = new Map(scenes.map(s => [s.hash, s.id]))
  const children = new Map(scenes.map(s => [s.id, new Set<number>()]))
  const hasParent = new Set<number>()
  for (const port of ports) {
    if (port.entry || port.magic) continue
    const target = byHash.get(port.exportHash)
    if (target == null || target === port.nodeId || !children.has(port.nodeId)) continue
    children.get(port.nodeId)!.add(target)
    hasParent.add(target)
  }

  const sizes = new Map<number, number>()
  const visiting = new Set<number>()
  function subtreeSize(id: number): number {
    if (visiting.has(id)) return 0
    if (sizes.has(id)) return sizes.get(id)!
    visiting.add(id)
    let size = 1
    for (const child of children.get(id) || []) size += subtreeSize(child)
    visiting.delete(id)
    sizes.set(id, size)
    return size
  }
  const foldCounts = new Map<number, number>()
  for (const scene of scenes) {
    let count = 0
    for (const child of children.get(scene.id) || []) count += subtreeSize(child)
    foldCounts.set(scene.id, count)
  }

  const collapsed = new Set(collapsedIds)
  const compact = new Set(compactNodeIds)
  const visibleIds = new Set<number>()
  const compactIds = new Set<number>()
  const foldOwners = new Map<number, number>()
  const queue = scenes.filter(s => !hasParent.has(s.id)).map(s => ({ id: s.id, owner: null as number | null }))
  for (let i = 0; i < queue.length; i++) {
    const { id, owner } = queue[i]
    if (visibleIds.has(id)) continue
    visibleIds.add(id)
    if (owner != null || collapsed.has(id) || compact.has(id)) {
      compactIds.add(id)
      foldOwners.set(id, owner ?? id)
    }
    if (owner != null || collapsed.has(id)) continue
    for (const child of children.get(id) || []) queue.push({ id: child, owner: compact.has(id) ? id : null })
  }
  return { visibleIds, foldCounts, compactIds, foldOwners }
}
