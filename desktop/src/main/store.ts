import { app, safeStorage } from 'electron'
import fs from 'node:fs'
import path from 'node:path'
import type { ServerProfile } from '../shared/types'

interface StoredSettings {
  profile: ServerProfile
  /** 加密后的 token（base64）。未登录为空。 */
  tokenCipher: string
  /**
   * 本机 SSH 私钥，按主机档案 id 索引。
   * 新格式：`cipher:<base64(safeStorage)>` 存密钥正文；
   * 旧格式：绝对路径（读取时升级为 cipher）。
   */
  privateKeys: Record<string, string>
  /** 仅用于界面展示「已选哪把钥匙」，可为空。 */
  privateKeyLabels: Record<string, string>
  lastProjectId: number
  lastTaskId: number
}

const defaultProfile: ServerProfile = {
  baseUrl: 'http://127.0.0.1:8183',
  username: '',
  clientId: '',
  clientName: 'ADE 桌面',
  workspace: '',
  modelId: 0
}

let cached: StoredSettings | null = null
let tokenCache = ''

function dataDir(): string {
  const dir = path.join(app.getPath('userData'), 'ade')
  fs.mkdirSync(dir, { recursive: true })
  return dir
}

function settingsPath(): string {
  return path.join(dataDir(), 'settings.json')
}

export function skillsDir(): string {
  const dir = path.join(app.getPath('userData'), 'skills')
  fs.mkdirSync(dir, { recursive: true })
  return dir
}

function ensureClientId(profile: ServerProfile): ServerProfile {
  if (profile.clientId) return profile
  const host = require('node:os').hostname().replace(/[^a-zA-Z0-9-]/g, '-').toLowerCase()
  const suffix = Math.random().toString(36).slice(2, 6)
  profile.clientId = `ade-${host}-${suffix}`
  return profile
}

export function loadSettings(): StoredSettings {
  if (cached) return cached
  let raw: Partial<StoredSettings> = {}
  try {
    raw = JSON.parse(fs.readFileSync(settingsPath(), 'utf8'))
  } catch {
    raw = {}
  }
  const profile = ensureClientId({ ...defaultProfile, ...(raw.profile ?? {}) })
  const previous = profile.workspace
  if (!profile.workspace) {
    profile.workspace = path.join(app.getPath('home'), '.every-code')
  }
  cached = {
    profile,
    tokenCipher: raw.tokenCipher ?? '',
    privateKeys: raw.privateKeys ?? {},
    privateKeyLabels: raw.privateKeyLabels ?? {}, 
    lastProjectId: raw.lastProjectId ?? 0,
    lastTaskId: raw.lastTaskId ?? 0
  }
  if (profile.workspace !== previous) persist()
  return cached
}

function persist(): void {
  if (!cached) return
  fs.mkdirSync(dataDir(), { recursive: true })
  fs.writeFileSync(settingsPath(), JSON.stringify(cached, null, 2), 'utf8')
}

export function saveProfile(patch: Partial<ServerProfile>): ServerProfile {
  const s = loadSettings()
  s.profile = ensureClientId({ ...s.profile, ...patch })
  persist()
  return s.profile
}

export function getProfile(): ServerProfile {
  return loadSettings().profile
}

export function setToken(token: string): void {
  const s = loadSettings()
  tokenCache = token
  if (!token) {
    s.tokenCipher = ''
    persist()
    return
  }
  try {
    s.tokenCipher = safeStorage.isEncryptionAvailable()
      ? safeStorage.encryptString(token).toString('base64')
      : `plain:${Buffer.from(token, 'utf8').toString('base64')}`
  } catch {
    s.tokenCipher = `plain:${Buffer.from(token, 'utf8').toString('base64')}`
  }
  persist()
}

export function getToken(): string {
  if (tokenCache) return tokenCache
  const s = loadSettings()
  if (!s.tokenCipher) return ''
  try {
    if (s.tokenCipher.startsWith('plain:')) {
      tokenCache = Buffer.from(s.tokenCipher.slice(6), 'base64').toString('utf8')
      return tokenCache
    }
    tokenCache = safeStorage.decryptString(Buffer.from(s.tokenCipher, 'base64'))
    return tokenCache
  } catch {
    return ''
  }
}

export function clearToken(): void {
  tokenCache = ''
  setToken('')
}

export function setLastIds(projectId: number, taskId: number): void {
  const s = loadSettings()
  if (projectId) s.lastProjectId = projectId
  if (taskId) s.lastTaskId = taskId
  persist()
}

function encryptSecret(plain: string): string {
  try {
    if (safeStorage.isEncryptionAvailable()) {
      return `cipher:${safeStorage.encryptString(plain).toString('base64')}`
    }
  } catch {
    /* fall through */
  }
  return `plain:${Buffer.from(plain, 'utf8').toString('base64')}`
}

function decryptSecret(stored: string): string {
  if (stored.startsWith('cipher:')) {
    return safeStorage.decryptString(Buffer.from(stored.slice(7), 'base64'))
  }
  if (stored.startsWith('plain:')) {
    return Buffer.from(stored.slice(6), 'base64').toString('utf8')
  }
  return ''
}

/** 取该主机私钥正文（供 ssh2 使用）。兼容旧路径存储并就地升级。 */
export function getPrivateKeyMaterial(hostId: number): string {
  const s = loadSettings()
  const key = String(hostId)
  const stored = s.privateKeys[key] ?? ''
  if (!stored) return ''
  if (stored.startsWith('cipher:') || stored.startsWith('plain:')) {
    try {
      return decryptSecret(stored)
    } catch {
      return ''
    }
  }
  // 旧版：存的是文件路径
  if (stored.startsWith('/') || /^[A-Za-z]:[\\/]/.test(stored)) {
    try {
      if (!fs.existsSync(stored)) return ''
      const body = fs.readFileSync(stored, 'utf8')
      s.privateKeys[key] = encryptSecret(body)
      s.privateKeyLabels[key] = stored
      persist()
      return body
    } catch {
      return ''
    }
  }
  return ''
}

/** 界面展示用：是否已绑定私钥，以及原始文件路径（若有）。 */
export function getPrivateKeyLabel(hostId: number): string {
  const s = loadSettings()
  const key = String(hostId)
  if (s.privateKeyLabels[key]) return s.privateKeyLabels[key]
  const stored = s.privateKeys[key] ?? ''
  if (!stored) return ''
  if (stored.startsWith('/') || /^[A-Za-z]:[\\/]/.test(stored)) return stored
  return '已加密保存在本机'
}

/** @deprecated 兼容旧调用：有密钥则返回标签，否则空。 */
export function getPrivateKeyPath(hostId: number): string {
  return getPrivateKeyLabel(hostId)
}

/** 从本机文件读取私钥，用 safeStorage 加密后按 hostId 保存。 */
export function setPrivateKeyPath(hostId: number, keyPath: string): void {
  const s = loadSettings()
  const key = String(hostId)
  if (!keyPath) {
    delete s.privateKeys[key]
    delete s.privateKeyLabels[key]
    persist()
    return
  }
  const body = fs.readFileSync(keyPath, 'utf8')
  if (!body.trim()) throw new Error('私钥文件是空的')
  s.privateKeys[key] = encryptSecret(body)
  s.privateKeyLabels[key] = keyPath
  persist()
}
