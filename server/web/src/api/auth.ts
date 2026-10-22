import { get, post } from '@/utils/request'
import type { AuthResult, OAuthAuthorizeResult, OAuthCallbackResult, User } from '@/types/api'

/** 通过 user-feedback 校验账号密码登录 */
export function login(payload: { username: string; password: string }) {
  return post<AuthResult>('/auth/login', payload)
}

/** 获取当前登录用户信息 */
export function fetchMe() {
  return get<{ user: User }>('/auth/me').then((data) => data.user)
}

/** 获取 user-feedback 的 OAuth2 授权地址 */
export function oauthAuthorize(state?: string) {
  return get<OAuthAuthorizeResult>('/auth/oauth/authorize', { params: { state } })
}

/** 处理 OAuth2 授权码回调 */
export function oauthCallback(code: string) {
  return get<OAuthCallbackResult>('/auth/oauth/callback', { params: { code } })
}
