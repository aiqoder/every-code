/** 还没有名称时，用创建时间当作默认标题。 */
export function taskDisplayName(task: { name?: string; createdAt?: string }): string {
  const name = task.name?.trim()
  if (name) return name
  const at = task.createdAt?.trim()
  return at ? `新任务-${at}` : '新任务'
}
