import request from '@/utils/request'

export interface AuthUser {
  id: number
  username: string
  name: string
  role: string
  roles?: string[]
}

export async function getMe(): Promise<AuthUser> {
  const data = (await request.get('/auth/me')) as { user?: AuthUser } | AuthUser
  if (data && typeof data === 'object' && 'user' in data && data.user) return data.user
  return data as AuthUser
}
