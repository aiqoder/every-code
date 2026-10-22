import axios, { type AxiosError, type AxiosInstance, type AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'

import type { ApiEnvelope } from '@/types/api'

const TOKEN_KEY = 'chengzi_token'

const request: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 15000,
})

request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem(TOKEN_KEY)
    if (token) {
      config.headers = config.headers ?? {}
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error),
)

request.interceptors.response.use(
  (response) => {
    const envelope = response.data as ApiEnvelope
    if (envelope && typeof envelope.code === 'number') {
      if (envelope.code === 0) {
        return envelope.data as never
      }
      ElMessage.error(envelope.msg || '请求失败')
      if (envelope.sysMsg) {
        console.error('[api]', envelope.sysMsg)
      }
      return Promise.reject(new Error(envelope.msg || envelope.sysMsg || '请求失败'))
    }
    return response.data
  },
  (error: AxiosError<ApiEnvelope>) => {
    const msg = error.response?.data?.msg || error.message || '网络异常'
    const sysMsg = error.response?.data?.sysMsg
    ElMessage.error(msg)
    if (sysMsg) {
      console.error('[api]', sysMsg)
    }
    return Promise.reject(error)
  },
)

/** 类型安全的请求封装：拦截器已解包 data */
export function get<T>(url: string, config?: AxiosRequestConfig) {
  return request.get<unknown, T>(url, config)
}

export function post<T>(url: string, data?: unknown, config?: AxiosRequestConfig) {
  return request.post<unknown, T>(url, data, config)
}

export function put<T>(url: string, data?: unknown, config?: AxiosRequestConfig) {
  return request.put<unknown, T>(url, data, config)
}

export function del<T>(url: string, config?: AxiosRequestConfig) {
  return request.delete<unknown, T>(url, config)
}

export default request
