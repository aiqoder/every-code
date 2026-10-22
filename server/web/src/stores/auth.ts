import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { fetchMe, login as loginApi } from '@/api/auth'
import type { User } from '@/types/api'

const TOKEN_KEY = 'chengzi_token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) || '')
  const user = ref<User | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const displayName = computed(() => user.value?.name || user.value?.username || '')

  function setSession(nextToken: string, nextUser: User) {
    token.value = nextToken
    user.value = nextUser
    localStorage.setItem(TOKEN_KEY, nextToken)
  }

  function clearSession() {
    token.value = ''
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
  }

  async function login(username: string, password: string) {
    const res = await loginApi({ username, password })
    setSession(res.token, res.user)
    return res.user
  }

  async function loadMe() {
    if (!token.value) {
      user.value = null
      return null
    }
    try {
      const me = await fetchMe()
      user.value = me
      return me
    } catch {
      clearSession()
      return null
    }
  }

  function logout() {
    clearSession()
  }

  return {
    token,
    user,
    isLoggedIn,
    isAdmin,
    displayName,
    login,
    loadMe,
    logout,
    setSession,
    clearSession,
  }
})
