<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

import { oauthCallback } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

/**
 * 授权码是一次性的，重复提交会让 user-feedback 返回 invalid_grant。
 * 用 sessionStorage 记住本标签页已用过的 code，刷新/后退时不再重复提交。
 */
const USED_CODE_KEY = 'chengzi_oauth_used_code'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const message = ref('正在完成授权登录…')

onMounted(async () => {
  const code = typeof route.query.code === 'string' ? route.query.code : ''
  const errorDesc = typeof route.query.error_description === 'string' ? route.query.error_description : ''
  if (errorDesc) {
    message.value = errorDesc
    ElMessage.error(errorDesc)
    return
  }
  if (!code) {
    message.value = '缺少授权码，无法完成登录'
    ElMessage.error(message.value)
    return
  }

  // 已经登录成功的用户回到回调页，直接送回原落地页，不再消耗授权码。
  if (auth.token) {
    router.replace(auth.user?.role === 'admin' ? '/admin/dashboard' : '/guest')
    return
  }

  if (sessionStorage.getItem(USED_CODE_KEY) === code) {
    message.value = '该授权链接已使用过，正在返回登录页…'
    ElMessage.warning(message.value)
    router.replace('/login')
    return
  }

  try {
    const res = await oauthCallback(code)
    // 先落标记再跳转，避免跳转过程中组件重渲染导致重复提交。
    sessionStorage.setItem(USED_CODE_KEY, code)
    auth.setSession(res.token, res.user)
    ElMessage.success('授权登录成功')
    router.replace(res.user.role === 'admin' ? '/admin/dashboard' : '/guest')
  } catch (e) {
    message.value = e instanceof Error ? e.message : '授权登录失败'
    ElMessage.error(message.value)
  }
})
</script>

<template>
  <div class="callback-page">
    <p>{{ message }}</p>
    <RouterLink to="/login">返回登录</RouterLink>
  </div>
</template>

<style scoped>
.callback-page {
  min-height: 100%;
  display: grid;
  place-items: center;
  gap: 12px;
  text-align: center;
  font-family: 'Source Sans 3', 'Noto Sans SC', sans-serif;
  color: #6b5e55;
}

.callback-page a {
  color: #bc4b02;
}
</style>
