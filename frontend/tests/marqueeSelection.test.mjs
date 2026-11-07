import test from 'node:test'
import assert from 'node:assert/strict'
import { marqueeNodeIds } from '../src/marqueeSelection.ts'

test('forward and reverse boxes select the same cards across zoom and translation', () => {
  const nodes = [{ id: 1, x: -240, y: -44 }, { id: 2, x: 0, y: 50 }, { id: 3, x: 480, y: 300 }]
  for (const zoom of [0.25, 0.5, 1, 3]) {
    const view = { x: 300, y: 150, zoom }
    const start = { x: -250 * zoom + view.x, y: -50 * zoom + view.y }
    const end = { x: 250 * zoom + view.x, y: 100 * zoom + view.y }
    assert.deepEqual(marqueeNodeIds(nodes, start, end, view), [1, 2])
    assert.deepEqual(marqueeNodeIds(nodes, end, start, view), [1, 2])
  }
})

test('box intersects cards only, excludes empty gestures and off-card ports', () => {
  const nodes = [{ id: 1, x: 0, y: 0 }]
  const view = { x: 0, y: 0, zoom: 1 }
  assert.deepEqual(marqueeNodeIds(nodes, {x: 230,y: 30}, {x: 250,y: 60}, view), [1])
  assert.deepEqual(marqueeNodeIds(nodes, {x: 270,y: 0}, {x: 290,y: 40}, view), [])
  assert.deepEqual(marqueeNodeIds(nodes, {x: 5,y: 5}, {x: 6,y: 6}, view), [])
})
