import type { ConnectorInput, ConnectorView } from '../../types'

const emptySpec = '{\n  \n}\n'

export function emptyConnectorSpec() {
  return emptySpec
}

/** 一条连接器在编辑框里的配置，不含名称、说明和启用状态。 */
export function connectorToSpec(item: ConnectorView) {
  const spec: Record<string, unknown> = {}
  if (item.transport === 'stdio') {
    spec.command = item.command
    if (item.args?.length) spec.args = item.args
    if (item.env && Object.keys(item.env).length) spec.env = item.env
  } else if (item.url) {
    spec.url = item.url
    const headers = { ...(item.headers ?? {}) }
    if (Object.keys(headers).length) spec.headers = headers
  }
  return `${JSON.stringify(spec, null, 2)}\n`
}

function asRecord(value: unknown, label: string) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${label}必须是 JSON 对象`)
  }
  return value as Record<string, unknown>
}

function strMap(value: unknown) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  const out: Record<string, string> = {}
  for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
    if (key.trim()) out[key] = item == null ? '' : String(item)
  }
  return out
}

function strList(value: unknown) {
  if (!Array.isArray(value)) return []
  return value.map((item) => String(item ?? '').trim()).filter(Boolean)
}

/** 拆出这一条服务的配置。整段 mcpServers 只允许一个服务，并带回它的 key。 */
export function readConnectorSpec(raw: string) {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    throw new Error('JSON 格式不正确')
  }
  const doc = asRecord(parsed, '配置')
  if (!('mcpServers' in doc)) {
    return { pastedName: '', spec: doc }
  }
  const servers = asRecord(doc.mcpServers, 'mcpServers')
  const entries = Object.entries(servers).filter(([key]) => key.trim())
  if (!entries.length) throw new Error('mcpServers 里需要一个服务')
  if (entries.length > 1) throw new Error('一次只能配置一个 MCP 服务')
  const first = entries[0]
  if (!first) throw new Error('mcpServers 里需要一个服务')
  const [pastedName, specValue] = first
  return { pastedName: pastedName.trim(), spec: asRecord(specValue, `「${pastedName.trim()}」`) }
}

export function formatSpec(spec: Record<string, unknown>) {
  return `${JSON.stringify(spec, null, 2)}\n`
}

/** 把手填名称和 JSON 配置还原成一条连接器。名称为空时采用粘贴文档里的服务名。 */
export function parseConnectorDraft(raw: string, name: string, description: string, prev: ConnectorView | null): ConnectorInput {
  const { pastedName, spec } = readConnectorSpec(raw)
  const key = name.trim() || pastedName
  if (!key) throw new Error('请填写名称')
  const command = String(spec.command ?? '').trim()
  const url = String(spec.url ?? '').trim()
  if (!command && !url) throw new Error('JSON 需要 command 或 url')
  if (command && url) throw new Error('command 和 url 只能填一个')
  const headers = strMap(spec.headers)
  let authType = 'none'
  let token = ''
  let clearToken = false
  if (url) {
    const auth = headers.Authorization || headers.authorization || ''
    const bearer = auth.match(/^Bearer\s+(\S+)/i)
    if (bearer) {
      authType = 'bearer'
      token = bearer[1] ?? ''
      delete headers.Authorization
      delete headers.authorization
    } else if (Object.keys(headers).length) {
      authType = 'headers'
    } else if (prev?.authType === 'bearer' && prev.tokenSet) {
      authType = 'bearer'
    }
    if (prev?.authType === 'bearer' && prev.tokenSet && authType !== 'bearer') clearToken = true
  }
  return {
    id: prev?.id ?? '',
    name: key,
    description: description.trim(),
    enabled: prev?.enabled ?? true,
    transport: command ? 'stdio' : 'http',
    command,
    args: strList(spec.args),
    env: command ? strMap(spec.env) : {},
    url,
    authType,
    token,
    clearToken,
    headers: authType === 'headers' ? headers : {},
  }
}
