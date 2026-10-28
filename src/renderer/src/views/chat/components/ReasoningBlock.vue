<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  content: string
  live: boolean
}>()

const open = ref(false)

watch(
  () => props.live,
  (live) => {
    open.value = live
  },
  { immediate: true }
)

function onToggle(event: Event): void {
  const el = event.currentTarget
  if (!(el instanceof HTMLDetailsElement)) return
  if (el.open === open.value) return
  open.value = el.open
}
</script>

<template>
  <details
    class="ec-skin-inset mb-2 rounded-lg border border-[var(--ec-border)]"
    :open="open"
    @toggle="onToggle"
  >
    <summary
      class="flex cursor-pointer list-none items-center gap-2 rounded-xl px-3 py-2 text-xs text-[var(--ec-fg-muted)] select-none hover:bg-[color-mix(in_srgb,var(--ec-fg)_6%,transparent)] hover:text-[var(--ec-fg)] [&::-webkit-details-marker]:hidden"
    >
      <span
        class="inline-block h-1.5 w-1.5 shrink-0 rounded-full"
        :class="live ? 'animate-pulse bg-[var(--ec-accent)]' : 'bg-[var(--ec-fg-muted)]'"
      />
      <span>{{ live ? '思考中…' : '已深度思考' }}</span>
    </summary>
    <div
      class="max-h-40 overflow-auto px-3 pb-2 text-xs leading-5 whitespace-pre-wrap text-[var(--ec-fg-muted)] select-text"
    >
      {{ content }}
    </div>
  </details>
</template>
