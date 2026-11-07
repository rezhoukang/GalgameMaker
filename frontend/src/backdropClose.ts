import type { ObjectDirective } from 'vue'

// 等 click 再关闭，避免在 mouseup 时移除遮罩，让后续点击落到画布上。
export function createBackdropClose(close: () => void) {
  let pressedOutside = false
  let releasedOutside = false
  const reset = () => { pressedOutside = releasedOutside = false }
  return {
    reset,
    mousedown(event: MouseEvent) {
      reset()
      pressedOutside = event.button === 0 && event.target === event.currentTarget
    },
    mouseup(event: MouseEvent) {
      releasedOutside = pressedOutside && event.button === 0 && event.target === event.currentTarget
    },
    click(event: MouseEvent) {
      const dismiss = pressedOutside && releasedOutside && event.button === 0 && event.target === event.currentTarget
      reset()
      if (dismiss) close()
    }
  }
}

const bindings = new WeakMap<HTMLElement, { close: () => void; dispose: () => void }>()

export const vBackdropClose: ObjectDirective<HTMLElement, () => void> = {
  mounted(element, binding) {
    const state = { close: binding.value, dispose: () => {} }
    const handlers = createBackdropClose(() => state.close())
    const view = element.ownerDocument.defaultView
    // 捕获阶段记录起止位置，子控件的 stopPropagation 不影响判断。
    element.addEventListener('mousedown', handlers.mousedown, true)
    element.addEventListener('mouseup', handlers.mouseup, true)
    element.addEventListener('click', handlers.click, true)
    element.addEventListener('pointercancel', handlers.reset, true)
    view?.addEventListener('blur', handlers.reset)
    state.dispose = () => {
      element.removeEventListener('mousedown', handlers.mousedown, true)
      element.removeEventListener('mouseup', handlers.mouseup, true)
      element.removeEventListener('click', handlers.click, true)
      element.removeEventListener('pointercancel', handlers.reset, true)
      view?.removeEventListener('blur', handlers.reset)
    }
    bindings.set(element, state)
  },
  updated(element, binding) {
    const state = bindings.get(element)
    if (state) state.close = binding.value
  },
  unmounted(element) {
    bindings.get(element)?.dispose()
    bindings.delete(element)
  }
}
