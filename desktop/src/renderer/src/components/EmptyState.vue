<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  emptyTitle?: string
  emptyHint?: string
  compact?: boolean
}>()

const shown = computed(() => !props.emptyTitle)
</script>

<template>
  <div class="empty" :class="{ compact }">
    <div class="mark" aria-hidden="true">
      <svg viewBox="0 0 48 48" width="34" height="34" fill="none" stroke="currentColor">
        <rect x="6" y="9" width="36" height="30" rx="3" stroke-width="1.2" opacity="0.5" />
        <path d="M6 18h36" stroke-width="1.2" opacity="0.5" />
        <circle cx="11" cy="13.5" r="1.3" fill="currentColor" opacity="0.6" />
        <path d="M14 25h20M14 30h13" stroke-width="1.2" opacity="0.35" />
      </svg>
    </div>
    <p class="title serif">{{ emptyTitle }}</p>
    <p v-if="emptyHint" class="hint">{{ emptyHint }}</p>
    <slot v-if="!shown" />
  </div>
</template>

<style scoped>
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 56px 24px;
  text-align: center;
  color: var(--text-faint);
}
.empty.compact {
  padding: 26px 16px;
  gap: 6px;
}
.mark {
  color: var(--ink-500);
  margin-bottom: 2px;
}
.title {
  font-size: 15px;
  color: var(--text-dim);
}
.hint {
  font-size: 12.5px;
  max-width: 34em;
  line-height: 1.7;
  color: var(--text-faint);
}
</style>
