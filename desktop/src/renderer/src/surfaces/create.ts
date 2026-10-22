import { reactive } from 'vue'
import { adeApi, bridge } from '../api'
import { messageOf, useSessionStore } from '../stores/session'
import { toast } from '../composables/toast'
import type { EnvCheckItem, Scene } from '../../../shared/types'
import { asNumber, asText, briefProject, briefTask } from './format'
import { openPath } from './nav'
import { fail, ok, emptyParams, params, type ActionResult, type PageSurface } from './types'

/** 任务页上按钮和智能体共用的状态。检测进度必须在这里，智能体触发时按钮才会转圈。 */
export const createPage = reactive({
  envBusy: false,
  envChatNonce: 0,
  envChatSeen: 0,
  envResult: [] as EnvCheckItem[],
  scenes: [] as Scene[]
})

export async function reloadScenes(): Promise<Scene[]> {
  const data = await adeApi.listScenes()
  createPage.scenes = data.list ?? []
  return createPage.scenes
}

export async function persistScene(input: { id?: number; title: string; description: string }): Promise<Scene> {
  const title = input.title.trim()
  const description = input.description.trim()
  if (!title) throw new Error('请填写场景标题')
  if (!description) throw new Error('请填写场景描述')
  const scene = input.id
    ? await adeApi.updateScene(input.id, { title, description })
    : await adeApi.createScene({ title, description })
  await reloadScenes()
  return scene
}

export async function removeCustomScene(id: number, confirm: boolean): Promise<ActionResult> {
  if (!id) return fail('缺少场景 id')
  if (confirm !== true) return fail('删除场景需要 confirm=true。已创建的项目不受影响。')
  try {
    await adeApi.deleteScene(id)
    createPage.scenes = createPage.scenes.filter((item) => item.id !== id)
    return ok(`已删除场景 #${id}`, '场景已删除', 'plain')
  } catch (err) {
    return fail(messageOf(err))
  }
}

export async function createProject(input: {
  name: string
  description?: string
  localDir: string
  modelId: number
  sceneId?: number
}): Promise<ActionResult> {
  try {
    const session = useSessionStore()
    const name = input.name.trim()
    const description = input.description?.trim() ?? ''
    const localDir = input.localDir.trim()
    const modelId = input.modelId || session.profile.modelId
    const sceneId = input.sceneId ?? 0
    if (!name) return fail('请填写项目名称')
    if (!localDir) return fail('请选择本地目录')
    if (!modelId) return fail('先选择一个模型')
    const status = await bridge.gitStatus(localDir)
    if (!status.isRepo || !status.ready) {
      const init = await bridge.gitInit(localDir)
      if (!init.ok) return fail(`初始化 Git 仓库失败：${init.output}`)
      toast.show(`已初始化仓库：${init.output.split('\n').filter(Boolean).pop() ?? '完成'}`)
    }
    const project = await adeApi.createProject({
      name,
      description,
      localDir,
      scenario: '',
      technologies: [],
      modelId,
      sceneId
    })
    await session.loadProjects()
    session.selectProject(project.id)
    await session.loadTasks(project.id)
    return ok(`已创建项目「${project.name}」#${project.id}`, '项目已创建')
  } catch (err) {
    return fail(messageOf(err))
  }
}

export async function updateProject(input: {
  name: string
  description: string
  localDir?: string
}): Promise<ActionResult> {
  try {
    const session = useSessionStore()
    const project = session.currentProject
    if (!project) return fail('先选择一个项目')
    const name = input.name.trim()
    if (!name) return fail('项目名称不能为空')
    const description = input.description.trim()
    const localDir = input.localDir?.trim() ?? ''
    if (input.localDir !== undefined && !localDir) return fail('请选择本地目录')
    if (localDir && localDir !== project.localDir) {
      const status = await bridge.gitStatus(localDir)
      if (!status.isRepo || !status.ready) {
        const init = await bridge.gitInit(localDir)
        if (!init.ok) return fail(`初始化 Git 仓库失败：${init.output}`)
        toast.show(`已初始化仓库：${init.output.split('\n').filter(Boolean).pop() ?? '完成'}`)
      }
    }
    const updated = await adeApi.updateProject(project.id, {
      name,
      description,
      ...(localDir ? { localDir } : {})
    })
    const idx = session.projects.findIndex((item) => item.id === updated.id)
    if (idx >= 0) session.projects[idx] = updated
    return ok(`已更新项目「${updated.name}」`, '项目已更新')
  } catch (err) {
    return fail(messageOf(err))
  }
}

export async function deleteProject(confirm: boolean): Promise<ActionResult> {
  try {
    const session = useSessionStore()
    const project = session.currentProject
    if (!project) return fail('先选择一个项目')
    if (confirm !== true) return fail('删除项目需要 confirm=true。任务记录会删除，本机目录保留。')
    await adeApi.deleteProject(project.id)
    await session.loadProjects()
    session.currentProjectId = session.projects[0]?.id ?? 0
    return ok(`已删除项目「${project.name}」`, '项目已删除', 'plain')
  } catch (err) {
    return fail(messageOf(err))
  }
}

