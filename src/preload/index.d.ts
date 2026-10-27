import { ElectronAPI } from '@electron-toolkit/preload'
import type { LoopAgentHome, LoopInventory, LoopMessage, LoopPackage } from '../shared/loop'
import type {
  ScheduleDraft,
  ScheduleFire,
  ScheduleRun,
  ScheduleRunPatch,
  ScheduleState,
  ScheduledTask
} from '../shared/schedule'
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
  schedule: {
    state: () => Promise<ScheduleState>
    save: (draft: ScheduleDraft) => Promise<ScheduledTask>
    remove: (ids: string[]) => Promise<void>
    setEnabled: (id: string, enabled: boolean) => Promise<ScheduledTask>
    tick: () => Promise<ScheduleFire[]>
    run: (id: string) => Promise<ScheduleFire>
    finish: (runId: string, patch: ScheduleRunPatch) => Promise<ScheduleRun>
    reap: () => Promise<void>
    removeRuns: (ids: string[]) => Promise<void>
    archiveRuns: (ids: string[], archived: boolean) => Promise<void>
    getNotify: () => Promise<boolean>
    setNotify: (value: boolean) => Promise<boolean>
    pickDirectory: () => Promise<string>
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
