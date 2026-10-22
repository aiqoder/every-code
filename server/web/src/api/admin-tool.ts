import { del, get, post, put } from '@/utils/request'
import type { PageResult, Tool, ToolPayload } from '@/types/api'

export function fetchAdminTools(params: { page?: number; pageSize?: number } = {}) {
  return get<PageResult<Tool>>('/admin/tools', { params })
}

export function fetchAdminTool(id: number) {
  return get<Tool>(`/admin/tools/${id}`)
}

export function createAdminTool(payload: ToolPayload) {
  return post<Tool>('/admin/tools', payload)
}

export function updateAdminTool(id: number, payload: ToolPayload) {
  return put<Tool>(`/admin/tools/${id}`, payload)
}

export function deleteAdminTool(id: number) {
  return del(`/admin/tools/${id}`)
}
