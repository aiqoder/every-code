import { contextBridge, ipcRenderer } from 'electron'
import { IPC } from '../shared/ipc'

const invoke = <T>(channel: string, ...args: unknown[]): Promise<T> =>
  ipcRenderer.invoke(channel, ...args) as Promise<T>

const api = {
  // 设置与连接
  getSettings: () => invoke(IPC.settingsGet),
  saveSettings: (patch: unknown) => invoke(IPC.settingsSave, patch),
  login: (payload: { baseUrl: string; username: string; token: string }) =>
    invoke(IPC.login, payload),
  logout: () => invoke(IPC.logout),
  connect: () => invoke(IPC.connect),
  getStatus: () => invoke(IPC.statusGet),
  onStatus: (cb: (state: unknown) => void) => {
    const handler = (_e: unknown, payload: unknown): void => cb(payload)
    ipcRenderer.on(IPC.statusEvent, handler)
    return () => ipcRenderer.removeListener(IPC.statusEvent, handler)
  },

  pickDirectory: () => invoke<string>(IPC.pickDirectory),
  pickFile: (filters?: { name: string; extensions: string[] }[]) =>
    invoke<string>(IPC.pickFile, filters),

  // Git
  gitInit: (dir: string) => invoke(IPC.gitInit, dir),
  gitStatus: (dir: string) => invoke(IPC.gitStatus, dir),

  // 本机环境
  probe: (items: Array<{ tech: string; label: string; command: string }>) =>
    invoke('ade:probe', items),
  installHint: () => invoke('ade:install-hint'),

  // PTY
  createTab: (opts: { title?: string; cwd?: string; taskId?: number }) =>
    invoke(IPC.ptyCreate, opts),
  listTabs: () => invoke('ade:pty:list'),
  ensureTaskTab: (payload: { taskId: number; title: string; cwd: string }) =>
    invoke('ade:pty:ensure-task', payload),
  runInTab: (payload: { id: string; command: string; timeoutMs?: number }) =>
    invoke('ade:pty:run', payload),
  writePty: (id: string, data: string) => ipcRenderer.send(IPC.ptyWrite, { id, data }),
  resizePty: (id: string, cols: number, rows: number) =>
    ipcRenderer.send(IPC.ptyResize, { id, cols, rows }),
  closePty: (id: string) => ipcRenderer.send(IPC.ptyClose, id),
  onPtyData: (cb: (payload: { id: string; data: string }) => void) => {
    const handler = (_e: unknown, payload: { id: string; data: string }): void => cb(payload)
    ipcRenderer.on(IPC.ptyData, handler)
    return () => ipcRenderer.removeListener(IPC.ptyData, handler)
  },
  onPtyExit: (cb: (payload: { id: string; exitCode: number }) => void) => {
    const handler = (_e: unknown, payload: { id: string; exitCode: number }): void => cb(payload)
    ipcRenderer.on(IPC.ptyExit, handler)
    return () => ipcRenderer.removeListener(IPC.ptyExit, handler)
  },

  // 本机技能
  listLocalSkills: () => invoke(IPC.skillListLocal),
  openSkillsDir: () => invoke(IPC.skillOpenDir),
  importSkill: () => invoke(IPC.skillImport),
  downloadSkill: (id: string) => invoke('ade:skill-download', id),
  uploadSkill: (id: string) => invoke('ade:skill-upload', id),

  // SSH
  openSsh: (opts: unknown) => invoke(IPC.sshOpen, opts),
  openSshTab: (opts: unknown) => invoke('ade:ssh:open-tab', opts),
  closeSsh: (id: string) => ipcRenderer.send(IPC.sshClose, id),
  execSsh: (payload: { id: string; command: string }) => invoke('ade:ssh:exec', payload),
  uploadSsh: (payload: { id: string; localPath: string; remotePath: string }) =>
    invoke('ade:ssh:upload', payload),
  getSshKey: (hostId: number) => invoke('ade:ssh:key', hostId),
  pickSshKey: (hostId: number) => invoke('ade:ssh:pick-key', hostId),

  openExternal: (target: string) => invoke(IPC.shellOpenInEditor, target),

  setWorkspace: (dir: string) => invoke('ade:workspace:set', dir),

  publishPageSurface: (payload: unknown) => invoke<boolean>(IPC.pageSurfacePublish, payload),
  onPageAction: (cb: (payload: { id: string; name: string; arguments: string }) => void) => {
    const handler = (_e: unknown, payload: { id: string; name: string; arguments: string }): void => cb(payload)
    ipcRenderer.on(IPC.pageSurfaceCall, handler)
    return () => ipcRenderer.removeListener(IPC.pageSurfaceCall, handler)
  },
  replyPageAction: (id: string, result: { content: string; error: string }) =>
    ipcRenderer.send(IPC.pageSurfaceResult, { id, result }),

  platform: process.platform
}

contextBridge.exposeInMainWorld('ade', api)

export type AdeApi = typeof api
