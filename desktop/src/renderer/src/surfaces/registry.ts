import { messageOf } from '../stores/session'
import { toast } from '../composables/toast'
import { emptyParams, fail, pack, type ActionResult, type PageSurface, type PageToolDef } from './types'

let current: PageSurface | null = null

export function activeSurface(): PageSurface | null {
  return current
}

export function catalogOf(surface: PageSurface): PageToolDef[] {
  const state: PageToolDef = {
    name: 'page_state',
    description: `读取「${surface.title}」上用户当前能看到的内容。操作这个页面前先调用。`,
    parameters: emptyParams()
  }
  const actions = surface.actions.map((action) => ({
    name: `page_${action.name}`,
    description: action.description,
    parameters: action.parameters
  }))
  return [state, ...actions]
}

async function publish(surface: PageSurface | null): Promise<void> {
  const ade = window.ade
  if (!ade?.publishPageSurface) return
  if (!surface) {
    await ade.publishPageSurface(null).catch(() => undefined)
    return
  }
  await ade
    .publishPageSurface({
      channel: surface.channel,
      page: surface.page,
      tools: catalogOf(surface)
    })
    .catch(() => undefined)
}

export function mountSurface(surface: PageSurface): void {
  current = surface
  void publish(surface)
}

export function unmountSurface(surface: PageSurface): void {
  if (current !== surface) return
  current = null
  void publish(null)
}

function decodeArgs(raw: unknown): Record<string, unknown> {
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw || '{}') as unknown
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) return parsed as Record<string, unknown>
    } catch {
      return {}
    }
    return {}
  }
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) return raw as Record<string, unknown>
  return {}
}

/** 智能体路径：执行后把短句提示给正在看页面的用户。 */
export function showResult(result: ActionResult, withErrors = true): ActionResult {
  if (result.error) {
    if (withErrors) toast.error(result.error)
    return result
  }
  if (result.toast) {
    if (result.notice === 'plain') toast.show(result.toast)
    else toast.success(result.toast)
  }
  return result
}

export async function invokeSurface(name: string, raw: unknown): Promise<ActionResult> {
  const surface = current
  if (!surface) return fail('当前没有打开可操作的页面')
  const short = name.startsWith('page_') ? name.slice('page_'.length) : name
  try {
    if (short === 'state') {
      return pack({
        page: surface.page,
        title: surface.title,
        channel: surface.channel,
        state: surface.snapshot()
      })
    }
    const action = surface.actions.find((item) => item.name === short)
    if (!action) return fail(`当前页面没有这个操作：${name}`)
    return showResult(await action.run(decodeArgs(raw)))
  } catch (err) {
    return showResult(fail(messageOf(err)))
  }
}
