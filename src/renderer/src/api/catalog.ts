import { listAllLlmModels } from '@/api/llm'
import { isProviderEnabled } from '@/lib/providerEnabled'
import request from '@/utils/request'

export interface AgentItem {
  id: number
  /** 用户填写的标识，只含字母，多段用中划线连接。聊天请求用这个，不用数字主键。 */
  agentId: string
  name: string
  description: string
}

export interface ModelItem {
  id: number
  name: string
  providerId?: number
  providerName?: string
}

interface Page<T> {
  list?: T[]
}

export function listAgents(): Promise<AgentItem[]> {
  return request.get('/agents').then((data) => {
    const page = data as unknown as Page<Partial<AgentItem>> | Partial<AgentItem>[]
    const list = Array.isArray(page) ? page : page.list || []
    return list.flatMap((item) => {
      if (!item || typeof item.id !== 'number') return []
      return [
        {
          id: item.id,
          agentId: typeof item.agentId === 'string' ? item.agentId.trim() : '',
          name: item.name || '',
          description: item.description || ''
        }
      ]
    })
  })
}

/** 把本地缓存换成服务端标识。旧缓存可能是数字主键。 */
export function resolveAgentKey(stored: string, agents: AgentItem[]): string {
  const current = stored.trim()
  const bySlug = agents.find((item) => item.agentId && item.agentId === current)
  if (bySlug) return bySlug.agentId
  if (/^\d+$/.test(current)) {
    const byId = agents.find((item) => String(item.id) === current && item.agentId)
    if (byId) return byId.agentId
    if (agents.length === 0) return current
  } else if (current) {
    return current
  }
  return agents.find((item) => item.agentId)?.agentId || ''
}

export async function listModels(): Promise<ModelItem[]> {
  const list = await listAllLlmModels()
  return list
    .filter((item) => isProviderEnabled(item.providerId))
    .map((item) => ({
      id: item.id,
      name: item.name,
      providerId: item.providerId,
      providerName: item.providerName || undefined
    }))
}
