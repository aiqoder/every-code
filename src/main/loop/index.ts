import { BrowserWindow, dialog, ipcMain, shell, type WebContents } from 'electron'
import type { LoopMessage } from '../../shared/loop'
import { dispatch } from './agent/dispatch'
import { getWorkspace, setWorkspace } from './agent/workspace'
import { listInventory } from './inventory'
import {
  installPackageFile,
  saveInstalledEnv,
  setInstalledRemoteId,
  uninstallInstalledPackage
} from './package/install'
import { listPackageViews } from './package/view'
import { userToolsDir } from './package/paths'
import { toolNotes } from './tool'

export function registerLoop(): void {
  ipcMain.handle('loop:dispatch', async (_event, message: LoopMessage) => dispatch(message))
  ipcMain.handle('loop:list-inventory', async () => listInventory())
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
  ipcMain.handle('loop:list-agent-home', () => ({
    toolsDir: userToolsDir(),
    packages: listPackageViews(),
    toolNotes: toolNotes()
  }))
  ipcMain.handle('loop:install-package', async (event, filePath?: string) => {
    const target = await pickPackage(event.sender, filePath)
    if (!target) return null
    return installPackageFile(target, [])
  })
  ipcMain.handle('loop:uninstall-package', (_event, id: string) => {
    uninstallInstalledPackage(typeof id === 'string' ? id : '')
  })
  ipcMain.handle('loop:save-package-env', (_event, id: string, values: Record<string, unknown>) => {
    return saveInstalledEnv(typeof id === 'string' ? id : '', values || {})
  })
  ipcMain.handle('loop:set-package-remote', (_event, id: string, remoteId: number) => {
    return setInstalledRemoteId(typeof id === 'string' ? id : '', Number(remoteId))
  })
  ipcMain.handle('loop:open-tools-dir', async () => {
    const opened = await shell.openPath(userToolsDir())
    if (opened) throw new Error(opened)
  })
}

async function pickPackage(sender: WebContents, filePath: string | undefined): Promise<string> {
  const given = typeof filePath === 'string' ? filePath.trim() : ''
  if (given) return given
  const win = BrowserWindow.fromWebContents(sender)
  const options = {
    title: '安装智能体插件',
    filters: [{ name: '智能体插件', extensions: ['cpk', 'agent'] }],
    properties: ['openFile'] as 'openFile'[]
  }
  const result = win
    ? await dialog.showOpenDialog(win, options)
    : await dialog.showOpenDialog(options)
  if (result.canceled || !result.filePaths[0]) return ''
  return result.filePaths[0]
}
