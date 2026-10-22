/** 消息中的技能指令：/skill:<id>（id 可含冒号，如 builtin:git-commit-zh） */
export const SKILL_TOKEN_RE = /\/skill:([^\s]+)/g

export type SkillRef = {
  id: string
  name: string
  description?: string
  source?: string
}

export function extractSkillIds(content: string): string[] {
  const ids: string[] = []
  const seen = new Set<string>()
  const re = new RegExp(SKILL_TOKEN_RE.source, 'g')
  let m: RegExpExecArray | null
  while ((m = re.exec(content)) !== null) {
    const id = m[1]
    if (!id || seen.has(id)) continue
    seen.add(id)
    ids.push(id)
  }
  return ids
}

/** 去掉技能 token，保留用户任务正文（供本地预览等；服务端 LLM 也会 strip）。 */
export function stripSkillTokens(content: string): string {
  return content
    .replace(SKILL_TOKEN_RE, ' ')
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n[ \t]+/g, '\n')
    .replace(/[ \t]{2,}/g, ' ')
    .trim()
}

export type ContentPart =
  | { type: 'text'; text: string }
  | { type: 'skill'; id: string }

/** 将含 /skill:id 的字符串拆成文本与技能片段，便于气泡渲染。 */
export function parseContentParts(content: string): ContentPart[] {
  const parts: ContentPart[] = []
  const re = new RegExp(SKILL_TOKEN_RE.source, 'g')
  let last = 0
  let m: RegExpExecArray | null
  while ((m = re.exec(content)) !== null) {
    if (m.index > last) {
      parts.push({ type: 'text', text: content.slice(last, m.index) })
    }
    const id = m[1]
    if (id) parts.push({ type: 'skill', id })
    last = m.index + m[0].length
  }
  if (last < content.length) {
    parts.push({ type: 'text', text: content.slice(last) })
  }
  return parts
}

export function skillToken(id: string): string {
  return `/skill:${id}`
}

export const SKILL_PREVIEW_LIMIT = 20

/** 技能来源的展示名，桌面/网页/移动端保持一致。 */
export function skillScopeLabel(source?: string): string {
  if (source === 'builtin') return '内置'
  if (source === 'claude') return 'Claude Code'
  if (source === 'project') return '项目'
  if (source === 'pkg') return '智能体包'
  return '本地'
}

export function filterSkills(list: SkillRef[], query: string, limit?: number): SkillRef[] {
  const q = query.trim().toLowerCase()
  let rows = list
  if (q) {
    rows = list.filter((s) => {
      const hay = `${s.name} ${s.description || ''} ${s.id}`.toLowerCase()
      return hay.includes(q)
    })
  }
  if (limit != null && limit >= 0) return rows.slice(0, limit)
  return rows
}
