import { del, get, post, put } from '@/utils/request'

export interface Scene {
  id: number
  userId: number
  title: string
  description: string
  createdAt: string
  updatedAt: string
}

export function fetchAdminScenes() {
  return get<{ list: Scene[] }>('/admin/scenes')
}

export function createAdminScene(payload: { title: string; description: string }) {
  return post<Scene>('/admin/scenes', payload)
}

export function updateAdminScene(id: number, payload: { title: string; description: string }) {
  return put<Scene>(`/admin/scenes/${id}`, payload)
}

export function deleteAdminScene(id: number) {
  return del<{ id: number }>(`/admin/scenes/${id}`)
}
