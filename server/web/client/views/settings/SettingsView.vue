<script setup lang="ts">
import { RouterLink, RouterView, useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <div class="settings">
    <aside>
      <p class="aside-label">设置</p>
      <nav>
        <RouterLink :to="{ name: 'settings-link', query: route.query }">
          <strong>接入</strong>
          <span>客户端标识</span>
        </RouterLink>
        <RouterLink :to="{ name: 'settings-models', query: route.query }">
          <strong>模型配置</strong>
          <span>供应商与大模型</span>
        </RouterLink>
        <RouterLink :to="{ name: 'settings-usage', query: route.query }">
          <strong>用量统计</strong>
          <span>Token 与成本</span>
        </RouterLink>
        <RouterLink :to="{ name: 'settings-appearance', query: route.query }">
          <strong>外观</strong>
          <span>亮色、暗色与跟随系统</span>
        </RouterLink>
      </nav>
    </aside>
    <section>
      <RouterView />
    </section>
  </div>
</template>

<style scoped>
.settings {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 16px;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 20px 24px 28px;
}

aside {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  gap: 8px;
}

.aside-label {
  margin: 0;
  flex-shrink: 0;
  padding: 0 4px;
  color: var(--text-dim);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.04em;
}

nav {
  display: grid;
  gap: 4px;
  align-content: start;
  overflow: auto;
}

nav a {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  text-decoration: none;
  color: inherit;
  transition: background 0.12s;
}

nav a:hover {
  background: var(--surface-hover);
}

nav a.router-link-active {
  background: var(--accent-soft);
}

nav a.router-link-active strong {
  color: var(--accent);
}

nav strong {
  font-size: 13px;
  font-weight: 560;
  line-height: 1.25;
}

nav span {
  font-size: 11px;
  line-height: 1.35;
  color: var(--text-muted);
}

section {
  min-width: 0;
  min-height: 0;
  overflow: auto;
}
</style>
