import test from 'node:test'
import assert from 'node:assert/strict'
import { treeVisibility } from '../src/treeVisibility.ts'

const scenes = ['root', 'a', 'b', 'a1', 'a2', 'a11', 'b1'].map((hash, id) => ({ id: id + 1, hash }))
const ordinary = [[1, 'a'], [1, 'b'], [2, 'a1'], [2, 'a2'], [4, 'a11'], [3, 'b1']]
const ports = ordinary.map(([nodeId, exportHash]) => ({ nodeId, exportHash, entry: false, magic: false }))
const visible = ids => [...treeVisibility(scenes, ports, [], ids).visibleIds]

test('compact nodes and their retained children share a fold owner', () => {
  const folded = treeVisibility(scenes, ports, [], [1, 2])
  assert.deepEqual([...folded.compactIds], [1, 2, 3])
  assert.deepEqual([...folded.foldOwners], [[1, 1], [2, 1], [3, 1]])
  const branch = treeVisibility(scenes, ports, [], [2])
  assert.deepEqual([...branch.compactIds], [2, 4, 5])
  assert.deepEqual([...branch.foldOwners], [[2, 2], [4, 2], [5, 2]])
  assert.equal(treeVisibility(scenes, ports, []).compactIds.size, 0)
})

test('root compact state retains the next level, hiding deeper descendants', () => {
  assert.deepEqual(visible([1]), [1, 2, 3])
  const { foldCounts } = treeVisibility(scenes, ports, [], [1])
  assert.equal(foldCounts.get(1), 6)
  assert.equal(foldCounts.get(2), 3)
  assert.equal(foldCounts.get(3), 1) // Direct children can now be fully hidden.
  assert.equal(foldCounts.get(6), 0)
})

test('branch compact state preserves siblings and its direct children', () => {
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
  assert.deepEqual(treeVisibility(scenes, [...ports, ...extra], [], [2]), treeVisibility(scenes, ports, [], [2]))
})


test('fully collapsed state hides direct children too; expanding restores nested states', () => {
  assert.deepEqual([...treeVisibility(scenes, ports, [1]).visibleIds], [1])
  assert.deepEqual([...treeVisibility(scenes, ports, [2]).visibleIds], [1, 2, 3, 7])
  const compactParent = treeVisibility(scenes, ports, [2], [1])
  assert.deepEqual([...compactParent.visibleIds], [1, 2, 3])
  assert.deepEqual([...compactParent.compactIds], [1, 2, 3])
  assert.deepEqual([...treeVisibility(scenes, ports, [1], [2]).visibleIds], [1])
})

test('zero-exit leaves in the retained layer still get a numeric port dot', () => {
  const result = treeVisibility(scenes, ports, [], [2])
  assert.ok(result.compactIds.has(5))
  assert.equal(ports.filter(p => p.nodeId === 5).length, 0)
  assert.equal(result.foldCounts.get(5), 0)
})
