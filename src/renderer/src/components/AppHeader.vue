<script setup lang="ts">
import { useAppStore } from '@/stores/appStore'

const appStore = useAppStore()
</script>

<template>
  <header class="flex h-12 shrink-0 items-center border-b border-white/10 px-5">
    <span class="text-sm font-medium tracking-tight">{{ appStore.title }}</span>
  </header>
</template>
