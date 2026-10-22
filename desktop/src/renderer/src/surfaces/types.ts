/**
 * 页面按钮。
 *
 * 这里只给界面上的按钮用。智能体改项目和任务时走服务端工具，直接写数据库，不调用这些动作。
 *
 * 后续加按钮时：
 * 1. 在对应页面的 surface 里加一个 action（名字、说明、参数、run）。
 * 2. 页面状态放进该文件导出的 reactive 对象，不要只放在 .vue 里。
 * 3. 按钮改为调用这个 action。
 */
export interface ActionResult {
  content: string
  error: string
  /** 短句。智能体调用时提示用户；按钮路径由页面决定要不要再提示。 */
  toast?: string
  /** plain 用普通提示，其余用成功提示。 */
  notice?: 'success' | 'plain'
}

export interface PageAction {
  /** 短名，登记时自动加上 page_ 前缀。模型要求工具名只含字母、数字、下划线和连字符。 */
  name: string
  description: string
  parameters: Record<string, unknown>
  run(args: Record<string, unknown>): Promise<ActionResult>
}

export interface PageSurface {
  /** 会话通道。同一通道可以出现在多处，不限于某一块侧栏。 */
  channel: string
  page: string
  title: string
  snapshot(): unknown
  actions: PageAction[]
}

export interface PageToolDef {
  name: string
  description: string
  parameters: Record<string, unknown>
}

export function ok(content: string, toast?: string, notice: ActionResult['notice'] = 'success'): ActionResult {
  return { content, error: '', toast, notice }
}

export function fail(error: string): ActionResult {
  return { content: '', error }
}

export function emptyParams(): Record<string, unknown> {
  return { type: 'object', properties: {}, additionalProperties: false }
}

export function params(properties: Record<string, unknown>, required: string[] = []): Record<string, unknown> {
  return { type: 'object', properties, required, additionalProperties: false }
}

export function pack(data: unknown): ActionResult {
  let content = ''
  try {
    content = JSON.stringify(data, null, 2)
  } catch {
    content = String(data)
  }
  if (content.length > 12_000) content = `${content.slice(0, 12_000)}\n…(已截断)`
  return { content, error: '' }
}
