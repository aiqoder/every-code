import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'

export type ThemePreference = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'chengzi.theme'

function readStored(): ThemePreference {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (raw === 'light' || raw === 'dark' || raw === 'system') return raw
  return 'system'
}

function systemDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

export const useThemeStore = defineStore('theme', () => {
  const preference = ref<ThemePreference>(readStored())
  const systemIsDark = ref(systemDark())

  const resolved = computed<'light' | 'dark'>(() => {
    if (preference.value === 'system') {
      return systemIsDark.value ? 'dark' : 'light'
    }
    return preference.value
  })

  function apply() {
    document.documentElement.setAttribute('data-theme', resolved.value)
  }

  function setPreference(next: ThemePreference) {
    preference.value = next
    localStorage.setItem(STORAGE_KEY, next)
    apply()
  }

  function init() {
    apply()
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => {
      systemIsDark.value = mq.matches
      if (preference.value === 'system') apply()
    }
    mq.addEventListener('change', onChange)
  }

  watch(resolved, apply)

  return { preference, resolved, setPreference, init }
})
