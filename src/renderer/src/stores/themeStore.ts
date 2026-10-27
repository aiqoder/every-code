import { acceptHMRUpdate, defineStore } from 'pinia'
import { computed, ref } from 'vue'
import {
  defaultThemeId,
  isThemeId,
  themeStorageKey,
  themes,
  type ThemeId
} from '@/theme/github'

function readThemeId(): ThemeId {
  try {
    const stored = localStorage.getItem(themeStorageKey)
    if (stored && isThemeId(stored)) return stored
  } catch {
    // localStorage 不可用时回落到默认暗色
  }
  return defaultThemeId
}

function applyDocumentTheme(id: ThemeId): void {
  const theme = themes[id]
  const root = document.documentElement
  root.dataset.theme = id
  root.style.colorScheme = theme.dark ? 'dark' : 'light'
  try {
    window.electron?.ipcRenderer.send('theme:background', theme.cssVars['--ec-canvas'])
  } catch {
    // 浏览器预览没有桌面窗口
  }
}

export const useThemeStore = defineStore('theme', () => {
  const current = ref<ThemeId>(readThemeId())
  applyDocumentTheme(current.value)

  const isDark = computed(() => themes[current.value].dark)
  const overrides = computed(() => themes[current.value].overrides)

  function setTheme(id: ThemeId): void {
    current.value = id
    try {
      localStorage.setItem(themeStorageKey, id)
    } catch {
      // 记住失败时仍切换当前文档主题
    }
    applyDocumentTheme(id)
  }

  return { current, setTheme, isDark, overrides }
})

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useThemeStore, import.meta.hot))
}
