import { onMounted, onUnmounted, reactive, ref } from 'vue'

export type ResolvedTheme = 'light' | 'dark'

function readTheme(): ResolvedTheme {
  const value = document.documentElement.dataset.theme
  if (value === 'light' || value === 'dark') return value
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

/** 读取 html[data-theme]，供桌面与网页共用的 Markdown / 代码高亮使用。 */
export function useDocumentTheme() {
  const resolved = ref<ResolvedTheme>(readTheme())
  let observer: MutationObserver | null = null

  function sync() {
    resolved.value = readTheme()
  }

  onMounted(() => {
    sync()
    observer = new MutationObserver(sync)
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
  })

  onUnmounted(() => observer?.disconnect())

  return reactive({ resolved })
}
