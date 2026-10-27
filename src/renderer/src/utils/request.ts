import axios from 'axios'
import { readApiBase, readApiKey } from '@/lib/sessionConfig'

const request = axios.create({
  timeout: 15000
})

request.interceptors.request.use(
  (config) => {
    config.baseURL = readApiBase()
    const apiKey = readApiKey()
    if (apiKey) {
      config.params = { ...config.params, api_key: apiKey }
    }
    return config
  },
  (error) => Promise.reject(error)
)

request.interceptors.response.use(
  (response) => {
    const body = response.data
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code !== 0) {
        const msg = body.msg || body.sysMsg || '请求失败'
        const detail = body.sysMsg && body.sysMsg !== msg ? `${msg}: ${body.sysMsg}` : msg
        return Promise.reject(new Error(detail))
      }
      return body.data
    }
    return body
  },
  (error) => {
    const msg = error?.response?.data?.msg || error?.message || '网络错误'
    return Promise.reject(new Error(msg))
  }
)

export default request
