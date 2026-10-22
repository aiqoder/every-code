import type {
  AdeCatalog,
  CommandAudit,
  DeployHost,
  DeployTask,
  Project,
  ReleaseRecord,
  Scene,
  SkillFile,
  SkillMeta,
  Task,
  Todo,
  EnvCheckItem
} from '../../../shared/types'
import { request, setHttpSession, loginAccount, watchTaskEvents, postSse, installAuthExpiry, httpReady } from './http'

export { setHttpSession, loginAccount, watchTaskEvents, postSse, installAuthExpiry, httpReady }

declare global {
  interface Window {
    ade: {
      getSettings(): Promise<{
        profile: {
          baseUrl: string
          username: string
          clientId: string
          clientName: string
          workspace: string
          modelId: number
        }
        loggedIn: boolean
        token: string
        skillsDir: string
      }>
      saveSettings(patch: Record<string, unknown>): Promise<unknown>
      login(payload: { baseUrl: string; username: string; token: string }): Promise<ConnectionStateLike>
      logout(): Promise<ConnectionStateLike>
      connect(): Promise<ConnectionStateLike>
      getStatus(): Promise<ConnectionStateLike>
      onStatus(cb: (state: ConnectionStateLike) => void): () => void
      pickDirectory(): Promise<string>
      pickFile(filters?: { name: string; extensions: string[] }[]): Promise<string>
      gitInit(dir: string): Promise<{ ok: boolean; output: string; created: boolean }>
      gitStatus(
        dir: string
      ): Promise<{ isRepo: boolean; branch: string; dirty: boolean; ready: boolean }>
      probe(items: Array<{ tech: string; label: string; command: string }>): Promise<EnvCheckItem[]>
      installHint(): Promise<{
        brew: boolean
        recipes: Array<{ tech: string; command: string; note: string }>
      }>
      createTab(opts: { title?: string; cwd?: string; taskId?: number }): Promise<ShellTabLike>
      listTabs(): Promise<ShellTabLike[]>
      ensureTaskTab(payload: { taskId: number; title: string; cwd: string }): Promise<ShellTabLike>
      runInTab(payload: { id: string; command: string; timeoutMs?: number }): Promise<{
        code: number
        output: string
      }>
      writePty(id: string, data: string): void
      resizePty(id: string, cols: number, rows: number): void
      closePty(id: string): void
      onPtyData(cb: (payload: { id: string; data: string }) => void): () => void
      onPtyExit(cb: (payload: { id: string; exitCode: number }) => void): () => void
      listLocalSkills(): Promise<Array<{ id: string; name: string; description: string; path: string }>>
      openSkillsDir(): Promise<string>
      importSkill(): Promise<{ id: string } | null>
      downloadSkill(id: string): Promise<string>
      uploadSkill(id: string): Promise<unknown>
      openSsh(opts: unknown): Promise<{ id: string; fingerprint: string }>
      openSshTab(opts: unknown): Promise<{ tab: ShellTabLike; fingerprint: string }>
      closeSsh(id: string): void
      execSsh(payload: { id: string; command: string }): Promise<{ code: number; output: string }>
      uploadSsh(payload: {
        id: string
        localPath: string
        remotePath: string
      }): Promise<{ files: number; bytes: number }>
      getSshKey(hostId: number): Promise<string>
      pickSshKey(hostId: number): Promise<string>
      openExternal(target: string): Promise<void>
      setWorkspace(dir: string): Promise<string>
      publishPageSurface(payload: {
        channel: string
        page: string
        tools: Array<{ name: string; description: string; parameters: Record<string, unknown> }>
      } | null): Promise<boolean>
      onPageAction(cb: (payload: { id: string; name: string; arguments: string }) => void): () => void
      replyPageAction(id: string, result: { content: string; error: string }): void
      platform: string
    }
  }
}

export interface ConnectionStateLike {
  loggedIn: boolean
  username: string
  baseUrl: string
  clientId: string
  wsOnline: boolean
  message: string
}

export interface ShellTabLike {
  id: string
  title: string
  cwd: string
  kind: 'local' | 'ssh'
  taskId?: number
  hostId?: number
  exited: boolean
  exitCode?: number
}

