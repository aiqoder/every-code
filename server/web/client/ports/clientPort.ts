import { inject, provide, type InjectionKey } from 'vue'
import type {
  Agent,
  LlmProvider,
  Model,
  ModelPayload,
  PackageInfo,
  ProviderPayload,
  ConnectorInput,
  ConnectorView,
  SkillItem,
  TokenUsageStats,
} from '../types'

export interface LinkSnapshot {
  clientId: string
  clientName: string
  connected: boolean
  loggedIn: boolean
  wsStatus: string
  busy: boolean
  notice: string
  lastError: string
  clientIdReadonly: boolean
}

export interface CpkDrop {
  path?: string
  error?: string
}

/** 桌面与网页共用页面的数据入口。界面组件只依赖这份接口。 */
export interface ClientPort {
  linkState(): LinkSnapshot
  saveLink(clientId: string, clientName: string): Promise<void>
  subscribeCpkDrop?(handler: (drop: CpkDrop) => void, event?: string): () => void

  listPackages(): Promise<PackageInfo[]>
  packageRoot(): Promise<string>
  openPackageRoot(): Promise<void>
  installPackage(fileName: string, base64: string): Promise<void>
  installPackageFromPath(path: string): Promise<void>
  uninstallPackage(id: string): Promise<void>
  savePackageEnv(id: string, values: Record<string, string>): Promise<void>
  clearPackageEnv(id: string): Promise<void>
  listAgents(): Promise<Agent[]>

  listConnectors(): Promise<ConnectorView[]>
  saveConnector(input: ConnectorInput): Promise<ConnectorView[]>
  deleteConnector(id: string): Promise<ConnectorView[]>
  setConnectorEnabled(id: string, enabled: boolean): Promise<ConnectorView[]>
  probeConnector(input: ConnectorInput): Promise<ConnectorView>

  listSkills(): Promise<SkillItem[]>
  deleteSkill(id: string): Promise<SkillItem[]>
  saveSkill(slug: string, content: string): Promise<void>

  listProviders(): Promise<LlmProvider[]>
  listModels(): Promise<Model[]>
  deleteProvider(id: number): Promise<void>
  deleteModel(id: number): Promise<void>
  listRemoteModels(providerId: number): Promise<string[]>
  createProvider(payload: ProviderPayload): Promise<void>
  updateProvider(id: number, payload: ProviderPayload): Promise<void>
  createModel(payload: ModelPayload): Promise<void>
  updateModel(id: number, payload: ModelPayload): Promise<void>
  listInputCapabilities(): Promise<string[]>
  getTokenUsage(range: string): Promise<TokenUsageStats | null>
}

export const clientPortKey: InjectionKey<ClientPort> = Symbol('clientPort')

export function provideClientPort(port: ClientPort) {
  provide(clientPortKey, port)
}

export function useClientPort(): ClientPort {
  const port = inject(clientPortKey)
  if (!port) throw new Error('未注入客户端端口')
  return port
}
