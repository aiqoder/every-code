import { ElectronAPI } from '@electron-toolkit/preload'
import type { LoopAgentHome, LoopInventory, LoopMessage, LoopPackage } from '../shared/loop'
import type { UpdateCheckResult, UpdateStatus } from '../shared/updater'

export interface DesktopApi {
  loop: {
    dispatch: (message: LoopMessage) => Promise<LoopMessage>
    getWorkspace: () => Promise<string>
    setWorkspace: (path: string) => Promise<string>
    pickWorkspace: () => Promise<string>
    listInventory: () => Promise<LoopInventory>
    listAgentHome: () => Promise<LoopAgentHome>
    installPackage: (filePath?: string) => Promise<LoopPackage | null>
    uninstallPackage: (id: string) => Promise<void>
    savePackageEnv: (id: string, values: Record<string, string>) => Promise<LoopPackage>
    setPackageRemoteId: (id: string, remoteId: number) => Promise<LoopPackage>
    openToolsDir: () => Promise<void>
  }
  files: {
    pathForFile: (file: File) => string
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
