import { get, post, put } from '@/utils/request'
import type { LlmModel, LlmModelPayload, PageResult } from '@/types/api'

export function fetchAdminLlmModels(
  params: { page?: number; pageSize?: number; providerId?: number } = {},
) {
  return get<PageResult<LlmModel>>('/admin/llm-models', { params })
}

export function createAdminLlmModel(payload: LlmModelPayload) {
  return post<LlmModel>('/admin/llm-models', payload)
}

export function updateAdminLlmModel(id: number, payload: LlmModelPayload) {
  return put<LlmModel>(`/admin/llm-models/${id}`, payload)
}
