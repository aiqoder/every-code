<script setup lang="ts">
import { useThemeStore, type ThemePreference } from '@client/stores/theme'

const theme = useThemeStore()

const options: { value: ThemePreference; label: string }[] = [
  { value: 'system', label: '系统' },
  { value: 'light', label: '亮色' },
  { value: 'dark', label: '暗色' },
]
</script>

<template>
  <el-segmented
    :model-value="theme.preference"
    :options="options"
    size="small"
    @change="(v: ThemePreference) => theme.setPreference(v)"
  />
</template>
