<script lang="ts">
let lockCount = 0
let bodyOverflow = ''
const openedStack: number[] = []
let nextUid = 0

function allocateUid(): number {
  nextUid += 1
  return nextUid
}

function lockScroll(): void {
  if (lockCount === 0) {
    bodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  }
  lockCount += 1
}

function unlockScroll(): void {
  lockCount = Math.max(0, lockCount - 1)
  if (lockCount === 0) document.body.style.overflow = bodyOverflow
}
</script>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    show: boolean
    title?: string
    width?: string
    maskClosable?: boolean
    closeOnEsc?: boolean
  }>(),
  {
    title: '',
    width: '480px',
    maskClosable: true,
    closeOnEsc: true
  }
)

const emit = defineEmits<{
  'update:show': [value: boolean]
}>()

defineOptions({ inheritAttrs: false })

const panelRef = ref<HTMLElement | null>(null)
const layer = ref(1000)
const uid = allocateUid()
const titleId = `ec-dialog-title-${uid}`
let held = false

const hasTitle = computed(() => props.title.trim().length > 0)
const panelStyle = computed(() => ({
  width: `min(${props.width}, calc(100% - 2rem))`
}))

function requestClose(): void {
  emit('update:show', false)
}

function onBackdrop(): void {
  if (!props.maskClosable) return
  requestClose()
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Escape' || !props.closeOnEsc) return
  if (openedStack[openedStack.length - 1] !== uid) return
  event.preventDefault()
  event.stopPropagation()
  requestClose()
}

function acquire(): void {
  if (held) return
  held = true
  lockScroll()
  openedStack.push(uid)
  layer.value = 1000 + openedStack.length - 1
  window.addEventListener('keydown', onKeydown)
}

function release(): void {
  if (!held) return
  held = false
  unlockScroll()
  const index = openedStack.lastIndexOf(uid)
  if (index >= 0) openedStack.splice(index, 1)
  window.removeEventListener('keydown', onKeydown)
}

async function focusPanel(): Promise<void> {
  await nextTick()
  const panel = panelRef.value
  if (!panel || panel.contains(document.activeElement)) return
  panel.focus()
}

watch(
  () => props.show,
  (show) => {
    if (show) {
      acquire()
      void focusPanel()
      return
    }
    release()
  },
  { immediate: true }
)

onBeforeUnmount(release)
</script>

<template>
  <Teleport to="body">
    <div v-if="show" class="overlay" :style="{ zIndex: layer }" v-bind="$attrs">
      <div class="backdrop" @click="onBackdrop" />
      <div
        ref="panelRef"
        class="panel ec-stage"
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        :style="panelStyle"
        :aria-labelledby="hasTitle ? titleId : undefined"
        :aria-label="hasTitle ? undefined : '对话框'"
        @click.stop
      >
        <header v-if="hasTitle" class="head">
          <h3 :id="titleId" class="title">{{ title }}</h3>
          <button type="button" class="close" aria-label="关闭" @click="requestClose">×</button>
        </header>
        <div class="body" :class="{ bare: !hasTitle }">
          <slot />
        </div>
        <footer v-if="$slots.footer" class="foot">
          <slot name="footer" />
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  display: grid;
  place-items: center;
}

.backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
}

.panel {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  max-height: min(80vh, calc(100vh - 2rem));
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
  overflow: hidden;
  outline: none;
}

.head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  padding: 14px 16px 0;
}

.title {
  flex: 1;
  min-width: 0;
  margin: 0;
  color: var(--text);
  font-size: var(--ec-text-title);
  font-weight: 600;
  line-height: 1.4;
}

.close {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  margin-left: auto;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font-size: var(--ec-text-title);
  line-height: 1;
  cursor: pointer;
}

.close:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.body {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 12px 16px 16px;
  color: var(--text);
  font-size: var(--ec-text-body);
  line-height: 1.57;
}

.body.bare {
  padding-top: 16px;
}

.foot {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-shrink: 0;
  padding: 0 16px 16px;
}
</style>
