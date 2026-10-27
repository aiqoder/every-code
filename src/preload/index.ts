import { contextBridge, ipcRenderer, type IpcRendererEvent } from 'electron'
import { electronAPI } from '@electron-toolkit/preload'
import type { LoopMessage } from '../shared/loop'
import type { UpdateCheckResult, UpdateStatus } from '../shared/updater'

const api = {
  loop: {
    dispatch: (message: LoopMessage): Promise<LoopMessage> =>
      ipcRenderer.invoke('loop:dispatch', message),
    getWorkspace: (): Promise<string> => ipcRenderer.invoke('loop:get-workspace'),
    setWorkspace: (path: string): Promise<string> => ipcRenderer.invoke('loop:set-workspace', path),
    pickWorkspace: (): Promise<string> => ipcRenderer.invoke('loop:pick-workspace')
  },
  updater: {
    version: (): Promise<string> => ipcRenderer.invoke('updater:version'),
    check: (): Promise<UpdateCheckResult> => ipcRenderer.invoke('updater:check'),
    download: (): Promise<boolean> => ipcRenderer.invoke('updater:download'),
    install: (): Promise<void> => ipcRenderer.invoke('updater:install'),
    onStatus: (cb: (status: UpdateStatus) => void): (() => void) => {
      const listener = (_event: IpcRendererEvent, status: UpdateStatus): void => {
        cb(status)
      }
      ipcRenderer.on('updater:status', listener)
      return () => {
        ipcRenderer.removeListener('updater:status', listener)
      }
    }
  }
}

if (process.contextIsolated) {
  try {
    contextBridge.exposeInMainWorld('electron', electronAPI)
    contextBridge.exposeInMainWorld('api', api)
  } catch (error) {
    console.error(error)
  }
} else {
  // @ts-ignore (define in dts)
  window.electron = electronAPI
  // @ts-ignore (define in dts)
  window.api = api
}

export type DesktopApi = typeof api
