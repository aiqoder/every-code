export type PickedConnector = {
  id: string
  name: string
}

const key = (conversationId: number) => `chengzi.connectors:${conversationId}`
const agentKey = (agentId: string) => `chengzi.agent.connectors:${agentId}`

function readList(storageKey: string): PickedConnector[] {
  try {
    const raw = localStorage.getItem(storageKey)
    if (!raw) return []
    const parsed = JSON.parse(raw) as PickedConnector[]
    if (!Array.isArray(parsed)) return []
    return parsed.filter((item) => item && item.id && item.name)
  } catch {
    return []
  }
}

export function loadPickedConnectors(conversationId: number): PickedConnector[] {
  if (!conversationId) return []
  return readList(key(conversationId))
}

export function loadAgentConnectors(agentId: string | null | undefined): PickedConnector[] {
  if (!agentId) return []
  return readList(agentKey(agentId))
}

export function saveAgentConnectors(agentId: string | null | undefined, items: PickedConnector[]) {
  if (!agentId) return
  localStorage.setItem(agentKey(agentId), JSON.stringify(items))
}

export function savePickedConnectors(conversationId: number, items: PickedConnector[]) {
  if (!conversationId) return
  localStorage.setItem(key(conversationId), JSON.stringify(items))
}

/** 把本轮选中的连接器附在消息末尾，服务端据此开放对应工具，并在入库前去掉。 */
export function withConnectorTokens(content: string, items: PickedConnector[]) {
  const tokens = items.map((item) => `/cx:${item.id}`).filter((token) => !content.includes(token))
  if (!tokens.length) return content
  return `${content}\n${tokens.join(' ')}`
}
