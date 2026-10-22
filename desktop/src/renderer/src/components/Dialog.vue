<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useSlots, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    open: boolean
    title?: string
    subtitle?: string
    /** CSS 长度，例如 420px */
    width?: string
    /** 为真时忽略遮罩、Esc 和关闭按钮 */
    persistent?: boolean
    /** 点击遮罩是否关闭，默认 true */
    closeOnBackdrop?: boolean
  }>(),
  {
    title: '',
    subtitle: '',
    width: '480px',
    persistent: false,
    closeOnBackdrop: true,
  },
)

const emit = defineEmits<{
  'update:open': [value: boolean]
  close: []
}>()

const slots = useSlots()
const panelRef = ref<HTMLElement | null>(null)
const layer = ref(1000)
const uid = ++nextUid
const titleId = `cz-dialog-title-${uid}`
let held = false

const hasHeading = computed(() => !!props.title || !!slots.title)
const showHeader = computed(() => hasHeading.value || !props.persistent)
const panelStyle = computed(() => ({
  width: `min(${props.width || '480px'}, 100%)`,
}))

function requestClose() {
  if (props.persistent) return
  emit('update:open', false)
  emit('close')
}

function onBackdrop() {
  if (!props.closeOnBackdrop) return
  requestClose()
}

function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape' || props.persistent) return
  if (openedStack[openedStack.length - 1] !== uid) return
  event.preventDefault()
  event.stopPropagation()
  requestClose()
}

function acquire() {
  if (held) return
  held = true
  lockScroll()
  openedStack.push(uid)
  layer.value = 40 + openedStack.length - 1
  window.addEventListener('keydown', onKeydown)
}

function release() {
  if (!held) return
  held = false
  unlockScroll()
  const index = openedStack.lastIndexOf(uid)
  if (index >= 0) openedStack.splice(index, 1)
  window.removeEventListener('keydown', onKeydown)
}

async function focusPanel() {
  await nextTick()
  const panel = panelRef.value
  if (!panel || panel.contains(document.activeElement)) return
  panel.focus()
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      acquire()
      void focusPanel()
      return
    }
    release()
  },
  { immediate: true },
)

onBeforeUnmount(release)
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="overlay" :style="{ zIndex: layer }">
      <div class="backdrop" @click="onBackdrop" />
      <div
        ref="panelRef"
        class="panel"
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        :style="panelStyle"
        :aria-labelledby="hasHeading ? titleId : undefined"
        :aria-label="hasHeading ? undefined : '对话框'"
        @click.stop
      >
        <header v-if="showHeader" class="head">
          <div v-if="hasHeading" class="title-col">
            <h3 :id="titleId" class="title">
              <slot name="title">{{ title }}</slot>
            </h3>
            <p v-if="subtitle" class="sub">{{ subtitle }}</p>
          </div>
          <button
            v-if="!persistent"
            type="button"
            class="close"
            aria-label="关闭"
            @click="requestClose"
          >
            ×
          </button>
        </header>
        <div class="body" :class="{ bare: !showHeader, end: !slots.footer }">
          <slot />
        </div>
        <footer v-if="slots.footer" class="foot">
          <slot name="footer" />
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<script lang="ts">
let lockCount = 0
let bodyOverflow = ''
const openedStack: number[] = []
let nextUid = 0

function lockScroll() {
  if (lockCount === 0) {
    bodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  }
  lockCount += 1
}

function unlockScroll() {
  lockCount = Math.max(0, lockCount - 1)
  if (lockCount === 0) document.body.style.overflow = bodyOverflow
}
</script>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  display: grid;
  align-items: center;
  justify-items: center;
  align-items: safe center;
  justify-items: safe center;
  padding: 24px;
  overflow: auto;
  -webkit-app-region: no-drag;
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
  min-height: 0;
  max-height: min(80vh, calc(100vh - 48px));
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
  overflow: hidden;
  outline: none;
  -webkit-app-region: no-drag;
}

.head {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 14px 16px 0;
  flex-shrink: 0;
}

.title-col {
  flex: 1;
  min-width: 0;
}

.title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}

.sub {
  margin: 4px 0 0;
  font-size: 12px;
  font-weight: 400;
  line-height: 1.5;
  color: var(--text-muted);
}

.close {
  margin-left: auto;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font-size: 20px;
  line-height: 1;
  cursor: pointer;
  -webkit-app-region: no-drag;
}

.close:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.body {
  min-height: 0;
  overflow: auto;
  padding: 12px 16px;
  color: var(--text);
  font-size: 13px;
  line-height: 1.5;
}

.body.bare {
  padding-top: 16px;
}

.body.end {
  padding-bottom: 16px;
}

.foot {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 4px 16px 14px;
  flex-shrink: 0;
}

.foot :slotted(:not(button)) {
  margin-right: auto;
}

.foot :slotted(button) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 0;
  padding: 6px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: 13px;
  line-height: 1.2;
  cursor: pointer;
}

.foot :slotted(button:hover:not(:disabled)) {
  background: var(--surface-hover);
}

.foot :slotted(button.primary) {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}

.foot :slotted(button.primary:hover:not(:disabled)) {
  background: var(--accent-hover);
  border-color: var(--accent-hover);
  color: #fff;
}

.foot :slotted(button.danger) {
  background: transparent;
  border-color: transparent;
  color: var(--error);
}

.foot :slotted(button.danger:hover:not(:disabled)) {
  background: color-mix(in srgb, var(--error) 12%, transparent);
  color: var(--error);
}

.foot :slotted(button:disabled) {
  opacity: 0.45;
  cursor: not-allowed;
}
</style>
