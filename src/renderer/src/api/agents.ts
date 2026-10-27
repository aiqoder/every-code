import request from '@/utils/request'

/** 与 chengzi-agent-server SaveAgentReq / ToolInput / SkillInput 对齐。PUT 会整表替换工具和技能。 */

export interface AgentHttpHeader {
  key: string
  value: string
}

export interface AgentHttpConfig {
  url: string
  headers: AgentHttpHeader[]
}

export interface AgentParamInput {
  name: string
  paramType: string
  description: string
  isRequired: boolean
  enumValue: string
  defaultValue: string
  itemType: string
  children: AgentParamInput[]
}

export interface AgentToolInput {
  id: number
  name: string
  description: string
  protocol: string
  parameters: AgentParamInput[]
  http?: AgentHttpConfig
}

export interface AgentSkillInput {
  id: number
  name: string
  description: string
  protocol: string
  folder: string
  http?: AgentHttpConfig
}

export interface AgentSaveBody {
  name: string
  description: string
  systemPrompt: string
  avatar: string
  maxSteps: number
  version: string
  tools: AgentToolInput[]
  skills: AgentSkillInput[]
}

interface AgentParamView {
  name?: string
  paramType?: string
  description?: string
  isRequired?: boolean
  enumValue?: string
  defaultValue?: string
  itemType?: string
  children?: AgentParamView[]
}

interface AgentHttpView {
  url?: string
  headers?: { key?: string; value?: string }[] | null
}

interface AgentToolView {
  id?: number
  name?: string
  description?: string
  protocol?: string
  parameters?: AgentParamView[] | null
  http?: AgentHttpView | null
}

interface AgentSkillView {
  id?: number
  name?: string
  description?: string
  protocol?: string
  folder?: string
  http?: AgentHttpView | null
}

export interface AgentDetail {
  id: number
  name: string
  description: string
  systemPrompt: string
  avatar: string
  maxSteps: number
  version: string
  tools: AgentToolView[]
  skills: AgentSkillView[]
}

export interface LocalSkillDraft {
  name: string
  description: string
  folder: string
}

export async function getAgent(id: number): Promise<AgentDetail> {
  const data = (await request.get(`/agents/${id}`)) as Partial<AgentDetail> | null
  return normalizeAgent(data, id)
}

export async function createAgent(body: AgentSaveBody): Promise<AgentDetail> {
  const data = (await request.post('/agents', body)) as Partial<AgentDetail> | null
  return normalizeAgent(data, 0)
}

function normalizeAgent(data: Partial<AgentDetail> | null, fallbackId: number): AgentDetail {
  if (!data || typeof data !== 'object') throw new Error('智能体不存在')
  return {
    id: typeof data.id === 'number' ? data.id : fallbackId,
    name: data.name ?? '',
    description: data.description ?? '',
    systemPrompt: data.systemPrompt ?? '',
    avatar: data.avatar ?? '',
    maxSteps:
      typeof data.maxSteps === 'number' && Number.isFinite(data.maxSteps) ? data.maxSteps : 0,
    version: data.version == null ? '' : String(data.version),
    tools: Array.isArray(data.tools) ? data.tools : [],
    skills: Array.isArray(data.skills) ? data.skills : []
  }
}

export function updateAgent(id: number, body: AgentSaveBody): Promise<AgentDetail> {
  return request.put(`/agents/${id}`, body) as unknown as Promise<AgentDetail>
}

/** 把 GET 结果映射回 PUT 体，并在末尾追加新技能。已有工具原样带回，不新增本地工具。 */
export function toAgentSaveBody(agent: AgentDetail, extraSkills: AgentSkillInput[]): AgentSaveBody {
  return {
    name: agent.name ?? '',
    description: agent.description ?? '',
    systemPrompt: agent.systemPrompt ?? '',
    avatar: agent.avatar ?? '',
    maxSteps:
      typeof agent.maxSteps === 'number' && Number.isFinite(agent.maxSteps) ? agent.maxSteps : 0,
    version: agent.version == null ? '' : String(agent.version),
    tools: agent.tools.map(mapTool),
    skills: [...agent.skills.map(mapSkill), ...extraSkills]
  }
}

/** 跳过智能体上已有同名或同目录的技能。新技能 id 为 0，协议为 websocket。 */
export function skillsToAttach(agent: AgentDetail, selected: LocalSkillDraft[]): AgentSkillInput[] {
  const names = new Set<string>()
  const folders = new Set<string>()
  for (const skill of agent.skills) {
    const name = (skill.name ?? '').trim()
    if (name) names.add(name)
    const folder = folderKey(skill.folder ?? '')
    if (folder) folders.add(folder)
  }
  const extra: AgentSkillInput[] = []
  for (const skill of selected) {
    const name = skill.name.trim()
    const folder = folderKey(skill.folder)
    if ((name && names.has(name)) || (folder && folders.has(folder))) continue
    extra.push({
      id: 0,
      name: skill.name,
      description: skill.description,
      protocol: 'websocket',
      folder: skill.folder
    })
  }
  return extra
}

function mapTool(tool: AgentToolView): AgentToolInput {
  const protocol = tool.protocol ?? ''
  const mapped: AgentToolInput = {
    id: mapId(tool.id),
    name: tool.name ?? '',
    description: tool.description ?? '',
    protocol,
    parameters: mapParams(tool.parameters)
  }
  if (isHttp(protocol) && tool.http) mapped.http = mapHttp(tool.http)
  return mapped
}

function mapSkill(skill: AgentSkillView): AgentSkillInput {
  const protocol = skill.protocol ?? ''
  const mapped: AgentSkillInput = {
    id: mapId(skill.id),
    name: skill.name ?? '',
    description: skill.description ?? '',
    protocol,
    folder: skill.folder ?? ''
  }
  if (isHttp(protocol) && skill.http) mapped.http = mapHttp(skill.http)
  return mapped
}

function mapParams(list: AgentParamView[] | null | undefined): AgentParamInput[] {
  if (!Array.isArray(list)) return []
  return list.map((param) => ({
    name: param.name ?? '',
    paramType: param.paramType ?? '',
    description: param.description ?? '',
    isRequired: param.isRequired === true,
    enumValue: param.enumValue ?? '',
    defaultValue: param.defaultValue ?? '',
    itemType: param.itemType ?? '',
    children: mapParams(param.children)
  }))
}

function mapHttp(http: AgentHttpView): AgentHttpConfig {
  const headers = Array.isArray(http.headers) ? http.headers : []
  return {
    url: http.url ?? '',
    headers: headers.map((header) => ({
      key: header?.key ?? '',
      value: header?.value ?? ''
    }))
  }
}

function mapId(id: unknown): number {
  return typeof id === 'number' && Number.isFinite(id) ? id : 0
}

function isHttp(protocol: string): boolean {
  return protocol.trim().toLowerCase() === 'http'
}

function folderKey(folder: string): string {
  return folder.trim().replace(/[\\/]+$/, '')
}
