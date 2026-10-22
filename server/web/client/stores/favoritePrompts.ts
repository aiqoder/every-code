import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface FavoritePrompt {
  id: string
  title: string
  content: string
  createdAt: number
}

const STORAGE_KEY = 'chengzi:favorite-prompts'
const MAX_ITEMS = 100

function load(): FavoritePrompt[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const list = JSON.parse(raw) as FavoritePrompt[]
    return Array.isArray(list) ? list : []
  } catch {
    return []
  }
}

function save(list: FavoritePrompt[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list))
}

function titleFromContent(content: string) {
  const t = content.replace(/\/skill:[^\s]+/g, ' ').replace(/\s+/g, ' ').trim()
  return t.slice(0, 30) || '未命名'
}

export const useFavoritePromptsStore = defineStore('favoritePrompts', () => {
  const items = ref<FavoritePrompt[]>(load())

  function persist() {
    save(items.value)
  }

  function add(content: string, title?: string) {
    const text = content.trim()
    if (!text) return false
    if (items.value.length >= MAX_ITEMS) return false
    items.value = [
      {
        id: crypto.randomUUID(),
        title: (title || titleFromContent(text)).trim(),
        content: text,
        createdAt: Date.now(),
      },
      ...items.value,
    ]
    persist()
    return true
  }

  function update(id: string, patch: Partial<Pick<FavoritePrompt, 'title' | 'content'>>) {
    const i = items.value.findIndex((x) => x.id === id)
    const current = items.value[i]
    if (!current) return
    items.value[i] = { ...current, ...patch }
    persist()
  }

  function remove(id: string) {
    items.value = items.value.filter((x) => x.id !== id)
    persist()
  }

  return { items, add, update, remove }
})
