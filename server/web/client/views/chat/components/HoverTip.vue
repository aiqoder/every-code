<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps<{
  content: string
  /** 仅在内容被省略或强制显示时展示；默认有 content 即显示 */
  disabled?: boolean
}>()

const open = ref(false)
const tipStyle = ref<Record<string, string>>({})
const triggerRef = ref<HTMLElement | null>(null)
let hideTimer: ReturnType<typeof setTimeout> | undefined

const showable = computed(() => !props.disabled && !!props.content?.trim())

function clearHide() {
  if (hideTimer) {
    clearTimeout(hideTimer)
    hideTimer = undefined
  }
}

async function place() {
  const el = triggerRef.value
  if (!el) return
  await nextTick()
  const rect = el.getBoundingClientRect()
  const tipW = 260
  const gap = 8
  let left = rect.right + gap
  if (left + tipW > window.innerWidth - 12) {
    left = Math.max(12, rect.left - tipW - gap)
  }
  let top = rect.top + rect.height / 2
  tipStyle.value = {
    left: `${left}px`,
    top: `${top}px`,
    transform: 'translateY(-50%)',
    maxWidth: `${tipW}px`,
  }
}

async function onEnter() {
  if (!showable.value) return
  clearHide()
  open.value = true
  await place()
}

function onLeave() {
  clearHide()
  hideTimer = setTimeout(() => {
    open.value = false
  }, 80)
}

function onTipEnter() {
  clearHide()
}

function onTipLeave() {
  onLeave()
}

watch(
  () => props.content,
  () => {
    if (open.value) place()
  },
)

onBeforeUnmount(() => {
  clearHide()
})
</script>

<template>
  <span
    ref="triggerRef"
    class="tip-trigger"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
  >
    <slot />
  </span>
  <Teleport to="body">
    <div
      v-if="open && showable"
      class="cz-tooltip"
      :style="tipStyle"
      role="tooltip"
      @mouseenter="onTipEnter"
      @mouseleave="onTipLeave"
    >
      {{ content }}
    </div>
  </Teleport>
</template>

<style scoped>
.tip-trigger {
  display: block;
  min-width: 0;
  width: 100%;
}
</style>

<style>
.cz-tooltip {
  position: fixed;
  z-index: 4000;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid var(--border-strong);
  background: var(--bg-elevated);
  color: var(--text);
  font-size: 12px;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-word;
  box-shadow: var(--shadow);
  pointer-events: auto;
}
</style>
