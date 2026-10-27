import { app, BrowserWindow, ipcMain, net } from 'electron'
import { autoUpdater } from 'electron-updater'
import type { ProgressInfo, UpdateInfo } from 'electron-updater'
import type { UpdateCheckResult, UpdateStatus } from '../shared/updater'

const GITHUB_DOWNLOAD_ROOT = 'https://github.com/aiqoder/every-code/releases/latest/download'

/** 拼在 GitHub 下载地址前面。直连超时后按这个顺序换源。 */
const DOWNLOAD_ACCELERATORS = [
  'https://ghfast.top/',
  'https://gh-proxy.com/',
  'https://gh-proxy.org/',
  'https://gh.nxnow.top/'
]

const PROBE_TIMEOUT_MS = 8000
const FEED_GRACE_MS = 1200

let quietErrors = false
let feedQueue: string[] = []
let checkTask: Promise<void> | null = null

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

function channelFileName(): string {
  if (process.platform === 'darwin') return 'latest-mac.yml'
  if (process.platform === 'linux') {
    const suffix = process.arch === 'x64' ? '' : `-${process.arch}`
    return `latest-linux${suffix}.yml`
  }
  return 'latest.yml'
}

function updateFeeds(): string[] {
  return [
    GITHUB_DOWNLOAD_ROOT,
    ...DOWNLOAD_ACCELERATORS.map((prefix) => `${prefix}${GITHUB_DOWNLOAD_ROOT}`)
  ]
}

function isUpdateManifest(body: string): boolean {
  return body.length < 65536 && /^version:\s*\d+\.\d+\.\d+/m.test(body) && body.includes('sha512:')
}

async function probeFeed(feed: string): Promise<boolean> {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), PROBE_TIMEOUT_MS)
  try {
    const response = await net.fetch(`${feed}/${channelFileName()}`, {
      signal: controller.signal,
      redirect: 'follow'
    })
    if (!response.ok) return false
    return isUpdateManifest(await response.text())
  } catch {
    return false
  } finally {
    clearTimeout(timer)
  }
}

function rankFeeds(): Promise<string[]> {
  const feeds = updateFeeds()
  const found = new Set<string>()
  let pending = feeds.length
  return new Promise((resolve) => {
    let settled = false
    let grace: ReturnType<typeof setTimeout> | undefined
    const finish = (): void => {
      if (settled) return
      settled = true
      if (grace) clearTimeout(grace)
      resolve(feeds.filter((feed) => found.has(feed)))
    }
    for (const feed of feeds) {
      void probeFeed(feed).then((ok) => {
        if (settled) return
        if (ok) {
          found.add(feed)
          if (!grace) grace = setTimeout(finish, FEED_GRACE_MS)
        }
        pending -= 1
        if (pending === 0) finish()
      })
    }
  })
}

function failUpdate(error: unknown, fallback: string): never {
  const message = toErrorMessage(error) || fallback
  broadcast({ state: 'error', message })
  if (error instanceof Error && error.message.trim()) throw error
  throw new Error(message)
}

async function runCheck(): Promise<void> {
  const feeds = await rankFeeds()
  if (feeds.length === 0) failUpdate(new Error('暂时连不上更新服务器'), '暂时连不上更新服务器')
  let lastError: unknown
  quietErrors = true
  try {
    for (let index = 0; index < feeds.length; index += 1) {
      try {
        autoUpdater.setFeedURL(feeds[index])
        const result = await autoUpdater.checkForUpdates()
        if (result == null || !result.isUpdateAvailable) {
          feedQueue = []
          return
        }
        feedQueue = feeds.slice(index)
        return
      } catch (error) {
        lastError = error
      }
    }
  } finally {
    quietErrors = false
  }
  feedQueue = []
  failUpdate(lastError, '检查更新失败')
}

function checkWithMirrors(): Promise<void> {
  if (checkTask) return checkTask
  checkTask = runCheck().finally(() => {
    checkTask = null
  })
  return checkTask
}

async function downloadWithMirrors(): Promise<void> {
  if (feedQueue.length === 0) await checkWithMirrors()
  if (feedQueue.length === 0) {
    throw new Error('当前没有可下载的更新')
  }
  let lastError: unknown
  quietErrors = true
  try {
    for (let index = 0; index < feedQueue.length; index += 1) {
      try {
        if (index > 0) {
          autoUpdater.setFeedURL(feedQueue[index])
          const result = await autoUpdater.checkForUpdates()
          if (result == null || !result.isUpdateAvailable) {
            throw new Error('暂时连不上更新服务器')
          }
        }
        await autoUpdater.downloadUpdate()
        return
      } catch (error) {
        lastError = error
      }
    }
  } finally {
    quietErrors = false
  }
  failUpdate(lastError, '下载更新失败')
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
    if (quietErrors) return
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
    await checkWithMirrors()
    return { packaged: true, version }
  })

  ipcMain.handle('updater:download', async (): Promise<boolean> => {
    if (!app.isPackaged) return false
    await downloadWithMirrors()
    return true
  })

  ipcMain.handle('updater:install', () => {
    if (!app.isPackaged) return
    autoUpdater.quitAndInstall(false, true)
  })

  if (!app.isPackaged) return

  const timer = setTimeout(() => {
    void checkWithMirrors().catch(() => undefined)
  }, 4000)
  timer.unref()
}