const api = (): Window['ade'] => {
  if (!window.ade) throw new Error('主进程接口未就绪')
  return window.ade
}

export const bridge = {
  settings: () => api().getSettings(),
  saveSettings: (patch: Record<string, unknown>) => api().saveSettings(patch),
  login: (payload: { baseUrl: string; username: string; token: string }) => api().login(payload),
  logout: () => api().logout(),
  connect: () => api().connect(),
  status: () => api().getStatus(),
  onStatus: (cb: (s: ConnectionStateLike) => void) => api().onStatus(cb),
  pickDirectory: () => api().pickDirectory(),
  pickFile: (filters?: { name: string; extensions: string[] }[]) => api().pickFile(filters),
  createTab: (opts: { title?: string; cwd?: string; taskId?: number }) => api().createTab(opts),
  listTabs: () => api().listTabs(),
  ensureTaskTab: (payload: { taskId: number; title: string; cwd: string }) =>
    api().ensureTaskTab(payload),
  writePty: (id: string, data: string) => api().writePty(id, data),
  resizePty: (id: string, cols: number, rows: number) => api().resizePty(id, cols, rows),
  closePty: (id: string) => api().closePty(id),
  onPtyData: (cb: (payload: { id: string; data: string }) => void) => api().onPtyData(cb),
  onPtyExit: (cb: (payload: { id: string; exitCode: number }) => void) => api().onPtyExit(cb),
  gitInit: (dir: string) => api().gitInit(dir),
  gitStatus: (dir: string) => api().gitStatus(dir),
  probe: (items: Array<{ tech: string; label: string; command: string }>) => api().probe(items),
  installHint: () => api().installHint(),
  openSsh: (opts: unknown) => api().openSsh(opts),
  openSshTab: (opts: unknown) => api().openSshTab(opts),
  closeSsh: (id: string) => api().closeSsh(id),
  execSsh: (payload: { id: string; command: string }) => api().execSsh(payload),
  uploadSsh: (payload: { id: string; localPath: string; remotePath: string }) =>
    api().uploadSsh(payload),
  runInTab: (payload: { id: string; command: string; timeoutMs?: number }) =>
    api().runInTab(payload),
  getSshKey: (hostId: number) => api().getSshKey(hostId),
  pickSshKey: (hostId: number) => api().pickSshKey(hostId),
  openExternal: (target: string) => api().openExternal(target),
  listLocalSkills: () => api().listLocalSkills(),
  openSkillsDir: () => api().openSkillsDir(),
  importSkill: () => api().importSkill(),
  downloadSkill: (id: string) => api().downloadSkill(id),
  uploadSkill: (id: string) => api().uploadSkill(id),
  setWorkspace: (dir: string) => api().setWorkspace(dir),
  platform: () => api().platform
}

/** 渲染进程直接请求服务端。 */
export function get<T>(path: string): Promise<T> {
  return request<T>(path)
}
export function post<T>(path: string, body?: unknown, timeoutMs?: number): Promise<T> {
  return request<T>(path, { method: 'POST', body, timeoutMs })
}
export function patch<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: 'PATCH', body })
}
export function put<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: 'PUT', body })
}
export function del<T>(path: string): Promise<T> {
  return request<T>(path, { method: 'DELETE' })
}

export interface LlmProvider {
  id: number
  name: string
  website: string
  apiKey: string
  openaiCompatibleBaseUrl: string
  anthropicBaseUrl: string
  editable: boolean
}

export interface LlmModelRow {
  id: number
  providerId: number
  providerName: string
  name: string
  attachment: boolean
  reasoning: boolean
  temperature: boolean
  inputContextLimit: number
  outputContextLimit: number
  editable: boolean
}

export interface StationAgent {
  agentId: string
  name: string
  tools: string[]
}

export interface ConversationItem {
  id: number
  title: string
  modelId: number
  clientId: string
  projectId: number
  channel: string
  taskId: number
  workspace: string
  createdAt: string
  updatedAt: string
}

