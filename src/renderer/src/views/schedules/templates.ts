import { blankDraft, defaultRule, type ScheduleDraft, type ScheduleRule } from '@shared/schedule'

export interface ScheduleTemplate {
  id: string
  name: string
  description: string
  prompt: string
  rule: ScheduleRule
}

function rule(partial: Partial<ScheduleRule>): ScheduleRule {
  return { ...defaultRule(), ...partial }
}

export const scheduleTemplates: ScheduleTemplate[] = [
  {
    id: 'ai-news',
    name: '每日 AI 新闻推送',
    description: '每个工作日早上汇总昨天到现在的行业动态',
    prompt:
      '整理昨天到现在的 AI 行业要闻，覆盖模型发布、产品更新和重要论文。输出一份简报：每条包含标题和一两句摘要，并按重要性排序。不要编造来源。',
    rule: rule({ repeat: 'weekly', weekdays: [1, 2, 3, 4, 5], time: '09:00' })
  },
  {
    id: 'weekly-report',
    name: '每周工作周报',
    description: '周五下午根据工作空间里的改动生成本周周报',
    prompt:
      '根据当前工作空间里本周改动过的文件，生成本周工作周报。包含完成事项、进行中事项和风险。用条目列出，不要扩大到工作空间以外的内容。',
    rule: rule({ repeat: 'weekly', weekdays: [5], time: '17:00' })
  },
  {
    id: 'meeting-prep',
    name: '会议前准备',
    description: '周一早上整理待办、未完成修改和需要确认的问题',
    prompt:
      '查看当前工作空间，整理今天可能要用的会议材料：待办、未完成修改和需要确认的问题。输出一份会前清单。',
    rule: rule({ repeat: 'weekly', weekdays: [1], time: '08:30' })
  }
]

export function draftFromTemplate(
  template: ScheduleTemplate,
  extras: {
    agentId: string
    agentName: string
    modelId: number | null
    workspace: string
  }
): ScheduleDraft {
  const draft = blankDraft()
  draft.name = template.name
  draft.prompt = template.prompt
  draft.agentId = extras.agentId
  draft.agentName = extras.agentName
  draft.modelId = extras.modelId
  draft.workspace = extras.workspace
  draft.rule = {
    ...template.rule,
    weekdays: [...template.rule.weekdays],
    monthDays: [...template.rule.monthDays]
  }
  return draft
}
