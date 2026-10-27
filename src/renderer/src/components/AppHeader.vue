<script setup lang="ts">
import { useAppStore } from '@/stores/appStore'

const appStore = useAppStore()
</script>

<template>
  <header
    class="flex h-12 shrink-0 items-center border-b border-[var(--ec-border)] bg-[var(--ec-canvas)] px-5 text-[var(--ec-fg)]"
  >
    <span class="text-sm font-medium tracking-tight">{{ appStore.title }}</span>
  </header>
</template>
