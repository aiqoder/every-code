import { del, get, post, put } from '@/utils/request'
import type { LlmModel, LlmModelPayload, PageResult } from '@/types/api'

export function fetchLlmModels(
  params: { page?: number; pageSize?: number; providerId?: number } = {},
) {
  return get<PageResult<LlmModel>>('/llm-models', { params })
}

export function fetchLlmModel(id: number) {
  return get<LlmModel>(`/llm-models/${id}`)
}

export function createLlmModel(payload: LlmModelPayload) {
  return post<LlmModel>('/llm-models', payload)
}

export function updateLlmModel(id: number, payload: LlmModelPayload) {
  return put<LlmModel>(`/llm-models/${id}`, payload)
}

export function deleteLlmModel(id: number) {
  return del(`/llm-models/${id}`)
}

export function fetchInputCapabilities() {
  return get<{ list: string[] }>('/llm-models/input-capabilities')
}
