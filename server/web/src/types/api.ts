/** 与后端 pkg/response 对齐的统一 JSON 信封 */
export interface ApiEnvelope<T = unknown> {
  code: 0 | 1
  msg: string
  data: T
  sysMsg: string
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

export interface User {
  id: number
  username: string
  name: string
  role: 'admin' | 'user' | string
  roles: string[]
}

export interface AuthResult {
  token: string
  user: User
}

/** OAuth2 授权地址 */
export interface OAuthAuthorizeResult {
  url: string
  state: string
}

/** OAuth2 授权码回调结果 */
export interface OAuthCallbackResult {
  token: string
  user: User
  redirect: string
}

export interface LlmProvider {
  id: number
  userId?: number | null
  name: string
  website: string
  apiKey: string
  openaiCompatibleBaseUrl: string
  anthropicBaseUrl: string
  editable?: boolean
  createdAt: string
  updatedAt: string
}

export interface LlmProviderPayload {
  name: string
  website?: string
  apiKey?: string
  openaiCompatibleBaseUrl?: string
  anthropicBaseUrl?: string
}

export interface LlmModel {
  id: number
  userId?: number | null
  providerId: number
  providerName: string
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
  editable?: boolean
  createdAt: string
  updatedAt: string
}

export interface LlmModelPayload {
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

export interface Conversation {
  id: number
  title: string
  modelId: number
  modelName: string
  providerName: string
  clientId: string
  workspace?: string
  agentId?: string
  contextUsage?: Record<string, unknown> | null
  createdAt: string
  updatedAt: string
}

export interface ChatMessage {
  id: number
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  toolCallId: string
  toolName: string
  metadata: string
  createdAt: string
}

export interface ConversationDetail extends Conversation {
  messages: ChatMessage[]
}

export type FlowEventType =
  | 'run_started'
  | 'run_finished'
  | 'error'
  | 'text_delta'
  | 'reasoning_delta'
  | 'tool_call_start'
  | 'tool_call_end'
  | 'plan_updated'
  | 'subagent_started'
  | 'subagent_finished'
  | 'ask_question'
  | 'context_usage'
  | 'context_compacted'
  | 'llm_retrying'
  | 'run_warning'

export interface FlowEvent {
  type: FlowEventType
  text?: string
  toolCallId?: string
  toolName?: string
  arguments?: string
  result?: string
  plan?: unknown
  goal?: string
  parentToolCallId?: string
  error?: string
  data?: Record<string, unknown>
}

export interface SkillItem {
  id: string
  name: string
  description: string
  source: 'claude' | 'project' | string
}

export interface Agent {
  id: string
  userId?: number | null
  name: string
  description: string
  systemPrompt: string
  skills: string[]
  avatar: string
  maxSteps?: number
  version?: string
  creatorName?: string
  creatorOnline?: boolean
  editable: boolean
  createdAt: string
  updatedAt: string
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

export interface Tool {
  id?: number
  userId?: number | null
  clientId?: string
  clientName?: string
  name: string
  displayName: string
  description: string
  parameters: Record<string, unknown>
  type: string
  extra: Record<string, unknown>
  builtin: boolean
  editable: boolean
  version?: number
  createdAt?: string
  updatedAt?: string
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

