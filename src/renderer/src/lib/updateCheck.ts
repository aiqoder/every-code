type CheckHandler = () => Promise<void>

let handler: CheckHandler | null = null

/** 侧边栏菜单触发检查，更新卡片仍由 UpdateNotifier 负责展示。 */
export function bindUpdateCheck(next: CheckHandler): () => void {
  handler = next
  return () => {
    if (handler === next) handler = null
  }
}

export function requestUpdateCheck(): Promise<void> {
  if (!handler) return Promise.resolve()
  return handler()
}
