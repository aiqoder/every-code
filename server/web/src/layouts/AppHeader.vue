<script setup lang="ts">
import { useRouter } from 'vue-router'

import ThemeToggle from '@/components/ThemeToggle.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const appStore = useAppStore()
const auth = useAuthStore()
const router = useRouter()

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <el-header class="admin-header" height="56px">
    <div class="left">
      <slot name="toggle" />
      <span class="page-title">{{ appStore.title }}</span>
    </div>
    <div class="right">
      <ThemeToggle />
      <span class="user-label">{{ auth.displayName || '管理员' }}</span>
      <el-button text type="primary" @click="logout">退出</el-button>
    </div>
  </el-header>
</template>

<style scoped>
.admin-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--admin-header-bg);
  border-bottom: 1px solid var(--admin-border);
  padding: 0 16px;
}

.left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-title {
  font-size: 15px;
  color: var(--admin-text-muted);
}

.right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-label {
  font-size: 14px;
  color: var(--admin-text);
}
</style>
