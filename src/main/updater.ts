import { app, BrowserWindow, ipcMain } from 'electron'
import { autoUpdater } from 'electron-updater'
import type { ProgressInfo, UpdateInfo } from 'electron-updater'
import type { UpdateCheckResult, UpdateStatus } from '../shared/updater'

function broadcast(status: UpdateStatus): void {
  for (const win of BrowserWindow.getAllWindows()) {
    if (win.isDestroyed() || win.webContents.isDestroyed()) continue
    win.webContents.send('updater:status', status)
  }
}

function toErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim()) return error.message
  if (typeof error === 'string' && error.trim()) return error
  return '检查更新失败'
}

function formatReleaseNotes(notes: UpdateInfo['releaseNotes']): string {
  if (typeof notes === 'string') return notes.trim()
  if (!Array.isArray(notes)) return ''
  return notes
    .map((item) => (item.note ?? '').trim())
    .filter((note) => note.length > 0)
    .join('\n\n')
}

function configurePackagedUpdater(): void {
  autoUpdater.autoDownload = false
  autoUpdater.autoInstallOnAppQuit = true

  autoUpdater.on('checking-for-update', () => {
    broadcast({ state: 'checking' })
  })

  autoUpdater.on('update-available', (info: UpdateInfo) => {
    broadcast({
      state: 'available',
      version: info.version,
      releaseNotes: formatReleaseNotes(info.releaseNotes)
    })
  })

  autoUpdater.on('update-not-available', (info: UpdateInfo) => {
    broadcast({ state: 'not-available', version: info.version })
  })

  autoUpdater.on('download-progress', (progress: ProgressInfo) => {
    const percent = Number.isFinite(progress.percent) ? progress.percent : 0
    broadcast({ state: 'downloading', percent })
  })

  autoUpdater.on('update-downloaded', (info: UpdateInfo) => {
    broadcast({ state: 'downloaded', version: info.version })
  })

  autoUpdater.on('error', (error: Error, message?: string) => {
    const text = error instanceof Error && error.message.trim() ? error.message : message
    broadcast({ state: 'error', message: text?.trim() || '更新失败' })
  })
}

export function registerUpdater(): void {
  if (app.isPackaged) configurePackagedUpdater()

  ipcMain.handle('updater:version', () => app.getVersion())

  ipcMain.handle('updater:check', async (): Promise<UpdateCheckResult> => {
    const version = app.getVersion()
    if (!app.isPackaged) {
      broadcast({ state: 'not-available', version })
      return { packaged: false, version }
    }
    try {
      await autoUpdater.checkForUpdates()
      return { packaged: true, version }
    } catch (error) {
      broadcast({ state: 'error', message: toErrorMessage(error) })
      throw error
    }
  })

  ipcMain.handle('updater:download', async (): Promise<boolean> => {
    if (!app.isPackaged) return false
    await autoUpdater.downloadUpdate()
    return true
  })

  ipcMain.handle('updater:install', () => {
    if (!app.isPackaged) return
    autoUpdater.quitAndInstall(false, true)
  })

  if (!app.isPackaged) return

  const timer = setTimeout(() => {
    void autoUpdater.checkForUpdates().catch((error: unknown) => {
      broadcast({ state: 'error', message: toErrorMessage(error) })
    })
  }, 4000)
  timer.unref()
}
