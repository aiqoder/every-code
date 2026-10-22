import { app, BrowserWindow, dialog, ipcMain, shell, Tray, Menu, nativeImage } from 'electron'
import path from 'node:path'
import fs from 'node:fs'
import os from 'node:os'
import { fileURLToPath } from 'node:url'
import { IPC } from '../shared/ipc'
import type { LocalSkill, ShellTab } from '../shared/types'
import * as store from './store'
import { request, upload, download, setUnauthorizedHandler, resetSessionExpiry } from './server'
import {
  registerPtyIpc,
  createTab,
  listTabs,
  ensureTaskTab,
  runInTab,
  ptySessions,
  closeTab
} from './pty/manager'
import { initRepo, repoStatus } from './git'
import {
  probeTech,
  installCommandFor,
  hasBrew,
  listLocalSkills,
  importSkillFolder,
  syncSkillCache,
  openInExternal
} from './local'
import {
  startLink,
  stopLink,
  connectionState,
  updateWorkspace,
  currentWorkspace,
  setSkills,
  registerCatalog
} from './chengzi/link'
import { installPageSurfaceIpc } from './chengzi/page-surface'
import { openSsh, writeSsh, resizeSsh, closeSsh, execSsh, uploadSsh } from './ssh'

const __dirname_ = path.dirname(fileURLToPath(import.meta.url))

let mainWindow: BrowserWindow | null = null
let tray: Tray | null = null
/** 托盘常驻：关闭窗口只隐藏，真正退出时才放行。 */
let quitting = false

function rendererUrl(): string {
  if (process.env['ELECTRON_RENDERER_URL']) return process.env['ELECTRON_RENDERER_URL']
  return `file://${path.join(__dirname_, '../renderer/index.html')}`
}

function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 1440,
    height: 920,
    minWidth: 1100,
    minHeight: 720,
    show: false,
    titleBarStyle: process.platform === 'darwin' ? 'hiddenInset' : 'default',
    trafficLightPosition: { x: 18, y: 18 },
    backgroundColor: '#14161a',
    title: 'ADE',
    webPreferences: {
      preload: path.join(__dirname_, '../preload/index.mjs'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
      webSecurity: true
    }
  })

  mainWindow.on('ready-to-show', () => mainWindow?.show())

  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url)
    return { action: 'deny' }
  })

  mainWindow.webContents.on('context-menu', (_event, params) => {
    const field =
      params.formControlType === 'input-text' ||
      params.formControlType === 'input-search' ||
      params.formControlType === 'text-area'
    const template: Electron.MenuItemConstructorOptions[] = field
      ? [{ role: 'cut' }, { role: 'copy' }, { role: 'paste' }, { type: 'separator' }, { role: 'selectAll' }]
      : params.selectionText.trim()
        ? [{ role: 'copy' }]
        : []
    if (!template.length || !mainWindow) return
    Menu.buildFromTemplate(template).popup({ window: mainWindow })
  })

  mainWindow.loadURL(rendererUrl())

  mainWindow.on('close', (event) => {
    if (!quitting) {
      event.preventDefault()
      mainWindow?.hide()
    }
  })
}

function trayIcon(): Electron.NativeImage {
  // 内嵌 16×16 PNG；macOS 菜单栏用 template 单色蒙版。
  const png = Buffer.from(
    'iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAIUlEQVR4nGP4//8/AyWYIs2D1wBCYNSAkWHAwKdE+hoAAOucPO5gR5dDAAAAAElFTkSuQmCC',
    'base64'
  )
  const img = nativeImage.createFromBuffer(png)
  if (process.platform === 'darwin') {
    img.setTemplateImage(true)
  }
  return img.isEmpty() ? nativeImage.createEmpty() : img
}

function createTray(): void {
  tray = new Tray(trayIcon())
  tray.setToolTip('ADE')
  tray.setContextMenu(
    Menu.buildFromTemplate([
      {
        label: '打开 ADE',
        click: () => {
          mainWindow?.show()
          mainWindow?.focus()
        }
      },
      { type: 'separator' },
      {
        label: '退出',
        click: () => {
          quitting = true
          app.quit()
        }
      }
    ])
  )
  tray.on('click', () => {
    mainWindow?.show()
    mainWindow?.focus()
  })
}

setUnauthorizedHandler(() => {
  store.clearToken()
  stopLink()
})

