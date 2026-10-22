<script setup lang="ts">
import { useRouter } from 'vue-router'

import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="guest">
    <h1>请使用桌面端</h1>
    <p>网页后台只对管理员开放。项目、会话和执行都在 ADE 桌面端里完成。</p>
    <el-button type="primary" @click="logout">退出登录</el-button>
  </div>
</template>

<style scoped>
.guest {
  min-height: 100vh;
  display: grid;
  place-content: center;
  gap: 12px;
  text-align: center;
  padding: 24px;
}
h1 {
  margin: 0;
  font-size: 22px;
}
p {
  margin: 0;
  color: #606266;
}
</style>