function asEnvItems(value: unknown): EnvCheckItem[] {
  if (!Array.isArray(value)) return []
  const out: EnvCheckItem[] = []
  for (const item of value) {
    if (!item || typeof item !== 'object') continue
    const row = item as Record<string, unknown>
    const label = asText(row.label)
    if (!label) continue
    out.push({
      tech: asText(row.tech) || label,
      label,
      installed: row.installed === true,
      version: asText(row.version).slice(0, 120),
      command: asText(row.command)
    })
  }
  return out
}

export function parseEnvReport(text: string): EnvCheckItem[] {
  const start = text.lastIndexOf('{')
  const end = text.lastIndexOf('}')
  if (start < 0 || end <= start) return []
  try {
    const parsed = JSON.parse(text.slice(start, end + 1)) as { items?: unknown }
    return asEnvItems(parsed.items)
  } catch {
    return []
  }
}

export async function saveEnvCheck(items: EnvCheckItem[]): Promise<ActionResult> {
  try {
    const session = useSessionStore()
    const project = session.currentProject
    if (!project) return fail('先选择一个项目')
    const cleaned = items.filter((item) => item.label)
    if (!cleaned.length) return fail('没有可保存的检测结果')
    createPage.envResult = cleaned
    const saved = await adeApi.saveEnvCheck(project.id, cleaned)
    const idx = session.projects.findIndex((item) => item.id === saved.id)
    if (idx >= 0) session.projects[idx] = saved
    const missing = cleaned.filter((item) => !item.installed)
    if (missing.length === 0) return ok('本机环境齐备', '本机环境齐备')
    const text = `${missing.map((item) => item.label).join('、')} 未检测到`
    return ok(text, text, 'plain')
  } catch (err) {
    return fail(messageOf(err))
  }
}

export async function saveTaskDescription(description: string): Promise<ActionResult> {
  try {
    const session = useSessionStore()
    const project = session.currentProject
    if (!project) return fail('先选择一个项目')
    const text = description.trim()
    if (!text) return fail('写一句要做什么')
    const task = await adeApi.createTask(project.id, {
      description: text,
      name: '',
      overview: '',
      plan: '',
      todos: []
    })
    session.upsertTask(task)
    await openPath(`/tasks/${task.id}`)
    return ok(`已打开任务 #${task.id}`, '已进入任务')
  } catch (err) {
    return fail(messageOf(err))
  }
}

export async function createTask(input: {
  description: string
  name: string
  overview: string
  plan: string
  todos: Array<{ title: string; detail?: string }>
}): Promise<ActionResult> {
  try {
    const session = useSessionStore()
    const project = session.currentProject
    if (!project) return fail('先选择一个项目')
    const text = input.description.trim()
    const name = input.name.trim()
    const todos = input.todos
      .map((item) => ({
        title: item.title.trim(),
        detail: item.detail?.trim() ?? ''
      }))
      .filter((item) => item.title)
    if (!text) return fail('写一句要做什么')
    if (!name) return fail('请填写任务名称')
    if (!todos.length) return fail('计划至少要有一条待办')
    const task = await adeApi.createTask(project.id, {
      description: text,
      name,
      overview: input.overview.trim(),
      plan: input.plan.trim(),
      todos
    })
    session.upsertTask(task)
    await session.loadProjects()
    return ok(`已创建任务「${task.name}」#${task.id}`, '任务已写入列表')
  } catch (err) {
    return fail(messageOf(err))
  }
}

export async function openTaskPage(taskId: number): Promise<ActionResult> {
  const session = useSessionStore()
  const id = asNumber(taskId)
  const task = (session.tasks[session.currentProjectId] ?? []).find((item) => item.id === id)
  if (!id) return fail('缺少任务 id')
  await openPath(`/tasks/${id}`)
  return ok(task ? `已打开任务「${task.name || id}」的计划页` : `已打开任务 #${id} 的计划页`)
}

function asTodos(value: unknown): Array<{ title: string; detail?: string }> {
  if (!Array.isArray(value)) return []
  return value.flatMap((item) => {
    if (!item || typeof item !== 'object') return []
    const row = item as Record<string, unknown>
    return [
      {
        title: asText(row.title),
        detail: asText(row.detail)
      }
    ]
  })
}

function snapshot(): unknown {
  const session = useSessionStore()
  const project = session.currentProject
  const tasks = session.currentProjectId ? (session.tasks[session.currentProjectId] ?? []) : []
  return {
    project: briefProject(project),
    env: createPage.envResult.length ? createPage.envResult : (project?.envCheck ?? []),
    envBusy: createPage.envBusy,
    scenes: createPage.scenes.map((item) => ({
      id: item.id,
      title: item.title,
      description: item.description,
      custom: item.userId > 0
    })),
    tasks: tasks.map(briefTask),
    projects: session.projects.map((item) => ({ id: item.id, name: item.name }))
  }
}

