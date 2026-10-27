import request from '@/utils/request'

export interface HealthResult {
  code: number
  msg: string
  data: unknown
  sysMsg?: string
}

export function getHealth(): Promise<HealthResult> {
  return request.get('/health') as Promise<HealthResult>
}
