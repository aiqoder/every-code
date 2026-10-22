import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useSessionStore } from './session'

function storageKey(workspace: string) {
  const session = useSessionStore()
  const who = session.status.username || session.status.userName || 'anon'
  return `chengzi.lastAgentId:${who}:${workspace}`
}

/**
 * 按工作目录记住用户上次选择的智能体（安装包 ID）。
 */
export const useAgentPrefStore = defineStore('agentPref', () => {
  const lastAgentId = ref<string | null>(null)

  function hydrate(workspace: string) {
    const raw = (localStorage.getItem(storageKey(workspace)) || '').trim()
    lastAgentId.value = raw && !/^\d+$/.test(raw) ? raw : null
  }

  function remember(workspace: string, id: string | null | undefined) {
    if (!id) return
    lastAgentId.value = id
    localStorage.setItem(storageKey(workspace), id)
  }

  /**
   * 从已安装智能体中挑选默认选中项：优先上次记忆，否则第一个。
   */
  function pick<T extends { id: string }>(workspace: string, agents: T[]) {
    hydrate(workspace)
    if (lastAgentId.value && agents.some((a) => a.id === lastAgentId.value)) {
      return lastAgentId.value
    }
    return agents[0]?.id ?? null
  }

  return { lastAgentId, hydrate, remember, pick }
})
