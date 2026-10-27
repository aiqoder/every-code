import { ElectronAPI } from '@electron-toolkit/preload'
import type { LoopMessage } from '../shared/loop'
import type { UpdateCheckResult, UpdateStatus } from '../shared/updater'

export interface DesktopApi {
  loop: {
    dispatch: (message: LoopMessage) => Promise<LoopMessage>
    getWorkspace: () => Promise<string>
    setWorkspace: (path: string) => Promise<string>
    pickWorkspace: () => Promise<string>
  }
  updater: {
    version: () => Promise<string>
    check: () => Promise<UpdateCheckResult>
    download: () => Promise<boolean>
    install: () => Promise<void>
    onStatus: (cb: (status: UpdateStatus) => void) => () => void
  }
}

declare global {
  interface Window {
    electron: ElectronAPI
    api: DesktopApi
  }
}
