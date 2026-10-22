// 渲染进程与主进程共用的类型。不要在这里 import 任何运行时依赖。

export type TaskStatus = 'pending' | 'running' | 'wait_test' | 'tested' | 'rerun' | 'failed'

export const TASK_STATUS_META: Record<TaskStatus, { label: string; hint: string; tone: string }> = {
  pending: { label: '待执行', hint: '计划已就绪，等待开始', tone: 'muted' },
  running: { label: '执行中', hint: 'Agent 正在开发', tone: 'brass' },
  wait_test: { label: '待测试', hint: '开发完成，等待验收', tone: 'amber' },
  tested: { label: '测试完成', hint: '已合入 dev 并复核', tone: 'moss' },
  rerun: { label: '重新执行', hint: '验收未过，回到开发', tone: 'clay' },
  failed: { label: '执行出错', hint: '需要人工介入', tone: 'rust' }
}

export const STATUS_ORDER: TaskStatus[] = [
  'pending',
  'running',
  'wait_test',
  'tested',
  'rerun',
  'failed'
]

export interface EnvCheckItem {
  tech: string
  label: string
  installed: boolean
  version: string
  command: string
}

export interface Scene {
  id: number
  userId: number
  title: string
  description: string
  createdAt: string
  updatedAt: string
}

export interface Project {
  id: number
  userId: number
  name: string
  description: string
  localDir: string
  scenario: string
  technologies: string[]
  modelId: number
  sceneId: number
  envReady: boolean
  clientId: string
  createdAt: string
  updatedAt: string
  envCheck: EnvCheckItem[]
  taskCount: number
  runningTask: number
}

export interface Todo {
  id: number
  taskId: number
  seq: number
  title: string
  detail: string
  acceptance: string
  dependsOn: string
  status: 'todo' | 'doing' | 'done'
  source: 'agent' | 'user'
  parentId: number
  retry: number
  createdAt: string
  updatedAt: string
}

export interface AskQuestionOption {
  id: string
  label: string
}

export interface AskQuestionItem {
  id: string
  prompt: string
  allow_multiple?: boolean
  options: AskQuestionOption[]
}

export interface PendingAskQuestion {
  conversationId: number
  questionId: string
  toolCallId: string
  title: string
  questions: AskQuestionItem[]
  deadlineAt: number
  timeoutSec: number
  status: 'pending' | 'submitting' | 'done' | 'timeout'
  error?: string
}

export interface TaskError {
  runId: number
  kind: string
  agentId: string
  message: string
  at: string
}

export interface Task {
  id: number
  userId: number
  projectId: number
  name: string
  overview: string
  description: string
  rawRequest: string
  status: TaskStatus
  mode: string
  branch: string
  worktree: string
  mergedDev: boolean
  mergedMain: boolean
  clientId: string
  coderAgentId: string
  verifierAgentId: string
  reviewerAgentId: string
  retries: number
  planned: boolean
  createdAt: string
  updatedAt: string
  projectName: string
  todos: Todo[]
  doneCount: number
  totalCount: number
  lastError?: TaskError | null
}

export interface RunRecord {
  id: number
  taskId: number
  projectId: number
  userId: number
  kind: 'planner' | 'advisor' | 'coder' | 'verifier' | 'reviewer' | 'installer' | 'deploy'
  agentId: string
  status: 'running' | 'done' | 'failed' | 'skipped'
  conversationId: number
  summary: string
  error: string
  startedAt: string
  finishedAt: string
}

export interface TaskEvent {
  taskId: number
  runId: number
  conversationId?: number
  kind: string
  seq: number
  type: string
  text?: string
  toolName?: string
  toolCallId?: string
  arguments?: string
  result?: string
  error?: string
  data?: Record<string, unknown>
}

export interface CommandAudit {
  id: number
  userId: number
  taskId: number
  hostId: number
  command: string
  exitCode: number
  output: string
  createdAt: string
}

export interface SkillMeta {
  id: string
  name: string
  description: string
  fileCount: number
  updatedAt: string
}

export interface SkillFile {
  path: string
  size: number
  isDir: boolean
}

export interface DeployHost {
  id: number
  userId: number
  name: string
  host: string
  port: number
  username: string
  fingerprint: string
  remoteDir: string
  playbook: string
  createdAt: string
  updatedAt: string
}

export interface DeployTask {
  id: number
  userId: number
  projectId: number
  title: string
  prompt: string
  createdAt: string
  updatedAt: string
}

export interface ReleaseRecord {
  id: number
  userId: number
  taskId: number
  hostId: number
  version: string
  status: string
  detail: string
  previous: string
  createdAt: string
}

export interface TechOption {
  id: string
  label: string
  kind: 'frontend' | 'backend' | 'tool'
  probe: string
  skill: string
  summary: string
}

export interface ScenarioOption {
  id: string
  label: string
  summary: string
  technologies: string[]
}

export interface AgentInfo {
  id: string
  name: string
  description: string
}

export interface AdeCatalog {
  technologies: TechOption[]
  scenarios: ScenarioOption[]
  agents: AgentInfo[]
}

/** 公共 Shell 的标签页。 */
export interface ShellTab {
  id: string
  title: string
  cwd: string
  kind: 'local' | 'ssh'
  taskId?: number
  hostId?: number
  exited: boolean
  exitCode?: number
}

export interface ServerProfile {
  baseUrl: string
  username: string
  clientId: string
  clientName: string
  workspace: string
  modelId: number
}

export interface ConnectionState {
  loggedIn: boolean
  username: string
  baseUrl: string
  clientId: string
  wsOnline: boolean
  message: string
}

/** 本机自定义技能。 */
export interface LocalSkill {
  id: string
  name: string
  description: string
  path: string
}