export interface ConversationMessage {
  id: number
  role: string
  content: string
  toolCallId?: string
  toolName: string
  metadata?: string
  createdAt: string
}

// ---------- ADE 接口 ----------
export const adeApi = {
  catalog: () => get<AdeCatalog>('/api/ade/catalog'),

  listScenes: () => get<{ list: Scene[] }>('/api/ade/scenes'),
  createScene: (payload: { title: string; description: string }) => post<Scene>('/api/ade/scenes', payload),
  updateScene: (id: number, payload: { title: string; description: string }) =>
    patch<Scene>(`/api/ade/scenes/${id}`, payload),
  deleteScene: (id: number) => del<{ id: number }>(`/api/ade/scenes/${id}`),

  listProjects: () => get<{ list: Project[] }>('/api/ade/projects'),
  createProject: (payload: {
    name: string
    description: string
    localDir: string
    scenario: string
    technologies: string[]
    modelId: number
    sceneId?: number
  }) => post<Project>('/api/ade/projects', payload),
  getProject: (id: number) => get<Project>(`/api/ade/projects/${id}`),
  updateProject: (id: number, payload: Partial<{
    name: string
    description: string
    localDir: string
    scenario: string
    technologies: string[]
    modelId: number
  }>) => patch<Project>(`/api/ade/projects/${id}`, payload),
  deleteProject: (id: number) => del<{ id: number }>(`/api/ade/projects/${id}`),
  probeProject: (id: number) =>
    get<{ items: EnvCheckItem[] }>(`/api/ade/projects/${id}/probe`),
  saveEnvCheck: (id: number, items: EnvCheckItem[]) =>
    post<Project>(`/api/ade/projects/${id}/env-check`, { items }),

  listTasks: (projectId: number) => get<{ list: Task[] }>(`/api/ade/projects/${projectId}/tasks`),
  createTask: (
    projectId: number,
    payload: {
      description: string
      name: string
      overview: string
      plan: string
      todos: Array<{ title: string; detail?: string }>
    }
  ) => post<Task>(`/api/ade/projects/${projectId}/tasks`, payload),

  getTask: (id: number) => get<Task>(`/api/ade/tasks/${id}`),
  deleteTask: (id: number) => del<{ id: number }>(`/api/ade/tasks/${id}`),
  updateDescription: (id: number, description: string) =>
    post<Task>(`/api/ade/tasks/${id}/description`, { description }),
  applyPlan: (
    id: number,
    payload: {
      name: string
      overview: string
      plan: string
      todos: Array<{ title: string; detail?: string }>
    }
  ) => post<Task>(`/api/ade/tasks/${id}/plan`, payload),
  startTask: (id: number) => post<Task>(`/api/ade/tasks/${id}/start`),
  retryTask: (id: number) => post<Task>(`/api/ade/tasks/${id}/retry`),
  addTodo: (id: number, payload: { title: string; detail?: string }) =>
    post<Todo>(`/api/ade/tasks/${id}/todos`, payload),
  updateTodo: (todoId: number, payload: Record<string, unknown>) =>
    patch<Todo>(`/api/ade/todos/${todoId}`, payload),
  deleteTodo: (todoId: number) => del<{ id: number }>(`/api/ade/todos/${todoId}`),

  answerQuestion: (
    conversationId: number,
    questionId: string,
    answers: Array<{ questionId: string; selected: string[]; text?: string }>
  ) =>
    post<{ ok: boolean }>(
      `/api/conversations/${conversationId}/questions/${encodeURIComponent(questionId)}`,
      { answers }
    ),

  listSkills: () => get<{ list: SkillMeta[] }>('/api/ade/skills'),
  skillFiles: (id: string) => get<{ id: string; files: SkillFile[] }>(`/api/ade/skills/${id}/files`),
  skillFile: (id: string, path: string) =>
    get<{ id: string; path: string; content: string }>(
      `/api/ade/skills/${id}/file?path=${encodeURIComponent(path)}`
    ),

  listHosts: () => get<{ list: DeployHost[] }>('/api/ade/hosts'),
  saveHost: (payload: Partial<DeployHost>) => post<DeployHost>('/api/ade/hosts', payload),
  deleteHost: (id: number) => del<{ id: number }>(`/api/ade/hosts/${id}`),
  listDeployTasks: (projectId: number) =>
    get<{ list: DeployTask[] }>(`/api/ade/projects/${projectId}/deploy-tasks`),
  createDeployTask: (projectId: number, payload: { title: string; prompt: string }) =>
    post<DeployTask>(`/api/ade/projects/${projectId}/deploy-tasks`, payload),
  updateDeployTask: (id: number, payload: { title: string; prompt: string }) =>
    patch<DeployTask>(`/api/ade/deploy-tasks/${id}`, payload),
  deleteDeployTask: (id: number) => del<{ id: number }>(`/api/ade/deploy-tasks/${id}`),
  listReleases: () => get<{ list: ReleaseRecord[] }>('/api/ade/releases'),
  createRelease: (payload: {
    taskId: number
    hostId: number
    version: string
    status: string
    detail: string
    previous: string
  }) => post<ReleaseRecord>('/api/ade/releases', payload),
  audit: (payload: { taskId: number; hostId: number; command: string; exitCode: number; output: string }) =>
    post('/api/ade/audits', payload),
  listStationAgents: () => get<{ list: StationAgent[] }>('/api/ade/station-agents'),

  listConversations: (params: { clientId?: string; projectId: number; channel: string; taskId?: number }) => {
    const q = new URLSearchParams()
    if (params.clientId) q.set('clientId', params.clientId)
    q.set('projectId', String(params.projectId))
    q.set('channel', params.channel)
    if (params.taskId) q.set('taskId', String(params.taskId))
    q.set('pageSize', '50')
    return get<{ list: ConversationItem[] }>(`/api/conversations?${q.toString()}`)
  },
  createConversation: (payload: {
    modelId: number
    clientId: string
    title: string
    workspace: string
    projectId: number
    channel: string
    taskId?: number
  }) => post<ConversationItem>('/api/conversations', payload),
  updateConversationModel: (id: number, modelId: number) =>
    put<ConversationItem>(`/api/conversations/${id}/model`, { modelId }),
  getConversation: (id: number) =>
    get<ConversationItem & { messages: ConversationMessage[] }>(`/api/conversations/${id}`),

  listAudits: (params?: { hostId?: number; taskId?: number; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.hostId) q.set('hostId', String(params.hostId))
    if (params?.taskId) q.set('taskId', String(params.taskId))
    if (params?.limit) q.set('limit', String(params.limit))
    const qs = q.toString()
    return get<{ list: CommandAudit[] }>(`/api/ade/audits${qs ? `?${qs}` : ''}`)
  }
}

