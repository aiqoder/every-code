export interface Status {
  loggedIn: boolean
  connected: boolean
  serverUrl: string
  clientId: string
  clientName: string
  workspace: string
  workspaces?: string[]
  username: string
  userName: string
  userRole: string
  lastError: string
  socketUrl?: string
  rev?: number
  catalog?: {
    tools?: { name: string; displayName?: string; description: string; parameters?: Record<string, unknown> }[]
    skills?: { id: string; name: string; description: string }[]
  }
}

export interface ConnectorInput {
  id: string
  name: string
  description: string
  enabled: boolean
  transport: string
  command: string
  args: string[]
  env: Record<string, string>
  url: string
  authType: string
  token: string
  clearToken: boolean
  headers: Record<string, string>
}

export interface ConnectorView {
  id: string
  name: string
  description: string
  enabled: boolean
  transport: string
  command: string
  args: string[] | null
  env: Record<string, string> | null
  url: string
  authType: string
  tokenSet: boolean
  headers: Record<string, string> | null
  status: string
  toolCount: number
  lastError: string
  tools: { name: string; description: string }[] | null
}

export interface SkillItem {
  id: string
  name: string
  description: string
  source: string
  content: string
  path: string
  editable?: boolean
}

export interface Model {
  id: number
  userId?: number | null
  providerId?: number
  providerName: string
  name: string
  attachment?: boolean
  reasoning?: boolean
  reasoningOptions?: string[]
  temperature?: boolean
  inputCapabilities?: string[]
  inputContextLimit?: number
  outputContextLimit?: number
  inputPrice?: number
  outputPrice?: number
  cachePrice?: number
  editable?: boolean
}

export interface ModelPayload {
  providerId: number
  name: string
  attachment: boolean
  reasoning: boolean
  reasoningOptions: string[]
  temperature: boolean
  inputCapabilities: string[]
  inputContextLimit: number
  outputContextLimit: number
  inputPrice: number
  outputPrice: number
  cachePrice: number
}

export interface LlmProvider {
  id: number
  userId?: number | null
  name: string
  website?: string
  apiKey?: string
  openaiCompatibleBaseUrl?: string
  anthropicBaseUrl?: string
  editable?: boolean
}

export interface ProviderPayload {
  name: string
  website: string
  apiKey: string
  openaiCompatibleBaseUrl: string
  anthropicBaseUrl: string
}

export interface Agent {
  id: string
  name: string
  description?: string
  systemPrompt?: string
  skills?: string[]
  avatar?: string
  maxSteps?: number
  version?: string
}

export interface AgentPayload {
  name: string
  description?: string
  systemPrompt?: string
  skills?: string[]
  avatar?: string
  maxSteps?: number
  version?: string
}

export interface ToolItem {
  id?: number
  userId?: number | null
  clientId?: string
  clientName?: string
  name: string
  displayName: string
  description?: string
  parameters?: Record<string, unknown>
  type: string
  extra?: Record<string, unknown>
  builtin?: boolean
  editable?: boolean
  version?: number
}

export interface ToolPayload {
  name: string
  displayName: string
  description?: string
  parameters?: Record<string, unknown>
  type?: string
  extra?: Record<string, unknown>
  version?: number
}

export interface AgentSkillOption {
  id: string
  name: string
  description: string
  source: string
}

// Conversation 会话。agentId 为本机侧归属的智能体安装包 ID（空表示未归属）。
export type TokenUsageRange = 'today' | '7d' | '30d'

export interface TokenUsageTrendPoint {
  at: string
  promptTokens: number
  completionTokens: number
  cacheCreationTokens: number
  cacheReadTokens: number
  cost: number
}

export interface TokenUsageStats {
  promptTokens: number
  completionTokens: number
  cacheCreationTokens: number
  cacheReadTokens: number
  realTokens: number
  requestCount: number
  cost: number
  cacheHitRate: number
  range: TokenUsageRange
  trend: TokenUsageTrendPoint[]
}

export interface Conversation {
  id: number
  title: string
  modelId: number
  modelName: string
  clientId: string
  workspace?: string
  agentId?: string
  contextUsage?: Record<string, unknown> | null
}

export interface ChatMessage {
  id: number
  role: string
  content: string
  toolCallId?: string
  toolName?: string
  metadata?: string
}

export interface ConversationDetail extends Conversation {
  messages: ChatMessage[]
}

export interface ChatEvent {
  type: string
  text?: string
  toolCallId?: string
  toolName?: string
  arguments?: string
  result?: string
  plan?: unknown
  goal?: string
  error?: string
  data?: Record<string, unknown>
}

export type TodoStepStatus = 'pending' | 'in_progress' | 'done' | 'cancelled'

export interface TodoStep {
  id: string
  title: string
  status: TodoStepStatus
}

export interface TodoPlan {
  title: string
  steps: TodoStep[]
}

export interface RevertSummary {
  files: number
  additions: number
  deletions: number
  fileList: string[]
}

export interface PackageEnvItem {
  name: string
  label: string
  secret: boolean
  required: boolean
}

export interface PackageEnvField {
  name: string
  label: string
  secret: boolean
  required: boolean
  type?: string
  items?: PackageEnvItem[]
  value: string
}

export interface PackageInfo {
  id: string
  name: string
  description: string
  version: string
  author: string
  agentId: number
  env?: PackageEnvField[]
  envReady: boolean
  skills?: string[]
  /** 包内头像的 data URL；无头像时为空 */
  avatar?: string
}

export interface TimelineQuestionOption {
  id: string
  label: string
}

export interface TimelineQuestionItem {
  id: string
  prompt: string
  allowMultiple: boolean
  options: TimelineQuestionOption[]
}

export interface TimelineQuestion {
  questionId: string
  title: string
  questions: TimelineQuestionItem[]
  status: 'pending' | 'submitting' | 'done' | 'timeout'
  /** 作答截止时间（Unix 毫秒），用于前端倒计时 */
  deadlineAt?: number
  /** 初始等待秒数（便于展示） */
  timeoutSec?: number
  error?: string
}

export interface QuestionAnswerPayload {
  toolCallId: string
  questionId: string
  answers: { questionId: string; selected: string[]; text?: string }[]
}

export interface TimelineItem {
  kind: 'message' | 'tool'
  id: string
  role?: string
  content?: string
  reasoning?: string
  reasoningStreaming?: boolean
  streaming?: boolean
  toolName?: string
  displayName?: string
  arguments?: string
  result?: string
  status?: 'running' | 'done' | 'error'
  question?: TimelineQuestion
}
