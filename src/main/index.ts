import { app, shell, BrowserWindow, ipcMain } from 'electron'
import { join } from 'path'
import { electronApp, optimizer, is } from '@electron-toolkit/utils'
import icon from '../../resources/icon.png?asset'
import { registerLoop } from './loop'
import { registerUpdater } from './updater'

function createWindow(): void {
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 1360,
    height: 760,
    minWidth: 1080,
    minHeight: 560,
    show: false,
    autoHideMenuBar: true,
    title: 'Every Code',
    backgroundColor: '#0d1117',
    // 系统标题栏不跟随页面主题。macOS 收进内容区后，顶栏背景由页面自己铺。
    ...(process.platform === 'darwin'
      ? { titleBarStyle: 'hiddenInset' as const, trafficLightPosition: { x: 14, y: 18 } }
      : {}),
    ...(process.platform === 'linux' ? { icon } : {}),
    webPreferences: {
      preload: join(__dirname, '../preload/index.js'),
      sandbox: false,
      // 渲染进程直连 chengzi-agent-server。服务端没有 CORS，桌面窗口需要放行跨域。
      webSecurity: false
    }
  })

  mainWindow.on('ready-to-show', () => {
    mainWindow.show()
  })

  mainWindow.webContents.setWindowOpenHandler((details) => {
    shell.openExternal(details.url)
    return { action: 'deny' }
  })

  // HMR for renderer base on electron-vite cli.
  // Load the remote URL for development or the local html file for production.
  if (is.dev && process.env['ELECTRON_RENDERER_URL']) {
    mainWindow.loadURL(process.env['ELECTRON_RENDERER_URL'])
  } else {
    mainWindow.loadFile(join(__dirname, '../renderer/index.html'))
  }
}

function revealMainWindow(): void {
  const win = BrowserWindow.getAllWindows()[0]
  if (!win) {
    createWindow()
    return
  }
  if (win.isMinimized()) win.restore()
  win.show()
  win.focus()
}

// 同一台电脑只保留一个进程。后启动的实例退出，并唤起已经打开的窗口。
const hasSingleInstanceLock = app.requestSingleInstanceLock()
if (!hasSingleInstanceLock) {
  app.quit()
} else {
  app.on('second-instance', () => {
    if (app.isReady()) revealMainWindow()
  })

  app.whenReady().then(() => {
    electronApp.setAppUserModelId('com.everycode.desktop')

    // Default open or close DevTools by F12 in development
    // and ignore CommandOrControl + R in production.
    // see https://github.com/alex8088/electron-toolkit/tree/master/packages/utils
    app.on('browser-window-created', (_, window) => {
      optimizer.watchWindowShortcuts(window)
    })

    ipcMain.on('ping', () => console.log('pong'))
    ipcMain.on('theme:background', (event, color: unknown) => {
      if (typeof color !== 'string' || !/^#[0-9a-fA-F]{6}$/.test(color)) return
      BrowserWindow.fromWebContents(event.sender)?.setBackgroundColor(color)
    })
    registerLoop()
    registerUpdater()

    createWindow()

    app.on('activate', () => {
      // macOS 关闭窗口后应用仍在。点 Dock 时如果没有窗口，再开一个。
      if (BrowserWindow.getAllWindows().length === 0) createWindow()
    })
  })

  // 非 macOS 在窗口全关时退出。macOS 需用户用 Cmd + Q 退出。
  app.on('window-all-closed', () => {
    if (process.platform !== 'darwin') {
      app.quit()
    }
  })
}
