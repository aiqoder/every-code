import { get } from '@/utils/request'

/** 后端健康检查（GET /api/health） */
export function fetchHealth() {
  return get<{ status: string }>('/health')
}
