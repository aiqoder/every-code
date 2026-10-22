import { onBeforeUnmount, onMounted } from 'vue'
import { invokeSurface, mountSurface, unmountSurface } from './registry'
import type { PageSurface } from './types'

/** 页面挂载时把自己登记为「当前页」。智能体随后只能调用这一页的操作。 */
export function usePageSurface(surface: PageSurface): void {
  onMounted(() => mountSurface(surface))
  onBeforeUnmount(() => unmountSurface(surface))
}

export function installPageSurfaceHost(): void {
  window.ade.onPageAction((payload) => {
    void invokeSurface(payload.name, payload.arguments).then((result) => {
      window.ade.replyPageAction(payload.id, { content: result.content, error: result.error })
    })
  })
}
