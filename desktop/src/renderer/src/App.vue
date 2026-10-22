<script setup lang="ts">
import { onMounted, computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppRail from './components/AppRail.vue'
import ShellPanel from './components/ShellPanel.vue'
import ToastStack from './components/ToastStack.vue'
import { bridge } from './api'
import { useSessionStore } from './stores/session'
import { useShellStore } from './stores/shell'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const shell = useShellStore()

const menus: Array<{ id: string; label: string; path: string }> = [
  { id: 'project', label: '项目概况', path: '/project' },
  { id: 'create', label: '任务看板', path: '/create' },
  { id: 'execute', label: '执行', path: '/execute' },
  { id: 'test', label: '测试', path: '/test' },
  { id: 'deploy', label: '部署', path: '/deploy' }
]

const navArmed = ref(false)
const bare = computed(() => route.meta.bare === true)
const showRail = computed(() => !bare.value)
const showShell = computed(() => !bare.value)
const channel = computed(() => {
  const path = route.path
  if (path.startsWith('/tasks/')) return 'create'
  return menus.find((item) => path.startsWith(item.path))?.id ?? ''
})
const showProjectChrome = computed(() => Boolean(channel.value) && session.currentProjectId > 0)

onMounted(async () => {
  await session.bootstrap()
  if (session.loggedIn) {
    await session.loadAllTasks()
  }
  shell.subscribe()
  navArmed.value = true
  if (!session.loggedIn && route.path !== '/login') {
    void router.replace('/login')
  } else if (
    session.loggedIn &&
    session.modelGateKnown &&
    !session.modelsConfigured &&
    route.path !== '/models'
  ) {
    void router.replace('/models')
  }
})

watch(
  () => session.loggedIn,
  (on) => {
    if (!navArmed.value || on || route.path === '/login') return
    void router.replace('/login')
  }
)
watch(
  () => session.modelGateKnown && session.modelsConfigured,
  (ok) => {
    if (!navArmed.value || !session.loggedIn || !session.modelGateKnown || ok) return
    if (route.path === '/models' || route.path === '/login') return
    void router.replace('/models')
  }
)

watch(
  () => session.currentProject?.localDir ?? '',
  (dir) => {
    if (session.loggedIn && dir) void bridge.setWorkspace(dir)
  },
  { immediate: true }
)
</script>

<template>
  <div class="app-shell">
    <div class="top-drag" />
    <div v-if="showRail" class="body">
      <AppRail />
      <main class="stage">
        <nav v-if="showProjectChrome" class="project-menu">
          <button
            v-for="item in menus"
            :key="item.id"
            class="menu-item"
            :class="{ on: channel === item.id }"
            @click="router.push(item.path)"
          >
            {{ item.label }}
          </button>
        </nav>
        <div class="work">
          <div class="page">
            <RouterView v-slot="{ Component }">
              <Transition name="page" mode="out-in">
                <component :is="Component" />
              </Transition>
            </RouterView>
          </div>
        </div>
        <ShellPanel v-if="showShell" />
      </main>
    </div>
    <RouterView v-else />
    <ToastStack />
  </div>
</template>

<style scoped>
.app-shell {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--ink-850);
  position: relative;
}
.top-drag {
  height: 0;
  -webkit-app-region: drag;
}
.body {
  flex: 1;
  display: flex;
  min-height: 0;
}
.stage {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
.project-menu {
  display: flex;
  gap: 4px;
  padding: 10px 16px 0;
  -webkit-app-region: drag;
}
.menu-item {
  -webkit-app-region: no-drag;
  min-height: 32px;
  padding: 5px 12px;
  border-radius: var(--r-sm);
  border: 1px solid transparent;
  color: var(--text-dim);
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
}
.menu-item:hover {
  color: var(--text-strong);
  background: var(--btn-default-hover);
}
.menu-item.on {
  color: var(--text-strong);
  background: var(--btn-default-bg);
  border-color: var(--line);
  box-shadow: inset 0 0 0 1px var(--line-soft);
}
.work {
  flex: 1;
  min-height: 0;
  display: flex;
  overflow: hidden;
}
.page {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.page-enter-active,
.page-leave-active {
  transition: opacity 0.14s ease;
}
.page-enter-from,
.page-leave-to {
  opacity: 0;
}
</style>
