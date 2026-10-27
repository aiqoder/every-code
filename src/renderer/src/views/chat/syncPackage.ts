import {
  createAgent,
  getAgent,
  toAgentSaveBody,
  updateAgent,
  type AgentDetail,
  type AgentSaveBody,
  type AgentSkillInput,
  type AgentToolInput
} from '@/api/agents'
import type { LoopPackage } from '@shared/loop'

/** 把已安装插件同步成服务器上的智能体。技能和工具都走 websocket，由这台电脑执行。 */
export async function syncPackageAgent(pkg: LoopPackage): Promise<number> {
  if (pkg.remoteId) {
    try {
      const current = await getAgent(pkg.remoteId)
      await updateAgent(pkg.remoteId, mergePackage(current, pkg))
      return pkg.remoteId
    } catch (err) {
      if (!isMissing(err)) throw err
    }
  }
  const created = await createAgent(freshBody(pkg))
  if (!created.id) throw new Error('创建智能体后没有返回编号')
  return created.id
}

function freshBody(pkg: LoopPackage): AgentSaveBody {
  return {
    agentId: packageAgentSlug(pkg),
    name: pkg.name,
    description: pkg.description,
    systemPrompt: pkg.systemPrompt,
    avatar: '',
    maxSteps: pkg.maxSteps,
    version: pkg.version,
    tools: toolInputs(pkg, []),
    skills: skillInputs(pkg)
  }
}

function mergePackage(current: AgentDetail, pkg: LoopPackage): AgentSaveBody {
  const body = toAgentSaveBody(current, [])
  if (!body.agentId.trim()) body.agentId = packageAgentSlug(pkg)
  body.name = pkg.name
  body.description = pkg.description
  body.systemPrompt = pkg.systemPrompt
  body.maxSteps = pkg.maxSteps
  body.version = pkg.version
  if (body.avatar.length > 1024) body.avatar = ''
  body.tools = mergeTools(body.tools, current, pkg)
  const names = new Set(pkg.skills.map((skill) => skill.name))
  const marker = `/packages/${pkg.id}/`
  body.skills = [
    ...body.skills.filter((skill) => {
      const folder = skill.folder.replace(/\\/g, '/')
      return !folder.includes(marker) && !names.has(skill.name)
    }),
    ...skillInputs(pkg)
  ]
  return body
}

function toolInputs(pkg: LoopPackage, current: AgentDetail['tools']): AgentToolInput[] {
  return pkg.tools.map((tool) => {
    const prev = current.find(
      (item) => item.name === tool.name && (item.protocol ?? '').toLowerCase() === 'websocket'
    )
    return {
      id: typeof prev?.id === 'number' ? prev.id : 0,
      name: tool.name,
      description: tool.description,
      protocol: 'websocket',
      parameters: (tool.parameters ?? []).map((param) => ({
        name: param.name,
        paramType: param.type,
        description: param.description,
        isRequired: param.required,
        enumValue: '',
        defaultValue: '',
        itemType: '',
        children: []
      }))
    }
  })
}

/** 保留智能体上不与插件重名的 HTTP 工具，插件工具改为 websocket。 */
function mergeTools(
  kept: AgentToolInput[],
  current: AgentDetail,
  pkg: LoopPackage
): AgentToolInput[] {
  const names = new Set(pkg.tools.map((tool) => tool.name))
  return [
    ...kept.filter((tool) => tool.protocol.toLowerCase() === 'http' && !names.has(tool.name)),
    ...toolInputs(pkg, current.tools)
  ]
}

function skillInputs(pkg: LoopPackage): AgentSkillInput[] {
  return pkg.skills
    .filter((skill) => skill.name.trim() && skill.folder.trim())
    .map((skill) => ({
      id: 0,
      name: skill.name,
      description: skill.description,
      protocol: 'websocket',
      folder: skill.folder
    }))
}

function isMissing(err: unknown): boolean {
  return err instanceof Error && err.message.includes('不存在')
}

const agentIdPattern = /^[A-Za-z]+(-[A-Za-z]+)*$/

/** 插件 id 转成服务端接受的标识：非字母收成中划线，数字丢掉。 */
function packageAgentSlug(pkg: { id: string; name: string }): string {
  return slugFrom(pkg.id) || slugFrom(pkg.name) || 'agent'
}

function slugFrom(raw: string): string {
  const trimmed = raw.trim()
  if (agentIdPattern.test(trimmed) && trimmed.length <= 64) return trimmed
  const parts = trimmed.split(/[^A-Za-z]+/).filter((part) => part.length > 0)
  let slug = parts.join('-')
  if (slug.length > 64) slug = slug.slice(0, 64).replace(/-+$/, '')
  return agentIdPattern.test(slug) ? slug : ''
}
