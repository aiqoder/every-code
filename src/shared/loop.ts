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
