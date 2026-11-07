import test from 'node:test'
import assert from 'node:assert/strict'
import { treeVisibility } from '../src/treeVisibility.ts'

const scenes = ['root', 'a', 'b', 'a1', 'a2', 'a11', 'b1'].map((hash, id) => ({ id: id + 1, hash }))
const ordinary = [[1, 'a'], [1, 'b'], [2, 'a1'], [2, 'a2'], [4, 'a11'], [3, 'b1']]
const ports = ordinary.map(([nodeId, exportHash]) => ({ nodeId, exportHash, entry: false, magic: false }))
const visible = ids => [...treeVisibility(scenes, ports, ids).visibleIds]

test('root collapse retains the next level, hiding deeper descendants', () => {
  assert.deepEqual(visible([1]), [1, 2, 3])
  const { foldCounts } = treeVisibility(scenes, ports, [1])
  assert.equal(foldCounts.get(1), 4)
  assert.equal(foldCounts.get(2), 1)
  assert.equal(foldCounts.get(3), 0) // Only direct leaf children: nothing to hide.
  assert.equal(foldCounts.get(6), 0)
})

test('branch collapse preserves siblings and its direct children', () => {
  assert.deepEqual(visible([2]), [1, 2, 3, 4, 5, 7])
  assert.deepEqual(visible([1, 2]), [1, 2, 3])
  assert.deepEqual(visible([]), [1, 2, 3, 4, 5, 7, 6])
})

test('magic links and derived input ports do not affect folding or counts', () => {
  const extra = [
    { nodeId: 3, exportHash: 'a11', entry: false, magic: true },
    { nodeId: 6, exportHash: 'root', entry: false, magic: true },
    { nodeId: 2, exportHash: 'root', entry: true, magic: false }
  ]
  assert.deepEqual(treeVisibility(scenes, [...ports, ...extra], [2]), treeVisibility(scenes, ports, [2]))
})
