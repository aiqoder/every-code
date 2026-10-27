<script setup lang="ts">
import { NSelect, type SelectOption } from 'naive-ui'
import { themeList, isThemeId } from '@/theme/github'
import { useThemeStore } from '@/stores/themeStore'

const themeStore = useThemeStore()

const options: SelectOption[] = themeList.map((item) => ({
  label: item.label,
  value: item.id
}))

function onChange(value: string | number | Array<string | number> | null): void {
  if (typeof value === 'string' && isThemeId(value)) themeStore.setTheme(value)
}
</script>

<template>
  <n-select
    class="theme-switch"
    size="small"
    :value="themeStore.current"
    :options="options"
    @update:value="onChange"
  />
</template>

<style scoped>
.theme-switch {
  width: 12.5rem;
  flex: 0 0 auto;
}
</style>
