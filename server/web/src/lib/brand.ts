/** 网页品牌接口前缀，与 axios 的 baseURL 一致。 */
export function webBrandApiBase(): string {
  return String(import.meta.env.VITE_API_BASE_URL || '/api').replace(/\/$/, '')
}
