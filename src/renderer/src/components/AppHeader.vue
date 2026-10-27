<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/appStore'

const appStore = useAppStore()
const router = useRouter()
const route = useRoute()
const mac = window.electron?.process.platform === 'darwin'

const nav = [
  { name: 'chat', label: '对话' },
  { name: 'agents', label: '智能体' },
  { name: 'settings', label: '设置' }
] as const

const items = computed(() =>
  nav.map((item) => ({
    ...item,
    registered: route.name === item.name || router.hasRoute(item.name)
  }))
)

const chatRoutes = new Set(['chat', 'schedules', 'schedule-detail'])

function itemActive(name: (typeof nav)[number]['name']): boolean {
  if (name === 'chat') return chatRoutes.has(String(route.name))
  return route.name === name
}

function onUnavailable(name: (typeof nav)[number]['name'], event: MouseEvent): void {
  if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) return
  event.preventDefault()
  if (!router.hasRoute(name)) return
  void router.push({ name })
}
</script>

<template>
  <header
    class="window-drag ec-skin flex h-12 w-full shrink-0 items-center gap-6 border-b border-[var(--ec-border)] text-sm text-[var(--ec-fg)]"
    :class="mac ? 'pr-5 pl-[88px]' : 'px-5'"
  >
    <span class="shrink-0 font-medium tracking-tight">{{ appStore.title }}</span>
    <nav class="window-no-drag flex items-center gap-1">
      <template v-for="item in items" :key="item.name">
        <router-link
          v-if="item.registered"
          v-slot="{ href, navigate }"
          :to="{ name: item.name }"
          custom
        >
          <a
            :href="href"
            class="rounded-md px-2.5 py-1 focus-visible:[outline:2px_solid_var(--ec-accent)] focus-visible:[outline-offset:2px]"
            :class="
              itemActive(item.name)
                ? 'bg-[color-mix(in_srgb,var(--ec-accent)_16%,transparent)] text-[var(--ec-accent)]'
                : 'text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)]'
            "
            :aria-current="itemActive(item.name) ? 'page' : undefined"
            @click="navigate"
          >
            {{ item.label }}
          </a>
        </router-link>
        <a
          v-else
          href="#"
          class="rounded-md px-2.5 py-1 text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] focus-visible:[outline:2px_solid_var(--ec-accent)] focus-visible:[outline-offset:2px]"
          @click="onUnavailable(item.name, $event)"
        >
          {{ item.label }}
        </a>
      </template>
    </nav>
  </header>
</template>

<style scoped>
.window-drag {
  -webkit-app-region: drag;
  user-select: none;
}

.window-no-drag {
  -webkit-app-region: no-drag;
}
</style>
