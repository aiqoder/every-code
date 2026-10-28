<script setup lang="ts">
import { appearanceList, isAppearanceId } from '@/theme/github'
import { useThemeStore } from '@/stores/themeStore'

const themeStore = useThemeStore()

function onChange(value: string): void {
  if (isAppearanceId(value)) themeStore.setTheme(value)
}
</script>

<template>
  <div class="theme-switch" role="radiogroup" aria-label="外观">
    <button
      v-for="item in appearanceList"
      :key="item.id"
      type="button"
      class="theme-option"
      role="radio"
      :aria-checked="themeStore.current === item.id"
      :class="{ 'is-current': themeStore.current === item.id }"
      @click="onChange(item.id)"
    >
      {{ item.label }}
    </button>
  </div>
</template>

<style scoped>
.theme-switch {
  display: inline-flex;
  align-items: stretch;
  border: 1px solid var(--ec-border);
  border-radius: 6px;
  overflow: hidden;
}

.theme-option {
  margin: 0;
  border: 0;
  border-right: 1px solid var(--ec-border);
  background: transparent;
  color: var(--ec-fg-muted);
  padding: 4px 10px;
  font-size: var(--ec-text-meta);
  line-height: 16px;
  cursor: pointer;
}

.theme-option:last-child {
  border-right: 0;
}

.theme-option:hover:not(.is-current) {
  background: var(--surface-hover);
  color: var(--ec-fg);
}

.theme-option.is-current {
  background: color-mix(in srgb, var(--ec-accent) 18%, transparent);
  color: var(--ec-accent);
}

.theme-option:focus-visible {
  outline: 2px solid var(--ec-accent);
  outline-offset: -2px;
}
</style>