function registerIpc(): void {
  installPageSurfaceIpc()
  // ---- 设置与登录 ----
  ipcMain.handle(IPC.settingsGet, () => ({
    profile: store.getProfile(),
    loggedIn: Boolean(store.getToken()),
    token: store.getToken(),
    skillsDir: store.skillsDir()
  }))

  ipcMain.handle(IPC.settingsSave, (_e, patch) => {
    return store.saveProfile(patch ?? {})
  })

  ipcMain.handle(IPC.login, async (_e, payload: { baseUrl: string; username: string; token: string }) => {
    store.saveProfile({ baseUrl: payload.baseUrl, username: payload.username })
    store.setToken(payload.token)
    resetSessionExpiry()
    await startLink()
    // 同步系统技能到本机缓存，并上报 catalog。
    await syncServerSkills().catch(() => undefined)
    return connectionState()
  })

  ipcMain.handle(IPC.logout, async () => {
    store.clearToken()
    stopLink()
    return connectionState()
  })

  ipcMain.handle(IPC.connect, async () => {
    await startLink()
    return connectionState()
  })

  ipcMain.handle(IPC.statusGet, () => connectionState())

  // ---- 目录选择 ----
  ipcMain.handle(IPC.pickDirectory, async () => {
    const res = await dialog.showOpenDialog({ properties: ['openDirectory', 'createDirectory'] })
    return res.canceled ? '' : res.filePaths[0]
  })

  ipcMain.handle(IPC.pickFile, async (_e, filters?: { name: string; extensions: string[] }[]) => {
    const res = await dialog.showOpenDialog({
      properties: ['openFile'],
      filters: filters ?? [{ name: '所有文件', extensions: ['*'] }]
    })
    return res.canceled ? '' : res.filePaths[0]
  })

  // ---- Git ----
  ipcMain.handle(IPC.gitInit, async (_e, dir: string) => initRepo(dir))
  ipcMain.handle(IPC.gitStatus, async (_e, dir: string) => repoStatus(dir))

  // ---- 环境探测 ----
  ipcMain.handle('ade:probe', async (_e, items: Array<{ tech: string; label: string; command: string }>) => {
    const out = []
    for (const item of items) {
      out.push(await probeTech(item.tech, item.label, item.command))
    }
    return out
  })
  ipcMain.handle('ade:install-hint', () => ({
    brew: hasBrew(),
    recipes: ['go', 'php', 'java', 'node', 'miniprogram'].map((tech) => ({
      tech,
      ...installCommandFor(tech)
    }))
  }))

  // ---- 技能 ----
  ipcMain.handle(IPC.skillListLocal, () => listLocalSkills(store.skillsDir()))
  ipcMain.handle(IPC.skillOpenDir, () => {
    const dir = store.skillsDir()
    openInExternal(dir)
    return dir
  })
  ipcMain.handle(IPC.skillImport, async () => {
    const res = await dialog.showOpenDialog({ properties: ['openDirectory'] })
    if (res.canceled) return null
    const out = await importSkillFolder(store.skillsDir(), res.filePaths[0])
    await refreshSkills()
    return out
  })
  ipcMain.handle('ade:skill-download', async (_e, id: string) => {
    const res = await dialog.showSaveDialog({ defaultPath: `${id}.zip` })
    if (res.canceled || !res.filePath) return ''
    await download(`/api/ade/skills/${encodeURIComponent(id)}/download`, res.filePath)
    return res.filePath
  })
  ipcMain.handle('ade:skill-upload', async (_e, id: string) => {
    const res = await dialog.showOpenDialog({
      properties: ['openFile'],
      filters: [{ name: '技能压缩包', extensions: ['zip'] }]
    })
    if (res.canceled || !res.filePaths[0]) return null
    return upload(`/api/ade/skills/${encodeURIComponent(id)}/upload`, res.filePaths[0], `${id}.zip`)
  })

  // ---- Shell ----
  registerPtyIpc()
  ipcMain.handle('ade:pty:list', () => listTabs())
  ipcMain.handle('ade:pty:ensure-task', (_e, payload: { taskId: number; title: string; cwd: string }) =>
    ensureTaskTab(payload.taskId, payload.title, payload.cwd)
  )
  ipcMain.handle('ade:pty:run', async (_e, payload: { id: string; command: string; timeoutMs?: number }) =>
    runInTab(payload.id, payload.command, payload.timeoutMs)
  )
  ipcMain.handle(IPC.shellOpenInEditor, (_e, target: string) => openInExternal(target))

  // ---- SSH ----
  ipcMain.handle(IPC.sshOpen, async (_e, opts) => {
    const res = await openSsh(opts)
    return res
  })
  // 打开一条 SSH 终端，登记成公共 Shell 的标签。同一主机复用已有标签，避免每次发布都堆一个。
  ipcMain.handle('ade:ssh:open-tab', async (_e, opts: {
    hostId: number
    host: string
    port: number
    username: string
    fingerprint?: string
    remoteDir?: string
    title?: string
  }) => {
    const title = opts.title ?? `SSH ${opts.username}@${opts.host}`
    const reused = findSshTab(opts.hostId)
    if (reused) return { tab: reused, fingerprint: '' }
    const tab = createTab({
      title,
      cwd: opts.remoteDir || os.homedir(),
      kind: 'ssh',
      hostId: opts.hostId
    })
    // 让 ssh.ts 复用这个标签 id，Shell 面板与 SSH 会话一一对应。
    const res = await openSsh({ ...opts, id: tab.id, cols: 120, rows: 30 })
    return { tab, fingerprint: res.fingerprint }
  })
  ipcMain.on(IPC.sshClose, (_e, id: string) => closeSsh(id))
  ipcMain.handle('ade:ssh:exec', async (_e, payload: { id: string; command: string }) =>
    execSsh(payload.id, payload.command)
  )
  ipcMain.handle('ade:ssh:upload', async (_e, payload: { id: string; localPath: string; remotePath: string }) =>
    uploadSsh(payload.id, payload.localPath, payload.remotePath)
  )
  ipcMain.handle('ade:ssh:key', (_e, hostId: number) => store.getPrivateKeyPath(hostId))
  ipcMain.handle('ade:ssh:pick-key', async (_e, hostId: number) => {
    const res = await dialog.showOpenDialog({
      properties: ['openFile', 'showHiddenFiles'],
      defaultPath: path.join(os.homedir(), '.ssh')
    })
    if (res.canceled || !res.filePaths[0]) return ''
    store.setPrivateKeyPath(hostId, res.filePaths[0])
    return res.filePaths[0]
  })

  ipcMain.handle('ade:workspace:set', async (_e, dir: string) => {
    await updateWorkspace(String(dir ?? ''))
    return currentWorkspace()
  })
}

