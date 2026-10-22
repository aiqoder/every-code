export interface Brand {
  name: string
  logo: string
}

export const DEFAULT_BRAND: Brand = { name: '橙子', logo: '🍊' }

/** logo 为图片地址时用 img 渲染，否则按文本或 emoji 渲染。 */
export function isLogoImage(logo: string): boolean {
  const value = logo.trim()
  return /^https?:\/\//i.test(value) || value.startsWith('/')
}

/** 把服务器根地址收成品牌接口前缀；地址无效时返回空串，调用方使用默认品牌。 */
export function brandApiBase(serverURL: string): string {
  const raw = serverURL.trim()
  if (!raw) return ''
  try {
    const url = new URL(raw)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return ''
    if (!url.host) return ''
    return `${url.origin}/api`
  } catch {
    return ''
  }
}

function normalizeBrand(raw: Partial<Brand> | null | undefined): Brand {
  const name = raw?.name?.trim() || DEFAULT_BRAND.name
  const logo = raw?.logo?.trim() || DEFAULT_BRAND.logo
  return { name, logo }
}

const cache = new Map<string, Brand>()
const inflight = new Map<string, Promise<Brand>>()

/** 读取品牌。同一地址只请求一次；失败时回退默认值且不写入缓存。 */
export function loadBrand(apiBase: string): Promise<Brand> {
  const base = apiBase.trim().replace(/\/$/, '')
  if (!base) return Promise.resolve({ ...DEFAULT_BRAND })
  const hit = cache.get(base)
  if (hit) return Promise.resolve(hit)
  const pending = inflight.get(base)
  if (pending) return pending
  const task = fetchBrand(base)
    .then((brand) => {
      cache.set(base, brand)
      return brand
    })
    .finally(() => {
      inflight.delete(base)
    })
  inflight.set(base, task)
  return task
}

async function fetchBrand(apiBase: string): Promise<Brand> {
  const res = await fetch(`${apiBase}/brand`, {
    headers: { Accept: 'application/json' },
    signal: AbortSignal.timeout(8000),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const body = (await res.json()) as { code?: number; data?: Partial<Brand> }
  if (body.code !== 0) throw new Error('品牌配置无效')
  return normalizeBrand(body.data)
}
