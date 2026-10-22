import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { useAuthStore } from '@/stores/auth'
import type { Status } from '../types'

/**
 * 客户端页面用的登录态。网页宿主没有独立桌面会话，这里接到账号 store。
 */
export const useSessionStore = defineStore('session', () => {
  const auth = useAuthStore()
  const busy = ref(false)
  const notice = ref('')

  const status = computed<Status>(() => ({
    loggedIn: auth.isLoggedIn,
    connected: auth.isLoggedIn,
    serverUrl: '',
    clientId: '',
    clientName: '',
    workspace: '',
    username: auth.user?.username || '',
    userName: auth.user?.name || '',
    userRole: auth.user?.role || '',
    lastError: notice.value,
  }))

  async function login(_serverURL: string, username: string, password: string) {
    busy.value = true
    notice.value = ''
    try {
      await auth.login(username, password)
    } catch (err) {
      notice.value = err instanceof Error ? err.message : '登录失败'
      throw err
    } finally {
      busy.value = false
    }
  }

  return { status, busy, notice, login }
})
