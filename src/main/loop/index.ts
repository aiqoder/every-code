import { BrowserWindow, dialog, ipcMain } from 'electron'
import type { LoopMessage } from '../../shared/loop'
import { dispatch } from './agent/dispatch'
import { getWorkspace, setWorkspace } from './agent/workspace'

export function registerLoop(): void {
  ipcMain.handle('loop:dispatch', async (_event, message: LoopMessage) => dispatch(message))
  ipcMain.handle('loop:get-workspace', () => getWorkspace())
  ipcMain.handle('loop:set-workspace', (_event, path: string) => setWorkspace(path))
  ipcMain.handle('loop:pick-workspace', async (event) => {
    const win = BrowserWindow.fromWebContents(event.sender)
    const options = {
      title: '选择工作目录',
      properties: ['openDirectory', 'createDirectory'] as 'openDirectory'[]
    }
    const result = win
      ? await dialog.showOpenDialog(win, options)
      : await dialog.showOpenDialog(options)
    if (result.canceled || !result.filePaths[0]) return ''
    return setWorkspace(result.filePaths[0])
  })
}
