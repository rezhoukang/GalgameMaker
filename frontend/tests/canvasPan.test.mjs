import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { runInNewContext } from 'node:vm'
import test from 'node:test'
import ts from 'typescript'
import { marqueeNodeIds } from '../src/marqueeSelection.ts'

const component = readFileSync(new URL('../src/components/CanvasArea.vue', import.meta.url), 'utf8')
const handlers = component.slice(component.indexOf('function selectionPoint('), component.indexOf('onMounted(() =>'))
const code = ts.transpileModule('let suppressPanClick = false;\n' + handlers, {
  compilerOptions: { target: ts.ScriptTarget.ES2022 }
}).outputText

function setup() {
  const calls = []
  const store = {
    isCanvasView: true, interaction: null,
    selectedSceneIds: [9], view: { x: 20, y: 10, zoom: 0.5 },
    selectScenes(ids) { this.selectedSceneIds = ids },
    beginPan(x, y) { this.interaction = { type: 'pan' }; calls.push(['start', x, y]) },
    movePan(x, y) { calls.push(['move', x, y]) },
    endPan() { this.interaction = null }
  }
  const context = {
    store, spaceDown: { value: false }, qDown: { value: false }, marquee: { value: null }, marqueeNodeIds,
    visibleScenes: { value: [{ id: 1, x: 0, y: 0 }, { id: 2, x: 0, y: 68 }, { id: 3, x: 480, y: 0 }] },
    viewportRef: { value: { getBoundingClientRect: () => ({left: 100, top: 50, width: 800, height: 600}) } },
    document: { querySelector: () => null }
  }
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

test('Q drag selects visible cards at current pan and zoom, blocks child clicks and survives early Q release', () => {
  const c = setup()
  c.onKey(event({ code: 'KeyQ' }))
  const down = event({ clientX: 110, clientY: 55 })
  c.onViewDown(down)
  assert.ok(down.prevented && down.stopped)
  c.onWindowMove(event({ clientX: 245, clientY: 120 }))
  assert.deepEqual(c.store.selectedSceneIds, [1, 2])
  assert.equal(c.calls.length, 0)
  c.onKeyUp(event({ code: 'KeyQ' }))
  c.onWindowUp(event({ clientX: 245, clientY: 120 }))
  assert.equal(c.marquee.value, null)
  assert.deepEqual(c.store.selectedSceneIds, [1, 2])
  const click = event()
  c.onViewAction(click)
  assert.ok(click.stopped)
})

test('Escape cancels a live selection and restores the previous selection', () => {
  const c = setup()
  c.onKey(event({ code: 'KeyQ' }))
  c.onViewDown(event({ clientX: 110, clientY: 55 }))
  c.onWindowMove(event({ clientX: 245, clientY: 120 }))
  c.onKey(event({ key: 'Escape' }))
  assert.deepEqual([...c.store.selectedSceneIds], [9])
  assert.equal(c.marquee.value, null)
  assert.equal(c.qDown.value, false)
})

test('Q remains ordinary text in inputs, dialogs and modified key combinations', () => {
  const c = setup()
  c.onKey(event({ code: 'KeyQ', target: { closest: () => ({}) } }))
  assert.equal(c.qDown.value, false)
  c.onKey(event({ code: 'KeyQ', key: 'q', ctrlKey: true }))
  assert.equal(c.qDown.value, false)
  c.document.querySelector = () => ({})
  c.onKey(event({ code: 'KeyQ' }))
  assert.equal(c.qDown.value, false)
})

test('Space pan takes priority over Q and window blur ends selection mode', () => {
  const c = setup()
  c.onKey(event({ code: 'KeyQ' }))
  c.onKey(event({ code: 'Space' }))
  c.onViewDown(event())
  assert.equal(c.store.interaction.type, 'pan')
  assert.equal(c.marquee.value, null)
  c.onBlur()
  assert.equal(c.qDown.value, false)
  assert.equal(c.spaceDown.value, false)
})
