import type { Project, Task, Todo } from '../../../shared/types'

export function briefProject(project: Project | null | undefined): Record<string, unknown> | null {
  if (!project) return null
  return {
    id: project.id,
    name: project.name,
    description: project.description,
    localDir: project.localDir,
    scenario: project.scenario,
    technologies: project.technologies,
    modelId: project.modelId,
    sceneId: project.sceneId,
    envReady: project.envReady,
    envCheck: project.envCheck ?? []
  }
}

export function briefTask(task: Task): Record<string, unknown> {
  return {
    id: task.id,
    name: task.name,
    projectId: task.projectId,
    projectName: task.projectName,
    doneCount: task.doneCount,
    totalCount: task.totalCount,
    overview: task.overview,
    description: task.description
  }
}

export function briefTodo(todo: Todo): Record<string, unknown> {
  return {
    id: todo.id,
    title: todo.title,
    detail: todo.detail,
    status: todo.status
  }
}

export function asStrings(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.map((item) => String(item).trim()).filter(Boolean)
}

export function asNumber(value: unknown): number {
  const n = Number(value)
  return Number.isFinite(n) ? n : 0
}

export function asText(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}
