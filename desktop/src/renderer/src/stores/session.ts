import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  AdeCatalog,
  DeployHost,
  Project,
  ReleaseRecord,
  SkillMeta,
  Task
} from '../../../shared/types'
import { adeApi, bridge, installAuthExpiry, llmApi, loginAccount, setHttpSession, type ConnectionStateLike } from '../api'
import { isProviderEnabled } from '../lib/providerEnabled'
import { toast } from '../composables/toast'

export const useSessionStore = defineStore('session', () => {
  const ready = ref(false)
  const loggedIn = ref(false)
  /** 至少有一个已启用供应商下的模型，才能创建项目和开工。 */
  const modelsConfigured = ref(false)
  const modelGateKnown = ref(false)
  let bootPromise: Promise<void> | null = null
  let expiryNotified = false
  const connection = ref<ConnectionStateLike>({
    loggedIn: false,
    username: '',
    baseUrl: '',
    clientId: '',
    wsOnline: false,
    message: '未连接'
  })
  const profile = ref({
    baseUrl: 'http://127.0.0.1:8183',
    username: '',
    clientId: '',
    clientName: 'ADE 桌面',
    workspace: '',
    modelId: 0
  })
  const skillsDir = ref('')

  const projects = ref<Project[]>([])
  const tasks = ref<Record<number, Task[]>>({})
  const catalog = ref<AdeCatalog>({ technologies: [], scenarios: [], agents: [] })
  const systemSkills = ref<SkillMeta[]>([])
  const localSkills = ref<Array<{ id: string; name: string; description: string; path: string }>>([])
  const hosts = ref<DeployHost[]>([])
  const releases = ref<ReleaseRecord[]>([])

  const currentProjectId = ref(0)
  const loadingProjects = ref(false)
  const lastError = ref('')

  const currentProject = computed(
    () => projects.value.find((p) => p.id === currentProjectId.value) ?? null
  )

  /** 六状态任务池，供执行页左栏。 */
  const allTasks = computed(() => Object.values(tasks.value).flat())

  function expireSession(): void {
    const wasIn = loggedIn.value || connection.value.loggedIn
    loggedIn.value = false
    connection.value = {
      ...connection.value,
      loggedIn: false,
      wsOnline: false,
      message: '登录已失效'
    }
    modelsConfigured.value = false
    modelGateKnown.value = false
    projects.value = []
    tasks.value = {}
    if (!wasIn || expiryNotified) return
    expiryNotified = true
    toast.error('登录已失效，请重新登录')
  }

  installAuthExpiry(() => {
    expireSession()
    void bridge.logout().then((state) => {
      connection.value = state
      loggedIn.value = state.loggedIn
    })
  })

  async function bootstrap(): Promise<void> {
    if (ready.value) return
    if (!bootPromise) {
      bootPromise = runBootstrap().catch((err) => {
        bootPromise = null
        throw err
      })
    }
    return bootPromise
  }

  async function runBootstrap(): Promise<void> {
    const settings = await bridge.settings()
    profile.value = settings.profile
    skillsDir.value = settings.skillsDir
    loggedIn.value = settings.loggedIn
    setHttpSession({ baseUrl: settings.profile.baseUrl, token: settings.loggedIn ? settings.token : '' })
    connection.value = await bridge.status()
    bridge.onStatus((state) => {
      const was = loggedIn.value
      connection.value = state
      loggedIn.value = state.loggedIn
      if (!state.loggedIn) setHttpSession({ token: '' })
      if (was && !state.loggedIn) {
        projects.value = []
        tasks.value = {}
        modelsConfigured.value = false
        modelGateKnown.value = false
      }
    })
    if (loggedIn.value) {
      await Promise.all([loadCatalog(), refreshAll(), refreshModelGate()])
    }
    ready.value = true
  }

  /** 已启用供应商下存在模型，才算配置完成。查询失败时不拦截，避免把用户困在模型页。 */
  async function refreshModelGate(): Promise<void> {
    if (!loggedIn.value) {
      modelsConfigured.value = false
      modelGateKnown.value = false
      return
    }
    try {
      const [providerData, modelData] = await Promise.all([llmApi.listProviders(), llmApi.listModels()])
      const enabled = new Set(
        (providerData.list ?? []).filter((item) => item.id && isProviderEnabled(item.id)).map((item) => item.id)
      )
      modelsConfigured.value = (modelData.list ?? []).some((item) => enabled.has(item.providerId))
      modelGateKnown.value = true
    } catch {
      modelGateKnown.value = false
    }
  }

  function postLoginPath(redirect = ''): string {
    if (modelGateKnown.value && !modelsConfigured.value) return '/models'
    if (redirect.startsWith('/') && !redirect.startsWith('/login')) return redirect
    return '/project'
  }

  async function login(baseUrl: string, username: string, password: string): Promise<void> {
    const res = await loginAccount(baseUrl, username, password)
    setHttpSession({ baseUrl, token: res.token })
    try {
      connection.value = await bridge.login({
        baseUrl,
        username: res.user?.username || username,
        token: res.token
      })
    } catch (err) {
      setHttpSession({ token: '' })
      throw err
    }
    expiryNotified = false
    loggedIn.value = true
    const settings = await bridge.settings()
    profile.value = settings.profile
    setHttpSession({ baseUrl: settings.profile.baseUrl, token: settings.token })
    await Promise.all([loadCatalog(), refreshAll(), refreshModelGate()])
  }

  async function setDefaultModel(modelId: number): Promise<void> {
    const next = await bridge.saveSettings({ modelId })
    if (next && typeof next === 'object') {
      profile.value = { ...profile.value, ...(next as typeof profile.value) }
    } else {
      profile.value = { ...profile.value, modelId }
    }
  }

  async function logout(): Promise<void> {
    setHttpSession({ token: '' })
    connection.value = await bridge.logout()
    loggedIn.value = false
    modelsConfigured.value = false
    modelGateKnown.value = false
    projects.value = []
    tasks.value = {}
  }

  async function loadCatalog(): Promise<void> {
    try {
      catalog.value = await adeApi.catalog()
    } catch (err) {
      lastError.value = messageOf(err)
    }
  }

  async function refreshAll(): Promise<void> {
    await Promise.all([loadProjects(), loadSkills(), loadLocalSkills(), loadHosts(), loadReleases()])
  }

  async function loadProjects(): Promise<void> {
    loadingProjects.value = true
    try {
      const data = await adeApi.listProjects()
      projects.value = data.list ?? []
      const saved = Number(localStorage.getItem('ade:project') ?? 0)
      if (!currentProjectId.value) {
        currentProjectId.value =
          projects.value.find((p) => p.id === saved)?.id ?? projects.value[0]?.id ?? 0
      }
    } catch (err) {
      lastError.value = messageOf(err)
    } finally {
      loadingProjects.value = false
    }
  }

  async function loadTasks(projectId: number): Promise<Task[]> {
    if (!projectId) return []
    try {
      const data = await adeApi.listTasks(projectId)
      const list = data.list ?? []
      tasks.value = { ...tasks.value, [projectId]: list }
      return list
    } catch (err) {
      lastError.value = messageOf(err)
      return []
    }
  }

  /** 执行页需要所有项目的任务，逐个项目拉一次。 */
  async function loadAllTasks(): Promise<void> {
    await Promise.all(projects.value.map((p) => loadTasks(p.id)))
  }

  function upsertTask(task: Task): void {
    const list = tasks.value[task.projectId] ?? []
    const idx = list.findIndex((t) => t.id === task.id)
    const next = idx >= 0 ? list.map((t) => (t.id === task.id ? task : t)) : [task, ...list]
    tasks.value = { ...tasks.value, [task.projectId]: next }
  }

  function removeTask(taskId: number, projectId: number): void {
    const list = tasks.value[projectId] ?? []
    tasks.value = { ...tasks.value, [projectId]: list.filter((t) => t.id !== taskId) }
  }

  function selectProject(id: number): void {
    currentProjectId.value = id
    localStorage.setItem('ade:project', String(id))
  }

  async function loadSkills(): Promise<void> {
    try {
      systemSkills.value = (await adeApi.listSkills()).list ?? []
    } catch {
      systemSkills.value = []
    }
  }

  async function loadLocalSkills(): Promise<void> {
    try {
      localSkills.value = await bridge.listLocalSkills()
    } catch {
      localSkills.value = []
    }
  }

  async function loadHosts(): Promise<void> {
    try {
      hosts.value = (await adeApi.listHosts()).list ?? []
    } catch {
      hosts.value = []
    }
  }

  async function loadReleases(): Promise<void> {
    try {
      releases.value = (await adeApi.listReleases()).list ?? []
    } catch {
      releases.value = []
    }
  }

  return {
    ready,
    loggedIn,
    modelsConfigured,
    modelGateKnown,
    connection,
    profile,
    skillsDir,
    projects,
    tasks,
    catalog,
    systemSkills,
    localSkills,
    hosts,
    releases,
    currentProjectId,
    currentProject,
    allTasks,
    loadingProjects,
    lastError,
    bootstrap,
    refreshModelGate,
    postLoginPath,
    login,
    setDefaultModel,
    logout,
    loadCatalog,
    refreshAll,
    loadProjects,
    loadTasks,
    loadAllTasks,
    upsertTask,
    removeTask,
    selectProject,
    loadSkills,
    loadLocalSkills,
    loadHosts,
    loadReleases
  }
})

export function messageOf(err: unknown): string {
  if (err instanceof Error) return err.message
  if (typeof err === 'string') return err
  return '操作失败'
}
