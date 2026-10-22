import { get, put } from '@/utils/request'

export interface SystemSkill {
  id: string
  name: string
  description: string
  fileCount: number
  updatedAt: string
  agentId: string
}

export interface AgentOption {
  id: string
  name: string
}

export function fetchSystemSkills() {
  return get<{ list: SystemSkill[]; agents: AgentOption[] }>('/admin/system-skills')
}

export function saveSystemSkills(list: Pick<SystemSkill, 'id' | 'agentId'>[]) {
  return put<{ list: SystemSkill[] }>('/admin/system-skills', {
    list: list.map((row) => ({ skillId: row.id, agentId: row.agentId ?? '' })),
  })
}
