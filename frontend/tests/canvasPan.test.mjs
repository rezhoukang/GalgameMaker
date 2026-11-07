import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { runInNewContext } from 'node:vm'
import test from 'node:test'
import ts from 'typescript'

const component = readFileSync(new URL('../src/components/CanvasArea.vue', import.meta.url), 'utf8')
const handlers = component.slice(component.indexOf('function onViewDown('), component.indexOf('onMounted(() =>'))
const code = ts.transpileModule('let suppressPanClick = false;\n' + handlers, {
  compilerOptions: { target: ts.ScriptTarget.ES2022 }
}).outputText

function setup() {
  const calls = []
  const store = {
    isCanvasView: true, interaction: null,
    beginPan(x, y) { this.interaction = { type: 'pan' }; calls.push(['start', x, y]) },
    movePan(x, y) { calls.push(['move', x, y]) },
    endPan() { this.interaction = null }
  }
  const context = { store, spaceDown: { value: false }, viewportRef: { value: {} }, document: { querySelector: () => null } }
  runInNewContext(code, context)
  return { ...context, calls }
}
function event(props = {}) {
  return {
    type: 'click', button: 0, clientX: 40, clientY: 50,
    target: { closest: () => null },
    prevented: false, stopped: false,
    preventDefault() { this.prevented = true },
    stopPropagation() { this.stopped = true }, ...props
  }
}

test('space begins pan over child controls and blocks the trailing click after space is released', () => {
  const c = setup()
  const key = event({ code: 'Space' })
  c.onKey(key)
  assert.ok(key.prevented && key.stopped)
  const down = event({ type: 'mousedown' })
  c.onViewDown(down)
  assert.ok(down.prevented && down.stopped)
  c.onWindowMove(event({ clientX: 80, clientY: 90 }))
  c.onKeyUp(event({ code: 'Space' }))
  c.onWindowUp()
  const click = event()
  c.onViewAction(click)
  assert.ok(click.prevented && click.stopped)
  assert.deepEqual(c.calls, [['start', 40, 50], ['move', 80, 90]])
  assert.equal(c.store.interaction, null)
  const nextDown = event({ type: 'mousedown' })
  c.onViewDown(nextDown)
  const nextClick = event()
  c.onViewAction(nextClick)
  assert.equal(nextClick.stopped, false)
})

test('holding space suppresses clicks, double clicks and context menus', () => {
  const c = setup()
  c.onKey(event({ code: 'Space' }))
  for (const type of ['click', 'dblclick', 'contextmenu']) {
    const e = event({ type })
    c.onViewAction(e)
    assert.ok(e.prevented && e.stopped)
  }
})

test('space in text inputs or a dialog remains ordinary input', () => {
  const c = setup()
  const inputKey = event({ code: 'Space', target: { closest: () => ({}) } })
  c.onKey(inputKey)
  assert.equal(c.spaceDown.value, false)
  assert.equal(inputKey.prevented, false)
  c.document.querySelector = () => ({})
  c.onKey(event({ code: 'Space' }))
  assert.equal(c.spaceDown.value, false)
})

test('window blur releases pan mode', () => {
  const c = setup()
  c.onKey(event({ code: 'Space' }))
  c.onViewDown(event())
  c.onBlur()
  assert.equal(c.spaceDown.value, false)
  assert.equal(c.store.interaction, null)
})