export const createSurface: PageSurface = {
  channel: 'create',
  page: 'create',
  title: '任务',
  snapshot,
  actions: [
    {
      name: 'create_project',
      description: '按当前页「新建项目」创建项目。目录不是 git 仓库时会先初始化。',
      parameters: params(
        {
          name: { type: 'string', description: '项目名称' },
          localDir: { type: 'string', description: '本机项目目录的绝对路径' },
          sceneId: { type: 'number', description: '场景 id，可省略表示不选场景' },
          modelId: { type: 'number', description: '模型 id，可省略则用当前默认模型' }
        },
        ['name', 'localDir']
      ),
      async run(args) {
        try {
          return await createProject({
            name: asText(args.name),
            localDir: asText(args.localDir),
            modelId: asNumber(args.modelId),
            sceneId: asNumber(args.sceneId)
          })
        } catch (err) {
          return fail(messageOf(err))
        }
      }
    },
    {
      name: 'list_scenes',
      description: '刷新新建项目可选的场景（管理员公共场景和当前用户自定义场景）。',
      parameters: emptyParams(),
      async run() {
        try {
          const list = await reloadScenes()
          return ok(`共 ${list.length} 个场景`, undefined, 'plain')
        } catch (err) {
          return fail(messageOf(err))
        }
      }
    },
    {
      name: 'save_scene',
      description: '保存用户自定义场景。传 id 则修改自己的场景，不传则新建。',
      parameters: params(
        {
          id: { type: 'number', description: '已有自定义场景 id，新建时省略' },
          title: { type: 'string', description: '场景标题' },
          description: { type: 'string', description: '场景描述，可以较长' }
        },
        ['title', 'description']
      ),
      async run(args) {
        try {
          const scene = await persistScene({
            id: asNumber(args.id) || undefined,
            title: asText(args.title),
            description: asText(args.description)
          })
          return ok(`已保存场景「${scene.title}」#${scene.id}`, '场景已保存')
        } catch (err) {
          return fail(messageOf(err))
        }
      }
    },
    {
      name: 'delete_scene',
      description: '删除当前用户自己的场景。必须传 confirm=true。不影响已创建的项目。',
      parameters: params(
        {
          id: { type: 'number' },
          confirm: { type: 'boolean', description: '确认删除' }
        },
        ['id', 'confirm']
      ),
      async run(args) {
        return removeCustomScene(asNumber(args.id), args.confirm === true)
      }
    },
    {
      name: 'update_project',
      description: '保存当前项目名称与详细介绍，等同「编辑项目」。',
      parameters: params(
        {
          name: { type: 'string' },
          description: { type: 'string', description: '项目详细介绍' }
        },
        ['name', 'description']
      ),
      async run(args) {
        try {
          return await updateProject({
            name: asText(args.name),
            description: asText(args.description)
          })
        } catch (err) {
          return fail(messageOf(err))
        }
      }
    },
    {
      name: 'delete_project',
      description: '删除当前项目的任务记录，不删除本机目录。必须传 confirm=true。',
      parameters: params({ confirm: { type: 'boolean', description: '确认删除' } }, ['confirm']),
      async run(args) {
        try {
          return await deleteProject(args.confirm === true)
        } catch (err) {
          return fail(messageOf(err))
        }
      }
    },
    {
      name: 'save_env_check',
      description: '把本机开发环境检查结果写回当前项目。每项含 tech、label、installed、version、command。',
      parameters: {
        type: 'object',
        additionalProperties: false,
        required: ['items'],
        properties: {
          items: {
            type: 'array',
            items: {
              type: 'object',
              additionalProperties: false,
              required: ['label', 'installed'],
              properties: {
                tech: { type: 'string' },
                label: { type: 'string' },
                installed: { type: 'boolean' },
                version: { type: 'string' },
                command: { type: 'string' }
              }
            }
          }
        }
      },
      run: (args) => saveEnvCheck(asEnvItems(args.items))
    },
    {
      name: 'create_task',
      description:
        '把对话里定好的计划写入当前项目。先和用户确认需求，再一次性写入名称、概览、润色需求和待办。不会自动执行。',
      parameters: {
        type: 'object',
        additionalProperties: false,
        required: ['description', 'name', 'overview', 'plan', 'todos'],
        properties: {
          description: { type: 'string', description: '用户原始需求' },
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
                detail: { type: 'string' }
              }
            }
          }
        }
      },
      async run(args) {
        try {
          return await createTask({
            description: asText(args.description),
            name: asText(args.name),
            overview: asText(args.overview),
            plan: asText(args.plan),
            todos: asTodos(args.todos)
          })
        } catch (err) {
          return fail(messageOf(err))
        }
      }
    },
    {
      name: 'open_task',
      description: '打开当前项目里某条任务的计划页。',
      parameters: params({ taskId: { type: 'number' } }, ['taskId']),
      async run(args) {
        try {
          return await openTaskPage(asNumber(args.taskId))
        } catch (err) {
          return fail(messageOf(err))
        }
      }
    }
  ]
}
