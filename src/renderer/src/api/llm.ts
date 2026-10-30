import request from '@/utils/request'

export interface LlmProvider {
  id: number
  name: string
  website: string
  apiKey: string
  openaiCompatibleBaseUrl: string
  anthropicBaseUrl: string
  editable: boolean
}

export interface ProviderPayload {
  name: string
  website: string
  apiKey: string
  openaiCompatibleBaseUrl: string
  anthropicBaseUrl: string
}

export interface LlmModel {
  id: number
  providerId: number
  providerName: string
  name: string
  attachment: boolean
  reasoning: boolean
  reasoningOptions: string[]
  temperature: boolean
  inputCapabilities: string[]
  inputContextLimit: number
  outputContextLimit: number
  inputPrice: number
  outputPrice: number
  cachePrice: number
  editable: boolean
}

export interface ModelPayload {
  providerId: number
  name: string
  attachment: boolean
  reasoning: boolean
  reasoningOptions: string[]
  temperature: boolean
  inputCapabilities: string[]
  inputContextLimit: number
  outputContextLimit: number
  inputPrice: number
  outputPrice: number
  cachePrice: number
}

const pageSize = 100

export function listLlmProviders(): Promise<LlmProvider[]> {
  return request.get('/llm-providers', { params: { all: 1 } }).then((data) => readProviders(data))
}

export function createLlmProvider(payload: ProviderPayload): Promise<LlmProvider> {
  return request.post('/llm-providers', payload).then((data) => requireProvider(data))
}

export function updateLlmProvider(id: number, payload: ProviderPayload): Promise<LlmProvider> {
  return request.put(`/llm-providers/${id}`, payload).then((data) => requireProvider(data))
}

export function deleteLlmProvider(id: number): Promise<void> {
  return request.delete(`/llm-providers/${id}`).then(() => undefined)
}

export function listRemoteModels(providerId: number): Promise<string[]> {
  return request
    .get(`/llm-providers/${providerId}/remote-models`, { timeout: 25000 })
    .then((data) => readStrings(readRecord(data)?.list))
}

export async function listAllLlmModels(): Promise<LlmModel[]> {
  const all: LlmModel[] = []
  let page = 1
  let total = Number.POSITIVE_INFINITY
  while (all.length < total && page <= 50) {
    const data = await request.get('/llm-models', { params: { page, pageSize } })
    const body = readRecord(data)
    const batch = readModels(body?.list)
    const reported = body?.total
    total = typeof reported === 'number' && Number.isFinite(reported) ? reported : all.length + batch.length
    all.push(...batch)
    if (batch.length < pageSize) break
    page += 1
  }
  return all
}

export function createLlmModel(payload: ModelPayload): Promise<LlmModel> {
  return request.post('/llm-models', payload).then((data) => requireModel(data))
}

export function updateLlmModel(id: number, payload: ModelPayload): Promise<LlmModel> {
  return request.put(`/llm-models/${id}`, payload).then((data) => requireModel(data))
}

export function deleteLlmModel(id: number): Promise<void> {
  return request.delete(`/llm-models/${id}`).then(() => undefined)
}

export function listInputCapabilities(): Promise<string[]> {
  return request.get('/llm-models/input-capabilities').then((data) => {
    const list = readStrings(readRecord(data)?.list)
    return list.length ? list : ['text', 'image', 'pdf']
  })
}

function readProviders(data: unknown): LlmProvider[] {
  const list = Array.isArray(data) ? data : readRecord(data)?.list
  if (!Array.isArray(list)) return []
  return list.flatMap((item) => {
    const provider = readProvider(item)
    return provider ? [provider] : []
  })
}

function requireProvider(data: unknown): LlmProvider {
  const provider = readProvider(data)
  if (!provider) throw new Error('供应商数据无效')
  return provider
}

function readProvider(data: unknown): LlmProvider | null {
  const row = readRecord(data)
  const id = num(row?.id)
  if (!row || !id) return null
  return {
    id,
    name: text(row.name),
    website: text(row.website),
    apiKey: text(row.apiKey),
    openaiCompatibleBaseUrl: text(row.openaiCompatibleBaseUrl),
    anthropicBaseUrl: text(row.anthropicBaseUrl),
    editable: row.editable !== false
  }
}

function readModels(data: unknown): LlmModel[] {
  if (!Array.isArray(data)) return []
  return data.flatMap((item) => {
    const model = readModel(item)
    return model ? [model] : []
  })
}

function requireModel(data: unknown): LlmModel {
  const model = readModel(data)
  if (!model) throw new Error('模型数据无效')
  return model
}

function readModel(data: unknown): LlmModel | null {
  const row = readRecord(data)
  const id = num(row?.id)
  if (!row || !id) return null
  return {
    id,
    providerId: num(row.providerId),
    providerName: text(row.providerName),
    name: text(row.name),
    attachment: row.attachment === true,
    reasoning: row.reasoning === true,
    reasoningOptions: readStrings(row.reasoningOptions),
    temperature: row.temperature === true,
    inputCapabilities: readStrings(row.inputCapabilities),
    inputContextLimit: num(row.inputContextLimit, 200000),
    outputContextLimit: num(row.outputContextLimit),
    inputPrice: num(row.inputPrice),
    outputPrice: num(row.outputPrice),
    cachePrice: num(row.cachePrice),
    editable: row.editable !== false
  }
}

function readRecord(data: unknown): Record<string, unknown> | null {
  if (!data || typeof data !== 'object' || Array.isArray(data)) return null
  return data as Record<string, unknown>
}

function readStrings(data: unknown): string[] {
  if (!Array.isArray(data)) return []
  return data.filter((item): item is string => typeof item === 'string' && item.trim().length > 0)
}

function text(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function num(value: unknown, fallback = 0): number {
  const parsed = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}
