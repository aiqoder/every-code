import { BrowserWindow, dialog, ipcMain, type WebContents } from 'electron'
import {
  archiveRuns,
  finishRun,
  getNotify,
  getState,
  reapRuns,
  removeRuns,
  removeTasks,
  runNow,
  saveTask,
  setEnabled,
  setNotify,
  takeDue
} from './store'

export function registerSchedule(): void {
  ipcMain.handle('schedule:state', () => getState())
  ipcMain.handle('schedule:save', (_event, draft: unknown) => saveTask(draft))
  ipcMain.handle('schedule:remove', (_event, ids: unknown) => {
    removeTasks(stringList(ids))
  })
  ipcMain.handle('schedule:enabled', (_event, id: unknown, enabled: unknown) =>
    setEnabled(typeof id === 'string' ? id : '', enabled === true)
  )
  ipcMain.handle('schedule:tick', () => takeDue())
  ipcMain.handle('schedule:run', (_event, id: unknown) => runNow(typeof id === 'string' ? id : ''))
  ipcMain.handle('schedule:finish', (_event, runId: unknown, patch: unknown) =>
    finishRun(typeof runId === 'string' ? runId : '', patch)
  )
  ipcMain.handle('schedule:reap', () => {
    reapRuns()
  })
  ipcMain.handle('schedule:remove-runs', (_event, ids: unknown) => {
    removeRuns(stringList(ids))
  })
  ipcMain.handle('schedule:archive-runs', (_event, ids: unknown, archived: unknown) => {
    archiveRuns(stringList(ids), archived === true)
  })
  ipcMain.handle('schedule:notify', () => getNotify())
  ipcMain.handle('schedule:set-notify', (_event, value: unknown) => setNotify(value === true))
  ipcMain.handle('schedule:pick-directory', (event) => pickDirectory(event.sender))
}

function stringList(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.filter((item): item is string => typeof item === 'string' && item.trim() !== '')
}

async function pickDirectory(sender: WebContents): Promise<string> {
  const win = BrowserWindow.fromWebContents(sender)
  const options = {
    title: '选择工作空间',
    properties: ['openDirectory', 'createDirectory'] as 'openDirectory'[]
  }
  const result = win
    ? await dialog.showOpenDialog(win, options)
    : await dialog.showOpenDialog(options)
  if (result.canceled || !result.filePaths[0]) return ''
  return result.filePaths[0]
}
