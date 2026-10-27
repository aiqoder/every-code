import request from '@/utils/request'

export interface Conversation {
  id: number
  title: string
  modelId: number
  modelName: string
  clientId: string
  workspace: string
  updatedAt?: string
}

export interface ChatMessage {
  id: number
  role: string
  content: string
  toolCallId: string
  toolName: string
  metadata: string
}

export interface ConversationDetail extends Conversation {
  messages: ChatMessage[]
}

interface Page<T> {
  list?: T[]
}

export function listConversations(clientId: string): Promise<Conversation[]> {
  return request
    .get('/conversations', { params: { page: 1, pageSize: 50, clientId } })
    .then((data) => {
      const page = data as unknown as Page<Conversation>
      return page.list || []
    })
}

export function getConversation(id: number): Promise<ConversationDetail> {
  return request.get(`/conversations/${id}`).then((data) => {
    const detail = data as unknown as ConversationDetail
    detail.messages = detail.messages || []
    return detail
  })
}

export function createConversation(input: {
  modelId: number
  clientId: string
  title: string
  workspace: string
}): Promise<Conversation> {
  return request.post('/conversations', input) as unknown as Promise<Conversation>
}

export function deleteConversation(id: number): Promise<void> {
  return request.delete(`/conversations/${id}`) as unknown as Promise<void>
}
