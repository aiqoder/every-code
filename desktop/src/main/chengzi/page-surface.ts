import { BrowserWindow, ipcMain } from 'electron'
import { randomUUID } from 'node:crypto'
import { IPC } from '../../shared/ipc'

export interface PageToolDef {
  name: string
  description: string
  parameters: Record<string, unknown>
}

interface PageSurfacePublish {
  channel: string
  page: string
  tools: PageToolDef[]
}

interface ToolResult {
  content: string
  error: string
}

let published: PageSurfacePublish | null = null
const pending = new Map<string, (result: ToolResult) => void>()
let onChange: (() => void) | null = null

export function onPageSurfaceChange(fn: () => void): void {
  onChange = fn
}

function send(channel: string, payload: unknown): void {
  for (const win of BrowserWindow.getAllWindows()) {
    if (!win.isDestroyed()) win.webContents.send(channel, payload)
  }
}

export function installPageSurfaceIpc(): void {
  ipcMain.handle(IPC.pageSurfacePublish, (_e, payload: PageSurfacePublish | null) => {
    if (!payload || !payload.channel) {
      published = null
    } else {
      published = {
        channel: String(payload.channel),
        page: String(payload.page ?? ''),
        tools: Array.isArray(payload.tools) ? payload.tools : []
      }
    }
    onChange?.()
    return true
  })

  ipcMain.on(IPC.pageSurfaceResult, (_e, payload: { id?: string; result?: ToolResult }) => {
    const id = String(payload?.id ?? '')
    const done = pending.get(id)
    if (!done) return
    pending.delete(id)
    done({
      content: String(payload?.result?.content ?? ''),
      error: String(payload?.result?.error ?? '')
    })
  })
}

/** 把 page_ 工具调用转到当前页面。页面操作可能包含环境检测，超时放宽。 */
export function runPageAction(name: string, rawArgs: string): Promise<ToolResult> {
  const id = randomUUID()
  return new Promise((resolve) => {
    const timer = setTimeout(() => {
      if (!pending.has(id)) return
      pending.delete(id)
      resolve({ content: '', error: '页面没有在限定时间内完成这个操作' })
    }, 600_000)
    pending.set(id, (result) => {
      clearTimeout(timer)
      resolve(result)
    })
    const windows = BrowserWindow.getAllWindows().filter((win) => !win.isDestroyed())
    if (!windows.length) {
      clearTimeout(timer)
      pending.delete(id)
      resolve({ content: '', error: '界面还没打开' })
      return
    }
    send(IPC.pageSurfaceCall, { id, name, arguments: rawArgs || '{}' })
  })
}
