import { acceptHMRUpdate, defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getMe, type AuthUser } from '@/api/auth'
import {
  clearApiKey,
  normalizeApiBase,
  readApiBase,
  readApiKey,
  rememberApiBase,
  saveSession
} from '@/lib/sessionConfig'

const clientKey = 'everycode.clientId'

function ensureClientId(): string {
  let id = localStorage.getItem(clientKey)
  if (!id) {
    id = crypto.randomUUID()
    localStorage.setItem(clientKey, id)
  }
  return id
}

export const useSessionStore = defineStore('session', () => {
  const apiBase = ref(readApiBase())
  const apiKey = ref(readApiKey())
  const user = ref<AuthUser | null>(null)
  const clientId = ref(ensureClientId())
  const loggedIn = computed(() => Boolean(apiKey.value))

  async function login(base: string, key: string): Promise<void> {
    const secret = key.trim()
    if (!secret) throw new Error('请填写密钥')
    const normalized = normalizeApiBase(base)
    rememberApiBase(normalized)
    saveSession(normalized, secret)
    apiBase.value = normalized
    try {
      const me = await getMe()
      if (!me?.id) throw new Error('无效的密钥')
      apiKey.value = secret
      user.value = me
    } catch (err) {
      clearApiKey()
      apiKey.value = ''
      user.value = null
      throw err
    }
  }

  function logout(): void {
    clearApiKey()
    apiKey.value = ''
    user.value = null
  }

  return { apiBase, apiKey, user, clientId, loggedIn, login, logout }
})

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useSessionStore, import.meta.hot))
}
