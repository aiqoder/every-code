import request from '@/utils/request'

export function getHealth(): Promise<{ status: string }> {
  return request.get('/health') as unknown as Promise<{ status: string }>
}
