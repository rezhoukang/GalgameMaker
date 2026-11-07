import assert from 'node:assert/strict'
import test from 'node:test'
import { createBackdropClose } from '../src/backdropClose.ts'

function setup() {
  const mask = {}, input = {}
  let closed = 0
  const events = createBackdropClose(() => closed++)
  const event = (outside, button = 0) => ({ target: outside ? mask : input, currentTarget: mask, button })
  return { events, event, count: () => closed }
}

test('text selection released on backdrop does not close; next backdrop click does', () => {
  const { events, event, count } = setup()
  events.mousedown(event(false))
  events.mouseup(event(true))
  events.click(event(true)) // 浏览器会把跨元素点击派发到共同祖先遮罩。
  assert.equal(count(), 0)
  events.mousedown(event(true))
  events.mouseup(event(true))
  events.click(event(true))
  assert.equal(count(), 1)
})

test('both endpoints must be on backdrop and use left button', () => {
  for (const [down, up, button] of [[true, false, 0], [false, false, 0], [true, true, 1], [true, true, 2]]) {
    const { events, event, count } = setup()
    events.mousedown(event(down, button))
    events.mouseup(event(up, button))
    events.click(event(true, button))
    assert.equal(count(), 0)
  }
})

test('incomplete, cancelled, and repeated clicks cannot dismiss', () => {
  const { events, event, count } = setup()
  events.click(event(true))
  events.mousedown(event(true))
  events.click(event(true))
  events.mousedown(event(true))
  events.mouseup(event(true))
  events.reset()
  events.click(event(true))
  assert.equal(count(), 0)
  events.mousedown(event(true))
  events.mouseup(event(true))
  events.click(event(true))
  events.click(event(true))
  assert.equal(count(), 1)
})
