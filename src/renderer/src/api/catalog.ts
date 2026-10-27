import request from '@/utils/request'

export interface AgentItem {
  id: number
  name: string
  description: string
}

export interface ModelItem {
  id: number
  name: string
  providerName?: string
}

interface Page<T> {
  list?: T[]
}

export function listAgents(): Promise<AgentItem[]> {
  return request.get('/agents').then((data) => {
    const page = data as unknown as Page<AgentItem> | AgentItem[]
    return Array.isArray(page) ? page : page.list || []
  })
}

export function listModels(): Promise<ModelItem[]> {
  return request.get('/llm-models', { params: { page: 1, pageSize: 100 } }).then((data) => {
    const page = data as unknown as Page<ModelItem> | ModelItem[]
    return Array.isArray(page) ? page : page.list || []
  })
}
