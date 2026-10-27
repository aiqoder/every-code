<script setup lang="ts">
import { NRadioButton, NRadioGroup } from 'naive-ui'
import { themeList, isThemeId } from '@/theme/github'
import { useThemeStore } from '@/stores/themeStore'

const themeStore = useThemeStore()

function onChange(value: string | number | boolean | null): void {
  if (typeof value === 'string' && isThemeId(value)) themeStore.setTheme(value)
}
</script>

<template>
  <n-radio-group :value="themeStore.current" size="small" @update:value="onChange">
    <n-radio-button v-for="item in themeList" :key="item.id" :value="item.id">
      {{ item.label }}
    </n-radio-button>
  </n-radio-group>
</template>
