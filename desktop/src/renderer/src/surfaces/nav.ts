/** 延迟取出路由，避免 surface 和路由模块互相引用。 */
export async function openPath(path: string): Promise<void> {
  const { router } = await import('../router')
  await router.push(path)
}
