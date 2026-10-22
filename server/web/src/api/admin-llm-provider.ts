import { get, post, put } from '@/utils/request'
import type { LlmProvider, LlmProviderPayload, PageResult } from '@/types/api'

export function fetchAdminLlmProviders(params: { page?: number; pageSize?: number } = {}) {
  return get<PageResult<LlmProvider>>('/admin/llm-providers', { params })
}

export function fetchAllAdminLlmProviders() {
  return get<{ list: LlmProvider[] }>('/admin/llm-providers', { params: { all: 1 } })
}

export function fetchAdminLlmProvider(id: number) {
  return get<LlmProvider>(`/admin/llm-providers/${id}`)
}

export function createAdminLlmProvider(payload: LlmProviderPayload) {
  return post<LlmProvider>('/admin/llm-providers', payload)
}

export function updateAdminLlmProvider(id: number, payload: LlmProviderPayload) {
  return put<LlmProvider>(`/admin/llm-providers/${id}`, payload)
}

export function fetchAdminProviderRemoteModels(providerId: number) {
  return get<{ list: string[] }>(`/admin/llm-providers/${providerId}/remote-models`, {
    timeout: 25000,
  })
}
