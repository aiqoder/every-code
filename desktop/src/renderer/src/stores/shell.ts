import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { bridge, type ShellTabLike } from '../api'

/** 公共 Shell：当前项目共用底栏终端，切换页面不销毁会话。 */
export const useShellStore = defineStore('shell', () => {
  const tabs = ref<ShellTabLike[]>([])
  const activeId = ref('')
  const collapsed = ref(true)
  const height = ref(280)
  /** 每个标签的输出缓冲，重挂载时重放，避免切页丢历史。 */
  const buffers = ref<Record<string, string>>({})
  const subscribed = ref(false)

  const activeTab = computed(() => tabs.value.find((t) => t.id === activeId.value) ?? null)

  function subscribe(): void {
    if (subscribed.value) return
    subscribed.value = true
    bridge.onPtyData(({ id, data }) => {
      buffers.value[id] = ((buffers.value[id] ?? '') + data).slice(-400_000)
    })
    bridge.onPtyExit(({ id, exitCode }) => {
      const tab = tabs.value.find((t) => t.id === id)
      if (tab) {
        tab.exited = true
        tab.exitCode = exitCode
      }
    })
  }

  async function refresh(): Promise<void> {
    tabs.value = await bridge.listTabs()
    if (!activeId.value && tabs.value.length) activeId.value = tabs.value[0].id
  }

  async function open(opts: { title?: string; cwd?: string; taskId?: number }): Promise<ShellTabLike> {
    subscribe()
    const tab = await bridge.createTab(opts)
    tabs.value = [...tabs.value, tab]
    activeId.value = tab.id
    collapsed.value = false
    return tab
  }

  /** 开始执行一条任务时自动打开（或聚焦）标题为该任务名的标签。 */
  async function openTask(taskId: number, title: string, cwd: string): Promise<ShellTabLike> {
    subscribe()
    const tab = await bridge.ensureTaskTab({ taskId, title, cwd })
    if (!tabs.value.some((t) => t.id === tab.id)) {
      tabs.value = [...tabs.value, tab]
    }
    activeId.value = tab.id
    collapsed.value = false
    return tab
  }

  /** 接管主进程已建好的标签（例如 SSH 会话）。 */
  async function adopt(tab: ShellTabLike): Promise<void> {
    subscribe()
    if (!tabs.value.some((t) => t.id === tab.id)) {
      tabs.value = [...tabs.value, tab]
    }
    activeId.value = tab.id
    collapsed.value = false
  }

  async function close(id: string): Promise<void> {
    bridge.closePty(id)
    tabs.value = tabs.value.filter((t) => t.id !== id)
    delete buffers.value[id]
    if (activeId.value === id) activeId.value = tabs.value[0]?.id ?? ''
    if (!tabs.value.length) collapsed.value = true
  }

  function rename(id: string, title: string): void {
    const tab = tabs.value.find((t) => t.id === id)
    if (tab) tab.title = title.trim() || tab.title
  }

  function setActive(id: string): void {
    activeId.value = id
  }

  function toggle(): void {
    collapsed.value = !collapsed.value
  }

  return {
    tabs,
    activeId,
    collapsed,
    height,
    buffers,
    activeTab,
    subscribe,
    refresh,
    open,
    openTask,
    adopt,
    close,
    rename,
    setActive,
    toggle
  }
})
