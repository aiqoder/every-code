import { reactive } from 'vue'
import { adeApi } from '../api'
import { messageOf, useSessionStore } from '../stores/session'
import { useShellStore } from '../stores/shell'
import type { PendingAskQuestion, Task } from '../../../shared/types'
import { asNumber, asText, briefTask, briefTodo } from './format'
import { fail, ok, params, type ActionResult, type PageSurface } from './types'

export const taskPage = reactive({
  task: null as Task | null,
  busy: false,
  pendingAsk: null as PendingAskQuestion | null
})

async function reload(id: number): Promise<Task> {
  const task = await adeApi.getTask(id)
  taskPage.task = task
  useSessionStore().upsertTask(task)
  return task
}

function current(): Task | null {
  return taskPage.task
}

export async function startCurrentTask(): Promise<ActionResult> {
  const task = current()
  if (!task) return fail('任务还没加载')
  if (taskPage.busy) return fail('上一个操作还没结束')
  taskPage.busy = true
  try {
    const updated = await adeApi.startTask(task.id)
    useSessionStore().upsertTask(updated)
    taskPage.task = updated
    if (updated.worktree) {
      await useShellStore().openTask(updated.id, updated.name, updated.worktree)
    }
    return ok(`已开始执行任务 #${updated.id}`, '已开始执行')
  } catch (err) {
    return fail(messageOf(err))
  } finally {
    taskPage.busy = false
  }
}

export async function retryCurrentTask(): Promise<ActionResult> {
  const task = current()
  if (!task) return fail('任务还没加载')
  if (taskPage.busy) return fail('上一个操作还没结束')
  taskPage.busy = true
  try {
    const updated = await adeApi.retryTask(task.id)
    useSessionStore().upsertTask(updated)
    taskPage.task = updated
    return ok(`任务 #${updated.id} 已改回重新执行`, '已改回重新执行', 'plain')
  } catch (err) {
    return fail(messageOf(err))
  } finally {
    taskPage.busy = false
  }
}

export async function saveTaskPlan(input: {
  name: string
  overview: string
  plan: string
  todos: Array<{ title: string; detail?: string; acceptance?: string; dependsOn?: string }>
}): Promise<ActionResult> {
  const task = current()
  if (!task) return fail('任务还没加载')
  const name = input.name.trim()
  const todos = input.todos
    .map((item) => ({
      title: item.title.trim(),
      detail: item.detail?.trim() ?? '',
      acceptance: item.acceptance?.trim() ?? '',
      dependsOn: item.dependsOn?.trim() ?? ''
    }))
    .filter((item) => item.title)
  if (!name) return fail('请填写任务名称')
  if (!todos.length) return fail('计划至少要有一条待办')
  taskPage.busy = true
  try {
    const updated = await adeApi.applyPlan(task.id, {
      name,
      overview: input.overview.trim(),
      plan: input.plan.trim(),
      todos
    })
    useSessionStore().upsertTask(updated)
    taskPage.task = updated
    return ok(`已写回任务「${updated.name}」的计划`, '计划已写回当前任务')
  } catch (err) {
    return fail(messageOf(err))
  } finally {
    taskPage.busy = false
  }
}

export async function updateTaskDescription(description: string): Promise<ActionResult> {
  const task = current()
  if (!task) return fail('任务还没加载')
  const text = description.trim()
  if (!text) return fail('需求不能为空')
  taskPage.busy = true
  try {
    const updated = await adeApi.updateDescription(task.id, text)
    useSessionStore().upsertTask(updated)
    taskPage.task = updated
    return ok('需求已保存', '需求已保存')
  } catch (err) {
    return fail(messageOf(err))
  } finally {
    taskPage.busy = false
  }
}

export async function addTaskTodo(input: {
  title: string
  detail?: string
  acceptance?: string
  dependsOn?: string
}): Promise<ActionResult> {
  const task = current()
  if (!task) return fail('任务还没加载')
  const title = input.title.trim()
  if (!title) return fail('待办标题不能为空')
  taskPage.busy = true
  try {
    await adeApi.addTodo(task.id, {
      title,
      detail: input.detail?.trim() ?? '',
      acceptance: input.acceptance?.trim() ?? '',
      dependsOn: input.dependsOn?.trim() ?? ''
    })
    await reload(task.id)
    return ok(`已添加待办「${title}」`, '已添加待办')
  } catch (err) {
    return fail(messageOf(err))
  } finally {
    taskPage.busy = false
  }
}

export async function removeTaskTodo(todoId: number): Promise<ActionResult> {
  const task = current()
  if (!task) return fail('任务还没加载')
  try {
    await adeApi.deleteTodo(todoId)
    await reload(task.id)
    return ok('待办已删除', '待办已删除', 'plain')
  } catch (err) {
    return fail(messageOf(err))
  }
}

export async function toggleTaskTodo(todoId: number, status: string): Promise<ActionResult> {
  const task = current()
  if (!task) return fail('任务还没加载')
  try {
    await adeApi.updateTodo(todoId, { status: status === 'done' ? 'todo' : 'done' })
    await reload(task.id)
    return ok(status === 'done' ? '已标为未完成' : '已标为完成')
  } catch (err) {
    return fail(messageOf(err))
  }
}

