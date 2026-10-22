import { Client } from 'ssh2'
import fs from 'node:fs'
import os from 'node:os'
import { createHash, randomUUID } from 'node:crypto'
import { BrowserWindow } from 'electron'
import { IPC } from '../shared/ipc'
import { getPrivateKeyMaterial } from './store'

interface SshSession {
  id: string
  taskId?: number
  hostId?: number
  client: Client
  stream?: any
}

const sessions = new Map<string, SshSession>()

function broadcast(channel: string, payload: unknown): void {
  for (const win of BrowserWindow.getAllWindows()) {
    if (!win.isDestroyed()) win.webContents.send(channel, payload)
  }
}

export interface OpenSshOptions {
  id?: string
  hostId: number
  host: string
  port: number
  username: string
  fingerprint?: string
  taskId?: number
  cols?: number
  rows?: number
  privateKeyPath?: string
}

/** 打开一条 SSH 会话，接 xterm。首次连接返回指纹，让用户核对。 */
export function openSsh(opts: OpenSshOptions): Promise<{ id: string; fingerprint: string }> {
  const id = opts.id ?? randomUUID()
  let privateKey = ''
  if (opts.privateKeyPath && fs.existsSync(opts.privateKeyPath)) {
    privateKey = fs.readFileSync(opts.privateKeyPath, 'utf8')
  } else {
    privateKey = getPrivateKeyMaterial(opts.hostId)
  }
  return new Promise((resolve, reject) => {
    if (!privateKey.trim()) {
      reject(new Error('本机没有这张主机的私钥，请先选择私钥文件'))
      return
    }
    const client = new Client()
    let fingerprint = ''
    let settled = false

    const fail = (err: Error): void => {
      if (settled) return
      settled = true
      try {
        client.end()
      } catch {
        /* ignore */
      }
      sessions.delete(id)
      reject(err)
    }

    client.on('ready', () => {
      client.shell({ term: 'xterm-256color', cols: opts.cols ?? 120, rows: opts.rows ?? 30 }, (err, stream) => {
        if (err) {
          fail(err)
          return
        }
        const session: SshSession = { id, taskId: opts.taskId, hostId: opts.hostId, client, stream }
        sessions.set(id, session)
        stream.on('data', (data: Buffer) => {
          broadcast(IPC.ptyData, { id, data: data.toString('utf8') })
        })
        stream.stderr.on('data', (data: Buffer) => {
          broadcast(IPC.ptyData, { id, data: data.toString('utf8') })
        })
        stream.on('close', () => {
          broadcast(IPC.ptyExit, { id, exitCode: 0 })
          sessions.delete(id)
          client.end()
        })
        settled = true
        resolve({ id, fingerprint })
      })
    })

    client.on('error', (err) => fail(err))

    client
      .on('close', () => {
        if (!settled) fail(new Error('SSH 连接被关闭'))
      })
      .connect({
        host: opts.host,
        port: opts.port || 22,
        username: opts.username,
        privateKey,
        readyTimeout: 20_000,
        hostVerifier: (key: Buffer) => {
          fingerprint = fingerprintOf(key)
          // 服务端已有指纹则以它为准，否则接受并回报给用户确认。
          if (opts.fingerprint && opts.fingerprint.trim()) {
            return opts.fingerprint.trim() === fingerprint
          }
          return true
        }
      })
  })
}

/** SHA256 指纹，与 openssh 的 SHA256: 形式一致（去掉尾部 =）。 */
function fingerprintOf(key: Buffer): string {
  const hash = createHash('sha256').update(key).digest('base64').replace(/=+$/, '')
  return `SHA256:${hash}`
}

export function writeSsh(id: string, data: string): void {
  const s = sessions.get(id)
  if (!s?.stream) return
  s.stream.write(data)
}

export function resizeSsh(id: string, cols: number, rows: number): void {
  const s = sessions.get(id)
  if (!s?.stream) return
  try {
    s.stream.setWindow(Math.max(20, rows), Math.max(20, cols), 0, 0)
  } catch {
    /* ignore */
  }
}

export function closeSsh(id: string): void {
  const s = sessions.get(id)
  if (!s) return
  try {
    s.stream?.end()
    s.client.end()
  } catch {
    /* ignore */
  }
  sessions.delete(id)
}

/** 在 SSH 会话里跑一条非交互命令，返回退出码与输出（用于发布剧本与审计）。 */
export function execSsh(
  id: string,
  command: string
): Promise<{ code: number; output: string }> {
  const s = sessions.get(id)
  if (!s) return Promise.reject(new Error('SSH 会话不存在'))
  return new Promise((resolve, reject) => {
    s.client.exec(command, (err, stream) => {
      if (err) {
        reject(err)
        return
      }
      let out = ''
      stream
        .on('data', (data: Buffer) => {
          out += data.toString('utf8')
        })
        .on('close', (code: number) => {
          resolve({ code: code ?? 0, output: out.slice(0, 20_000) })
        })
        .on('error', reject)
    })
  })
}

/** 用 SFTP 把本地文件或目录传到远端。远端目录不存在会自动建。 */
export function uploadSsh(
  id: string,
  localPath: string,
  remotePath: string
): Promise<{ files: number; bytes: number }> {
  const s = sessions.get(id)
  if (!s) return Promise.reject(new Error('SSH 会话不存在'))
  if (!fs.existsSync(localPath)) return Promise.reject(new Error(`本地路径不存在：${localPath}`))

  return new Promise((resolve, reject) => {
    s.client.sftp((err, sftp) => {
      if (err) {
        reject(err)
        return
      }
      let files = 0
      let bytes = 0

      const mkdirp = (dir: string): Promise<void> =>
        new Promise((done) => {
          if (!dir || dir === '/' || dir === '.') {
            done()
            return
          }
          sftp.stat(dir, (statErr) => {
            if (!statErr) {
              done()
              return
            }
            mkdirp(posixDirname(dir)).then(() => {
              sftp.mkdir(dir, () => done())
            })
          })
        })

      const putFile = (local: string, remote: string): Promise<void> =>
        new Promise((done, fail) => {
          sftp.fastPut(local, remote, (e: Error | null | undefined) => {
            if (e) {
              fail(e)
              return
            }
            files += 1
            try {
              bytes += fs.statSync(local).size
            } catch {
              /* ignore */
            }
            done()
          })
        })

      const walk = async (local: string, remote: string): Promise<void> => {
        const st = fs.statSync(local)
        if (st.isDirectory()) {
          await mkdirp(remote)
          for (const entry of fs.readdirSync(local, { withFileTypes: true })) {
            if (entry.name === '.git' || entry.name === 'node_modules') continue
            await walk(pathJoin(local, entry.name), `${remote}/${entry.name}`)
          }
          return
        }
        await mkdirp(posixDirname(remote))
        await putFile(local, remote)
      }

      walk(localPath, remotePath)
        .then(() => sftp.end())
        .then(() => resolve({ files, bytes }))
        .catch((e) => {
          try {
            sftp.end()
          } catch {
            /* ignore */
          }
          reject(e)
        })
    })
  })
}

function pathJoin(a: string, b: string): string {
  return a.endsWith('/') ? `${a}${b}` : `${a}/${b}`
}

function posixDirname(p: string): string {
  const idx = p.lastIndexOf('/')
  if (idx <= 0) return '/'
  return p.slice(0, idx)
}
