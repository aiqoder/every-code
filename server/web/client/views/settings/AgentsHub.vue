<script setup lang="ts">
import { RouterLink, RouterView, useRoute } from 'vue-router'

const route = useRoute()

const tabs = [
  { name: 'settings-agents', label: '已安装' },
]
</script>

<template>
  <div class="agents-hub">
    <nav class="sub" aria-label="智能体">
      <RouterLink
        v-for="tab in tabs"
        :key="tab.name"
        :to="{ name: tab.name, query: route.query }"
      >
        {{ tab.label }}
      </RouterLink>
    </nav>
    <RouterView />
  </div>
</template>

<style scoped>
.agents-hub {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.sub {
  display: flex;
  gap: 16px;
  width: min(960px, 100%);
  margin: 4px auto 0;
  padding: 0 24px;
}

.sub a {
  color: var(--text-muted);
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
  padding: 8px 0 6px;
  border-bottom: 2px solid transparent;
}

.sub a.router-link-exact-active {
  color: var(--text);
  border-bottom-color: var(--text);
}
</style>
