import { syncPackageAgent } from '@/views/chat/syncPackage'

/** 把本机插件同步到服务器。已绑定的也会更新，以便补上后加的工具。 */
export async function syncInstalledPackages(): Promise<{ changed: boolean; error: string }> {
  if (!window.api?.loop?.listAgentHome) return { changed: false, error: '' }
  const home = await window.api.loop.listAgentHome()
  const packages = Array.isArray(home.packages) ? home.packages : []
  let changed = false
  let error = ''
  for (const pkg of packages) {
    try {
      const remoteId = await syncPackageAgent(pkg)
      changed = true
      if (remoteId !== pkg.remoteId && window.api.loop.setPackageRemoteId) {
        await window.api.loop.setPackageRemoteId(pkg.id, remoteId)
      }
    } catch (err) {
      error = err instanceof Error ? err.message : '同步智能体失败'
    }
  }
  return { changed, error }
}
