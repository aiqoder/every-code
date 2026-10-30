<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

function tabClass(active: boolean): string {
  return active
    ? 'border-[var(--ec-accent)] font-medium'
    : 'border-transparent text-[var(--ec-fg-muted)] hover:text-[var(--ec-fg)]'
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col text-[var(--ec-fg)]">
    <div class="window-drag flex shrink-0 items-center border-b border-[var(--ec-border)] px-5">
      <div class="flex gap-1" role="tablist" aria-label="设置">
        <button
          type="button"
          role="tab"
          class="border-b-2 px-3 py-3 text-sm"
          :class="tabClass(route.name === 'settings')"
          :aria-selected="route.name === 'settings'"
          @click="router.push({ name: 'settings' })"
        >
          通用
        </button>
        <button
          type="button"
          role="tab"
          class="border-b-2 px-3 py-3 text-sm"
          :class="tabClass(route.name === 'settings-models')"
          :aria-selected="route.name === 'settings-models'"
          @click="router.push({ name: 'settings-models' })"
        >
          模型
        </button>
      </div>
    </div>
    <div class="min-h-0 flex-1">
      <router-view class="h-full" />
    </div>
  </div>
</template>
