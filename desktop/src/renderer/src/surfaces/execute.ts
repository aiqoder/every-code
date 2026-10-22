import { reactive } from 'vue'
import { adeApi } from '../api'
import { messageOf, useSessionStore } from '../stores/session'
import type { PendingAskQuestion, Task } from '../../../shared/types'
import { asNumber, asText, briefTask } from './format'
import { openPath } from './nav'
import { fail, ok, emptyParams, params, type ActionResult, type PageSurface } from './types'

export const executePage = reactive({
  selectedId: 0,
  busy: false,
  /** 由 ExecuteChat 同步，供 surface 快照与 answer_question 使用。 */
  pendingAsk: null as PendingAskQuestion | null
})

function findTask(id: number): Task | undefined {
  return useSessionStore().allTasks.find((task) => task.id === id)
}

export function stopExecuteWatch(): void {
  /* 任务事件改由 ExecuteChat 订阅 */
}

export async function refreshSelected(): Promise<void> {
  if (!executePage.selectedId) return
  try {
    const task = await adeApi.getTask(executePage.selectedId)
    useSessionStore().upsertTask(task)
  } catch {
    /* 下一轮再试 */
  }
}

export async function selectTask(task: Task): Promise<ActionResult> {
  executePage.selectedId = task.id
  localStorage.setItem('ade:task', String(task.id))
  await refreshSelected()
  return ok(`已选中任务 #${task.id} ${task.name || ''}`.trim())
}

export async function startTask(task: Task): Promise<ActionResult> {
  if (executePage.busy) return fail('上一个操作还没结束')
  executePage.busy = true
  try {
    const updated = await adeApi.startTask(task.id)
    useSessionStore().upsertTask(updated)
    executePage.selectedId = updated.id
    return ok(`已开始执行任务 #${updated.id}`, '已开始执行')
  } catch (err) {
    return fail(messageOf(err))
  } finally {
    executePage.busy = false
  }
}

export async function retryTask(task: Task): Promise<ActionResult> {
  if (executePage.busy) return fail('上一个操作还没结束')
  executePage.busy = true
  try {
    const updated = await adeApi.retryTask(task.id)
    useSessionStore().upsertTask(updated)
    return ok(`任务 #${updated.id} 已继续执行`, '已继续执行')
  } catch (err) {
    return fail(messageOf(err))
  } finally {
    executePage.busy = false
  }
}

export async function answerQuestion(payload: {
  conversationId: number
  questionId: string
  answers: Array<{ questionId: string; selected: string[]; text?: string }>
}): Promise<ActionResult> {
  if (!executePage.pendingAsk) return fail('当前没有待回答的问题')
  executePage.pendingAsk = { ...executePage.pendingAsk, status: 'submitting' }
  try {
    await adeApi.answerQuestion(payload.conversationId, payload.questionId, payload.answers)
    executePage.pendingAsk = { ...executePage.pendingAsk, status: 'done' }
    setTimeout(() => {
      executePage.pendingAsk = null
    }, 600)
    return ok('已提交答案', '已提交答案')
  } catch (err) {
    executePage.pendingAsk = { ...executePage.pendingAsk, status: 'pending', error: messageOf(err) }
    return fail(messageOf(err))
  }
}

async function taskByArg(args: Record<string, unknown>): Promise<Task | ActionResult> {
  const task = findTask(asNumber(args.taskId))
  if (!task) return fail('没有这条任务')
  return task
}

function isTask(value: Task | ActionResult): value is Task {
  return typeof (value as Task).id === 'number' && Array.isArray((value as Task).todos)
}

export const executeSurface: PageSurface = {
  channel: 'execute',
  page: 'execute',
  title: '执行',
  snapshot() {
    const session = useSessionStore()
    const tasks = session.allTasks
    const selected = tasks.find((task) => task.id === executePage.selectedId) ?? null
    return {
      busy: executePage.busy,
      doneCount: tasks.reduce((sum, task) => sum + task.doneCount, 0),
      totalCount: tasks.reduce((sum, task) => sum + task.totalCount, 0),
      selected: selected ? briefTask(selected) : null,
      pendingAsk: executePage.pendingAsk
        ? {
            conversationId: executePage.pendingAsk.conversationId,
            questionId: executePage.pendingAsk.questionId,
            title: executePage.pendingAsk.title,
            questions: executePage.pendingAsk.questions
          }
        : null,
      tasks: tasks.map(briefTask)
    }
  },
  actions: [
    {
      name: 'select_task',
      description: '选中一条任务并打开它的详情。',
      parameters: params({ taskId: { type: 'number' } }, ['taskId']),
      async run(args) {
        const task = await taskByArg(args)
        if (!isTask(task)) return task
        return selectTask(task)
      }
    },
    {
      name: 'start_task',
      description: '开始执行一条已拆解、还有未完成待办的任务。',
      parameters: params({ taskId: { type: 'number' } }, ['taskId']),
      async run(args) {
        const task = await taskByArg(args)
        if (!isTask(task)) return task
        if (task.totalCount <= 0) return fail('计划还没拆完')
        if (task.doneCount >= task.totalCount) return fail('待办已完成')
        return startTask(task)
      }
    },
    {
      name: 'retry_task',
      description: '继续执行还有未完成待办的任务。',
      parameters: params({ taskId: { type: 'number' } }, ['taskId']),
      async run(args) {
        const task = await taskByArg(args)
        if (!isTask(task)) return task
        return retryTask(task)
      }
    },
    {
      name: 'open_plan',
      description: '打开这条任务的计划页。',
      parameters: params({ taskId: { type: 'number' } }, ['taskId']),
      async run(args) {
        const id = asNumber(args.taskId)
        if (!findTask(id)) return fail('没有这条任务')
        await openPath(`/tasks/${id}`)
        return ok(`已打开任务 #${id} 的计划页`)
      }
    },
    {
      name: 'answer_question',
      description: '回答执行页上正在等待的问题。',
      parameters: params(
        {
          conversationId: { type: 'number' },
          questionId: { type: 'string' },
          answers: {
            type: 'array',
            items: {
              type: 'object',
              properties: {
                questionId: { type: 'string' },
                selected: { type: 'array', items: { type: 'string' } },
                text: { type: 'string' }
              },
              required: ['questionId', 'selected']
            }
          }
        },
        ['conversationId', 'questionId', 'answers']
      ),
      async run(args) {
        const answers = Array.isArray(args.answers) ? args.answers : []
        return answerQuestion({
          conversationId: asNumber(args.conversationId),
          questionId: asText(args.questionId),
          answers: answers
            .filter((item) => item && typeof item === 'object')
            .map((item) => {
              const row = item as Record<string, unknown>
              return {
                questionId: asText(row.questionId),
                selected: Array.isArray(row.selected) ? row.selected.map((value) => String(value)) : [],
                text: asText(row.text)
              }
            })
        })
      }
    },
    {
      name: 'refresh',
      description: '刷新当前选中任务。',
      parameters: emptyParams(),
      async run() {
        await refreshSelected()
        return ok('已刷新')
      }
    }
  ]
}