/** 某主机是否已有活着的 SSH 标签（发布与终端复用同一个）。 */
function findSshTab(hostId: number): ShellTab | undefined {
  return listTabs().find((t) => t.kind === 'ssh' && t.hostId === hostId && !t.exited)
}

async function syncServerSkills(): Promise<void> {
  const local = listLocalSkills(store.skillsDir())
  let system: Array<{ id: string; name: string; description: string; agentId?: string }> = []
  try {
    const data = await request<{
      list: Array<{ id: string; name: string; description: string; agentId?: string }>
    }>('/api/ade/skills')
    system = data.list ?? []
  } catch {
    system = []
  }
  const cache = path.join(app.getPath('userData'), 'skill-cache')
  await syncSkillCache(cache, system, async (id) => {
    const fsMod = await import('node:fs')
    const tmp = path.join(cache, `.tmp-${id}.zip`)
    await download(`/api/ade/skills/${encodeURIComponent(id)}/download`, tmp)
    const buf = fsMod.readFileSync(tmp)
    fsMod.rmSync(tmp, { force: true })
    return buf
  }).catch(() => undefined)

  const merged = [
    ...system.map((s) => ({
      id: s.id,
      name: s.name,
      description: s.description,
      agentId: s.agentId
    })),
    ...local.map((s: LocalSkill) => ({
      id: s.id,
      name: s.name,
      description: s.description,
      local: true
    }))
  ]
  setSkills(merged)
  await registerCatalog(currentWorkspace(), merged).catch(() => undefined)
  return
}

async function refreshSkills(): Promise<void> {
  await syncServerSkills().catch(() => undefined)
}

app.whenReady().then(async () => {
  try {
    if (process.platform === 'darwin') {
      app.setAboutPanelOptions({ applicationName: 'ADE', applicationVersion: app.getVersion() })
    }
    registerIpc()
    createWindow()
    createTray()

    if (store.getToken()) {
      await startLink()
      await syncServerSkills().catch(() => undefined)
    }
  } catch (err) {
    console.error('startup failed', err)
  }

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow()
    else mainWindow?.show()
  })
})

app.on('window-all-closed', () => {
  // 托盘常驻，不退出。
})

app.on('before-quit', () => {
  quitting = true
  stopLink()
  for (const sw of ptySessions.keys()) closeTab(sw)
})
