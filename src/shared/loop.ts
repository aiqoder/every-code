/** 与 chengzi-agent-server pkg/clienthub/protocol 对齐的 WebSocket 帧。 */

export interface CallMetadata {
  id?: string
  agentId?: string
  name?: string
  description?: string
  folder?: string
}

export interface LoopMessage {
  type: string
  id?: string
  metadata?: CallMetadata
  payload?: unknown
  data?: string
  error?: string
}

export type LoopSkillSource = 'claude' | 'project' | 'agents' | 'package'

export type LoopToolSource = 'builtin' | 'user' | 'package'

export interface LoopToolInfo {
  name: string
  description: string
  source: LoopToolSource
}

export interface LoopSkillInfo {
  id: string
  name: string
  description: string
  folder: string
  source: LoopSkillSource
}

export interface LoopPackageSkill {
  name: string
  description: string
  folder: string
}

export interface LoopPackageToolParam {
  name: string
  type: string
  description: string
  required: boolean
}

export interface LoopPackageTool {
  name: string
  description: string
  parameters: LoopPackageToolParam[]
}

export interface LoopPackageEnvItem {
  name: string
  label: string
  secret: boolean
  required: boolean
}

export interface LoopPackageEnvField {
  name: string
  label: string
  secret: boolean
  required: boolean
  /** 空字符串表示单行文本，list 表示用 | 分隔的多行清单。 */
  type: '' | 'list'
  items: LoopPackageEnvItem[]
  value: string
}

export interface LoopPackage {
  id: string
  name: string
  description: string
  version: string
  author: string
  avatar: string
  systemPrompt: string
  maxSteps: number
  remoteId: number | null
  skills: LoopPackageSkill[]
  tools: LoopPackageTool[]
  env: LoopPackageEnvField[]
  envReady: boolean
  installDir: string
}

export interface LoopAgentHome {
  toolsDir: string
  packages: LoopPackage[]
  toolNotes: string[]
}

export interface LoopInventory {
  toolsDir: string
  tools: LoopToolInfo[]
  skills: LoopSkillInfo[]
  toolNotes: string[]
}

export const LoopType = {
  toolCall: 'tool_call',
  toolResult: 'tool_result',
  skillCall: 'skill_call',
  skillResult: 'skill_result',
  skillRunner: 'skill_runner',
  skillRunnerResult: 'skill_runner_result',
  skillLoad: 'skill_load',
  skillContent: 'skill_content',
  ping: 'ping',
  pong: 'pong',
  error: 'error'
} as const
