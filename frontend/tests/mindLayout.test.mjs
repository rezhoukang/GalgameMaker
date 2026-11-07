import test from 'node:test'
import assert from 'node:assert/strict'
import { computeMindLayout, sceneDisplayHeight } from '../src/mindLayout.ts'
import { treeVisibility } from '../src/treeVisibility.ts'
import { NODE_H, NODE_W, LEVEL_GAP, CHILD_GAP, ROOT_Y } from '../src/canvasLayout.ts'

const scenes = Array.from({ length: 21 }, (_, id) => ({ id: id + 1, hash: `node-${id + 1}` }))
const ports = []
function connect(from, to) {
  const hash = `edge-${from}-${to}`
  ports.push({ id: ports.length + 1, nodeId: from, hash, exportHash: `node-${to}`, entry: false, magic: false })
  ports.push({ id: ports.length + 1, nodeId: to, hash, exportHash: '', entry: true, magic: false })
}
for (const child of [2, 3, 4]) connect(1, child)
for (let child = 5; child <= 21; child++) connect(3, child)

function layout(collapsed) {
  const { visibleIds, compactIds } = treeVisibility(scenes, ports, collapsed)
  return computeMindLayout(scenes.filter(s => visibleIds.has(s.id)), ports, compactIds)
}

test('collapsing a 17-exit branch removes hidden subtree space and compacts retained siblings', () => {
  const expanded = layout([])
  const folded = layout([1])
  assert.equal(expanded.size, 21)
  assert.equal(folded.size, 4)
  assert.equal(sceneDisplayHeight(3, ports, true), NODE_H)
  assert.ok(sceneDisplayHeight(3, ports) > NODE_H * 8)
  for (const [a, b] of [[2, 3], [3, 4]]) {
    assert.equal(folded.get(b).y - folded.get(a).y, NODE_H + CHILD_GAP)
    assert.ok(expanded.get(b).y - expanded.get(a).y > folded.get(b).y - folded.get(a).y)
  }
  assert.equal(folded.get(3).x - folded.get(1).x, NODE_W + LEVEL_GAP)
  assert.equal(folded.get(3).y, folded.get(1).y)
  assert.equal(folded.get(1).y + NODE_H / 2, ROOT_Y)
  assert.deepEqual(layout([]), expanded)
})

test('expanded card centers remain aligned despite tall outgoing port columns', () => {
  const positions = layout([])
  const first = positions.get(5).y
  const last = positions.get(21).y
  assert.equal(positions.get(3).y, (first + last) / 2)
})

test('compaction preserves the space required by incoming ports', () => {
  const inputs = Array.from({ length: 8 }, () => ({ nodeId: 3, entry: true, magic: true }))
  assert.ok(sceneDisplayHeight(3, [...ports, ...inputs], true) > NODE_H)
})
