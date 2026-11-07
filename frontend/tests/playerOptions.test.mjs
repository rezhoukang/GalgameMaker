import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { runInNewContext } from 'node:vm'
import test from 'node:test'

const template = readFileSync(new URL('../../backend/internal/service/templates/player.html', import.meta.url), 'utf8')
const afterContent = template.slice(template.indexOf('function afterContent(){'), template.indexOf("nextBtn.addEventListener('click'"))

function play(options) {
  const buttons = []
  const targets = []
  const context = {
    completed: false, cur: 1, DATA: { 1: { options } },
    nextBtn: { style: { display: 'block' } },
    optsBox: {
      style: { display: 'flex' },
      replaceChildren() { buttons.length = 0 },
      appendChild(button) { buttons.push(button) }
    },
    document: { createElement() { return {} } },
    navigate(target) { targets.push(target) }
  }
  runInNewContext(afterContent + '\nafterContent();', context)
  return { buttons, targets, context }
}

test('multiple exits show only visible choices and keep the correct destination', () => {
  const result = play([
    { label: 'hidden', target: 2, isShow: false },
    { label: 'visible', target: 3, isShow: true }
  ])
  assert.deepEqual(result.buttons.map(b => b.textContent), ['visible'])
  assert.deepEqual(result.targets, [])
  result.buttons[0].onclick()
  assert.deepEqual(result.targets, [3])
})

test('all hidden multiple exits stay on the current content', () => {
  const result = play([{ target: 2, isShow: false }, { target: 3, isShow: false }])
  assert.equal(result.buttons.length, 0)
  assert.deepEqual(result.targets, [])
  assert.equal(result.context.optsBox.style.display, 'none')
  assert.equal(result.context.nextBtn.style.display, 'none')
})

test('a hidden single exit still continues automatically, once', () => {
  const result = play([{ target: 2, isShow: false }])
  assert.deepEqual(result.targets, [2])
  runInNewContext(afterContent + '\nafterContent();', result.context)
  assert.deepEqual(result.targets, [2])
})

test('a visible single exit remains a choice; terminal nodes have none', () => {
  assert.equal(play([{ label: 'continue', target: 2, isShow: true }]).buttons.length, 1)
  const terminal = play([])
  assert.equal(terminal.buttons.length, 0)
  assert.deepEqual(terminal.targets, [])
})
