import { acceptHMRUpdate, defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import {
  defaultAppearance,
  isAppearanceId,
  resolveTheme,
  themeStorageKey,
  themes,
  type AppearanceId,
  type ThemeId
} from '@/theme/github'

function readAppearance(): AppearanceId {
  try {
    const stored = localStorage.getItem(themeStorageKey)
    if (stored && isAppearanceId(stored)) return stored
  } catch {
    // localStorage 不可用时回落到跟随系统
  }
  return defaultAppearance
}

function prefersDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
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
  const current = ref<AppearanceId>(readAppearance())
  const systemDark = ref(prefersDark())
  const media = window.matchMedia('(prefers-color-scheme: dark)')

  function onSystemChange(): void {
    systemDark.value = media.matches
  }

  media.addEventListener('change', onSystemChange)

  const resolved = computed(() => resolveTheme(current.value, systemDark.value))
  const isDark = computed(() => themes[resolved.value].dark)

  watch(resolved, (id) => applyDocumentTheme(id), { immediate: true })

  function setTheme(id: AppearanceId): void {
    current.value = id
    try {
      localStorage.setItem(themeStorageKey, id)
    } catch {
      // 记住失败时仍切换当前文档主题
    }
  }

  return { current, setTheme, isDark }
})

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useThemeStore, import.meta.hot))
}
