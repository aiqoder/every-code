<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ChatLineSquare,
  Cpu,
  Expand,
  Fold,
  Monitor,
  OfficeBuilding,
  SetUp,
  Files,
  Tickets,
} from '@element-plus/icons-vue'

import BrandLogo from '@client/components/BrandLogo.vue'
import { useBrand } from '@client/composables/useBrand'

import AppHeader from './AppHeader.vue'
import { webBrandApiBase } from '@/lib/brand'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
const { brand } = useBrand(webBrandApiBase)
const route = useRoute()
const router = useRouter()

const activeMenu = computed(() => {
  if (route.path.startsWith('/admin/conversations')) return '/admin/conversations'
  if (route.path.startsWith('/admin/system-skills')) return '/admin/system-skills'
  if (route.path.startsWith('/admin/scenes')) return '/admin/scenes'
  return route.path
})

const isBlank = computed(() => !!route.meta.blank)

type MenuItem = {
  path: string
  title: string
  icon: typeof Monitor
  children?: { path: string; title: string; icon: typeof Monitor }[]
}

const menus: MenuItem[] = [
  { path: '/admin/dashboard', title: '概览', icon: Monitor },
  { path: '/admin/system-skills', title: '系统技能', icon: Files },
  { path: '/admin/scenes', title: '场景', icon: Tickets },
  { path: '/admin/tools', title: '模型工具', icon: SetUp },
  { path: '/admin/conversations', title: '会话查看', icon: ChatLineSquare },
  {
    path: '/admin/llm',
    title: '大模型管理',
    icon: Cpu,
    children: [
      { path: '/admin/llm-providers', title: '大模型供应商', icon: OfficeBuilding },
      { path: '/admin/llm-models', title: '大模型维护', icon: Cpu },
    ],
  },
]

function onSelect(path: string) {
  if (path !== route.path) {
    router.push(path)
  }
}
</script>

<template>
  <div v-if="isBlank" class="admin-blank">
    <router-view />
  </div>
  <el-container v-else class="admin-layout">
    <el-aside :width="appStore.sidebarCollapsed ? '64px' : '220px'" class="admin-aside">
      <div class="brand">
        <BrandLogo class="brand-logo" :logo="brand.logo" :alt="brand.name" />
        <span v-if="!appStore.sidebarCollapsed" class="brand-text">{{ brand.name }}</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        :collapse="appStore.sidebarCollapsed"
        :collapse-transition="false"
        class="admin-menu"
        @select="onSelect"
      >
        <template v-for="item in menus" :key="item.path">
          <el-sub-menu v-if="item.children?.length" :index="item.path">
            <template #title>
              <el-icon><component :is="item.icon" /></el-icon>
              <span>{{ item.title }}</span>
            </template>
            <el-menu-item v-for="child in item.children" :key="child.path" :index="child.path">
              <el-icon><component :is="child.icon" /></el-icon>
              <template #title>{{ child.title }}</template>
            </el-menu-item>
          </el-sub-menu>
          <el-menu-item v-else :index="item.path">
            <el-icon><component :is="item.icon" /></el-icon>
            <template #title>{{ item.title }}</template>
          </el-menu-item>
        </template>
      </el-menu>
    </el-aside>

    <el-container direction="vertical">
      <AppHeader>
        <template #toggle>
          <el-icon class="collapse-btn" @click="appStore.toggleSidebar()">
            <Fold v-if="!appStore.sidebarCollapsed" />
            <Expand v-else />
          </el-icon>
        </template>
      </AppHeader>
      <el-main class="admin-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.admin-blank {
  min-height: 100%;
  background: var(--admin-main-bg);
}

.admin-layout {
  height: 100%;
}

.admin-aside {
  background: var(--admin-sidebar);
  border-right: 1px solid var(--admin-sidebar-border);
  transition: width 0.2s ease;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.brand {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 12px;
  color: var(--admin-sidebar-text);
  font-weight: 600;
  border-bottom: 1px solid var(--admin-sidebar-border);
}

.brand-logo {
  font-size: 20px;
}

.brand :deep(.brand-logo img) {
  height: 22px;
  max-width: 28px;
}

.brand:has(.brand-text) :deep(.brand-logo img) {
  max-width: 88px;
}

.brand-text {
  font-size: 15px;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.admin-aside :deep(.el-menu) {
  border-right: none;
  flex: 1;
}

.admin-main {
  background: var(--admin-main-bg);
  padding: 16px;
}

.admin-aside :deep(.admin-menu) {
  --el-menu-bg-color: var(--admin-sidebar);
  --el-menu-text-color: var(--admin-sidebar-muted);
  --el-menu-hover-bg-color: var(--admin-sidebar-active);
  --el-menu-active-color: var(--admin-sidebar-text);
  --el-menu-hover-text-color: var(--admin-sidebar-text);
}

.collapse-btn {
  cursor: pointer;
  font-size: 20px;
}
</style>