export async function answerTaskQuestion(payload: {
  conversationId: number
  questionId: string
  answers: Array<{ questionId: string; selected: string[]; text?: string }>
}): Promise<ActionResult> {
  if (!taskPage.pendingAsk) return fail('当前没有待回答的问题')
  taskPage.pendingAsk = { ...taskPage.pendingAsk, status: 'submitting' }
  try {
    await adeApi.answerQuestion(payload.conversationId, payload.questionId, payload.answers)
    taskPage.pendingAsk = { ...taskPage.pendingAsk, status: 'done' }
    setTimeout(() => {
      taskPage.pendingAsk = null
    }, 600)
    return ok('已提交答案', '已提交答案')
  } catch (err) {
    taskPage.pendingAsk = { ...taskPage.pendingAsk, status: 'pending', error: messageOf(err) }
    return fail(messageOf(err))
  }
}

function asPlanTodos(value: unknown): Array<{ title: string; detail?: string; acceptance?: string; dependsOn?: string }> {
  if (!Array.isArray(value)) return []
  return value.flatMap((item) => {
    if (!item || typeof item !== 'object') return []
    const row = item as Record<string, unknown>
    return [
      {
        title: asText(row.title),
        detail: asText(row.detail),
        acceptance: asText(row.acceptance),
        dependsOn: asText(row.dependsOn)
      }
    ]
  })
}

export async function focusTaskTerminal(): Promise<ActionResult> {
  const task = current()
  if (!task) return fail('任务还没加载')
  if (!task.worktree) return fail('这条任务还没有 worktree')
  await useShellStore().openTask(task.id, task.name, task.worktree)
  return ok(`已聚焦终端：${task.worktree}`)
}

export const taskSurface: PageSurface = {
  channel: 'create',
  page: 'task',
  title: '任务计划',
  snapshot() {
    const task = taskPage.task
    if (!task) return { task: null }
    return {
      busy: taskPage.busy,
      pendingAsk: taskPage.pendingAsk
        ? {
            conversationId: taskPage.pendingAsk.conversationId,
            questionId: taskPage.pendingAsk.questionId,
            title: taskPage.pendingAsk.title,
            questions: taskPage.pendingAsk.questions
          }
        : null,
      task: {
        ...briefTask(task),
        rawRequest: task.rawRequest,
        todos: (task.todos ?? []).map(briefTodo)
      }
    }
  },
  actions: [
    {
      name: 'start',
      description: '开始执行当前打开的任务。',
      parameters: params({}),
      run: () => startCurrentTask()
    },
    {
      name: 'retry',
      description: '把当前任务改回重新执行。',
      parameters: params({}),
      run: () => retryCurrentTask()
    },
    {
      name: 'save_plan',
      description: '把整理好的名称、概览、润色需求和待办写回当前已经打开的任务。不要新建任务。',
      parameters: {
        type: 'object',
        additionalProperties: false,
        required: ['name', 'overview', 'plan', 'todos'],
        properties: {
          name: { type: 'string', description: '不超过 20 个汉字' },
          overview: { type: 'string', description: '做完之后是什么样' },
          plan: { type: 'string', description: '润色后的工程需求' },
          todos: {
            type: 'array',
            description: '按依赖排序的待办',
            items: {
              type: 'object',
              additionalProperties: false,
              required: ['title'],
              properties: {
                title: { type: 'string' },
                detail: { type: 'string' },
                acceptance: { type: 'string' },
                dependsOn: { type: 'string' }
              }
            }
          }
        }
      },
      run: (args) =>
        saveTaskPlan({
          name: asText(args.name),
          overview: asText(args.overview),
          plan: asText(args.plan),
          todos: asPlanTodos(args.todos)
        })
    },
    {
      name: 'update_description',
      description: '保存当前任务的需求原文。不会在后台重新拆解，待办保持原样。',
      parameters: params({ description: { type: 'string' } }, ['description']),
      run: (args) => updateTaskDescription(asText(args.description))
    },
    {
      name: 'add_todo',
      description: '给当前任务添加一条待办。',
      parameters: params(
        {
          title: { type: 'string' },
          detail: { type: 'string' },
          acceptance: { type: 'string' },
          dependsOn: { type: 'string' }
        },
        ['title']
      ),
      run: (args) =>
        addTaskTodo({
          title: asText(args.title),
          detail: asText(args.detail),
          acceptance: asText(args.acceptance),
          dependsOn: asText(args.dependsOn)
        })
    },
    {
      name: 'remove_todo',
      description: '删除当前任务的一条待办。',
      parameters: params({ todoId: { type: 'number' } }, ['todoId']),
      run: (args) => removeTaskTodo(asNumber(args.todoId))
    },
    {
      name: 'toggle_todo',
      description: '把待办在完成和未完成之间切换。status 传待办当前状态。',
      parameters: params(
        {
          todoId: { type: 'number' },
          status: { type: 'string', description: '当前状态，done 或其他' }
        },
        ['todoId', 'status']
      ),
      run: (args) => toggleTaskTodo(asNumber(args.todoId), asText(args.status))
    },
    {
      name: 'answer_question',
      description: '回答计划页上正在等待的问题。',
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
        return answerTaskQuestion({
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
      name: 'focus_terminal',
      description: '聚焦当前任务的终端。',
      parameters: params({}),
      run: () => focusTaskTerminal()
    }
  ]
}
