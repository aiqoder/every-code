<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import { RouterLink } from 'vue-router'

interface Crumb {
  label: string
  to?: RouteLocationRaw
}

const props = defineProps<{
  items: Crumb[]
}>()

function isLink(item: Crumb, index: number) {
  return !!item.to && index < props.items.length - 1
}
</script>

<template>
  <nav class="crumbs" aria-label="面包屑">
    <span
      v-for="(item, index) in items"
      :key="index"
      class="crumb-slot"
    >
      <span v-if="index > 0" class="sep">/</span>
      <RouterLink v-if="isLink(item, index)" :to="item.to!" class="crumb link">
        {{ item.label }}
      </RouterLink>
      <span v-else class="crumb current">{{ item.label }}</span>
    </span>
  </nav>
</template>

<style scoped>
.crumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  font-size: 13px;
  color: var(--text-dim);
}

.crumb-slot {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.sep {
  color: var(--text-dim);
}

.crumb {
  line-height: 1.4;
}

.crumb.link {
  color: var(--text-muted);
  text-decoration: none;
  border-radius: var(--radius-sm);
  padding: 2px 4px;
  transition: color 0.12s, background 0.12s;
}

.crumb.link:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.crumb.current {
  color: var(--text);
  font-weight: 500;
}
</style>
