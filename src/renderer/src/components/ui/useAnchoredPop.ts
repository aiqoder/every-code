import { nextTick, onBeforeUnmount, ref, watch, type CSSProperties, type Ref } from 'vue'

export function useAnchoredPop(
  open: Ref<boolean>,
  anchor: Ref<HTMLElement | null>,
  pop: Ref<HTMLElement | null>,
  matchWidth: () => boolean,
  limits: { maxHeight?: number; maxWidth?: number } = {}
): { style: Ref<CSSProperties>; prepare: () => void } {
  const capHeight = limits.maxHeight ?? 280
  const capWidth = limits.maxWidth ?? 320
  const style = ref<CSSProperties>({
    position: 'fixed',
    zIndex: 4000
  })
  let listening = false
  let observer: ResizeObserver | null = null

  function place(): void {
    const trigger = anchor.value
    if (!trigger) return
    const rect = trigger.getBoundingClientRect()
    const node = pop.value
    const margin = 8
    const gap = 6
    const height = node?.offsetHeight ?? 0
    const naturalWidth = node?.scrollWidth ?? rect.width
    const width = matchWidth() ? rect.width : Math.min(Math.max(rect.width, naturalWidth), capWidth)
    const spaceBelow = window.innerHeight - rect.bottom - margin
    const spaceAbove = rect.top - margin
    const openUp = height > 0 && spaceBelow < height && spaceAbove > spaceBelow
    const available = Math.max(120, openUp ? spaceAbove - gap : spaceBelow - gap)
    let top = openUp ? rect.top - gap - height : rect.bottom + gap
    const maxTop = Math.max(margin, window.innerHeight - margin - height)
    top = Math.min(Math.max(margin, top), maxTop)
    let left = rect.left
    if (left + width > window.innerWidth - margin) left = window.innerWidth - margin - width
    left = Math.max(margin, left)
    style.value = {
      position: 'fixed',
      top: `${Math.round(top)}px`,
      left: `${Math.round(left)}px`,
      minWidth: `${Math.round(rect.width)}px`,
      width: matchWidth() ? `${Math.round(rect.width)}px` : 'max-content',
      maxWidth: matchWidth() ? undefined : `min(${capWidth}px, calc(100vw - 16px))`,
      maxHeight: `${Math.min(capHeight, Math.round(available))}px`,
      zIndex: 4000
    }
  }

  function prepare(): void {
    const trigger = anchor.value
    if (!trigger) return
    const rect = trigger.getBoundingClientRect()
    style.value = {
      position: 'fixed',
      top: `${Math.round(rect.bottom + 6)}px`,
      left: `${Math.round(rect.left)}px`,
      minWidth: `${Math.round(rect.width)}px`,
      width: matchWidth() ? `${Math.round(rect.width)}px` : 'max-content',
      maxWidth: matchWidth() ? undefined : `min(${capWidth}px, calc(100vw - 16px))`,
      maxHeight: `${capHeight}px`,
      zIndex: 4000
    }
  }

  function onKeydown(event: KeyboardEvent): void {
    if (event.key !== 'Escape' || !open.value) return
    event.preventDefault()
    event.stopPropagation()
    open.value = false
  }

  function bind(): void {
    if (listening) return
    listening = true
    window.addEventListener('scroll', place, true)
    window.addEventListener('resize', place)
    window.addEventListener('keydown', onKeydown, true)
    if (pop.value) {
      observer = new ResizeObserver(() => place())
      observer.observe(pop.value)
    }
  }

  function unbind(): void {
    if (!listening) return
    listening = false
    observer?.disconnect()
    observer = null
    window.removeEventListener('scroll', place, true)
    window.removeEventListener('resize', place)
    window.removeEventListener('keydown', onKeydown, true)
  }

  watch(
    open,
    async (value) => {
      if (!value) {
        unbind()
        return
      }
      await nextTick()
      place()
      await nextTick()
      place()
      bind()
    },
    { flush: 'post' }
  )

  onBeforeUnmount(unbind)

  return { style, prepare }
}