export const llmApi = {
  listProviders: () => get<{ list: LlmProvider[] }>('/api/llm-providers?all=1'),
  createProvider: (payload: {
    name: string
    website?: string
    apiKey?: string
    openaiCompatibleBaseUrl?: string
    anthropicBaseUrl?: string
  }) => post<LlmProvider>('/api/llm-providers', payload),
  updateProvider: (
    id: number,
    payload: {
      name: string
      website?: string
      apiKey?: string
      openaiCompatibleBaseUrl?: string
      anthropicBaseUrl?: string
    }
  ) => put<LlmProvider>(`/api/llm-providers/${id}`, payload),
  deleteProvider: (id: number) => del(`/api/llm-providers/${id}`),
  remoteModels: (id: number) => get<{ list: string[] }>(`/api/llm-providers/${id}/remote-models`),
  listModels: (providerId?: number) => {
    const q = new URLSearchParams({ page: '1', pageSize: '100' })
    if (providerId) q.set('providerId', String(providerId))
    return get<{ list: LlmModelRow[] }>(`/api/llm-models?${q.toString()}`)
  },
  createModel: (payload: Record<string, unknown>) => post<LlmModelRow>('/api/llm-models', payload),
  updateModel: (id: number, payload: Record<string, unknown>) =>
    put<LlmModelRow>(`/api/llm-models/${id}`, payload),
  deleteModel: (id: number) => del(`/api/llm-models/${id}`)
}
