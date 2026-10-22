import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ToolItem } from '../types'

type ToolsLoader = () => Promise<ToolItem[] | null>

let loadTools: ToolsLoader | null = null

/** 由桌面会话注入。网页复用聊天组件时不注册，显示名回退为工具名。 */
export function setToolsLoader(loader: ToolsLoader) {
  loadTools = loader
}

// 工具显示名：本地 catalog 优先，服务端 http/builtin 作补充。
export const useToolsStore = defineStore('tools', () => {
  const byName = ref<Record<string, string>>({})

  function applyList(list: ToolItem[] | null | undefined) {
    const next: Record<string, string> = {}
    for (const t of list ?? []) {
      const name = t.name?.trim()
      const displayName = t.displayName?.trim()
      if (name && displayName) next[name] = displayName
    }
    byName.value = next
  }

  async function refresh() {
    if (!loadTools) return
    try {
      applyList(await loadTools())
    } catch {
      // 未登录或接口失败时保留已有缓存
    }
  }

  function clear() {
    byName.value = {}
  }

  return { byName, refresh, clear, applyList }
})
