import { get } from '@/utils/request'
import type { SkillItem } from '@/types/api'

export function fetchAvailableSkills() {
  return get<{ list: SkillItem[] }>('/agents/skills')
}
