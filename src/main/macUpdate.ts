import { execFile, spawn } from 'child_process'
import { app } from 'electron'
import { constants, readFileSync } from 'fs'
import { access, mkdtemp, readdir, rm, stat, writeFile } from 'fs/promises'
import { homedir, tmpdir } from 'os'
import { join, resolve, sep } from 'path'
import { promisify } from 'util'

const execFileAsync = promisify(execFile)

let installStarted = false

export function macInstallInProgress(): boolean {
  return installStarted
}

function updaterPendingDir(): string {
  let dirName = 'desktop-updater'
  try {
    const text = readFileSync(join(process.resourcesPath, 'app-update.yml'), 'utf8')
    const matched = text.match(/^updaterCacheDirName:\s*['"]?([^'"\n#]+)['"]?\s*$/m)
    if (matched?.[1]) dirName = matched[1].trim()
  } catch {
    // 用 package name 对应的默认缓存目录
  }
  return join(homedir(), 'Library', 'Caches', dirName, 'pending')
}

async function newestZip(dir: string): Promise<string> {
  const names = await readdir(dir).catch(() => [])
  const zips = names.filter((name) => name.endsWith('.zip') && !name.startsWith('temp-'))
  if (zips.length === 0) throw new Error('找不到已下载的更新包，请重新下载')
  const ranked = await Promise.all(
    zips.map(async (name) => {
      const file = join(dir, name)
      const info = await stat(file)
      return { file, mtimeMs: info.mtimeMs }
    })
  )
  ranked.sort((left, right) => right.mtimeMs - left.mtimeMs)
  return ranked[0].file
}

function currentAppBundle(): string {
  const bundle = resolve(app.getPath('exe'), '..', '..', '..')
  if (!bundle.endsWith('.app')) throw new Error('无法定位当前应用')
  if (bundle.includes(`${sep}AppTranslocation${sep}`)) {
    throw new Error('请先把 Every Code 拖到「应用程序」文件夹，再安装更新')
  }
  return bundle
}

async function run(command: string, args: string[]): Promise<void> {
  try {
    await execFileAsync(command, args)
  } catch (error) {
    const stderr =
      error && typeof error === 'object' && 'stderr' in error ? String(error.stderr).trim() : ''
    const message = error instanceof Error ? error.message : '命令失败'
    throw new Error(stderr || message)
  }
}

async function stageApp(zipPath: string): Promise<string> {
  const extractDir = await mkdtemp(join(tmpdir(), 'every-code-update-'))
  try {
    await run('/usr/bin/ditto', ['-x', '-k', zipPath, extractDir])
    const appName = (await readdir(extractDir)).find((name) => name.endsWith('.app'))
    if (!appName) throw new Error('更新包里没有应用')
    const staged = join(extractDir, appName)
    try {
      await run('/usr/bin/codesign', ['--verify', '--deep', '--strict', staged])
    } catch (error) {
      const message = error instanceof Error ? error.message : '签名无效'
      throw new Error(`更新包签名无效：${message}`)
    }
    return staged
  } catch (error) {
    await rm(extractDir, { recursive: true, force: true })
    throw error
  }
}

async function scheduleSwap(bundlePath: string, stagedApp: string): Promise<void> {
  const script = `#!/bin/bash
set -euo pipefail
pid="$1"
target="$2"
source="$3"
backup="\${target}.previous"
i=0
while kill -0 "$pid" 2>/dev/null; do
  i=$((i + 1))
  if [ "$i" -gt 300 ]; then
    exit 1
  fi
  sleep 0.2
done
rm -rf "$backup"
if ! /bin/mv "$target" "$backup"; then
  exit 1
fi
if ! /bin/mv "$source" "$target"; then
  /bin/mv "$backup" "$target" || true
  exit 1
fi
/usr/bin/xattr -dr com.apple.quarantine "$target" || true
rm -rf "$backup"
/usr/bin/open "$target"
`
  const scriptPath = join(tmpdir(), `every-code-swap-${process.pid}.sh`)
  await writeFile(scriptPath, script, { mode: 0o700 })
  const child = spawn('/bin/bash', [scriptPath, String(process.pid), bundlePath, stagedApp], {
    detached: true,
    stdio: 'ignore'
  })
  child.unref()
  setTimeout(() => app.quit(), 200)
}

/**
 * macOS 不走 ShipIt。它会用当前进程的指定要求校验更新包，
 * ad-hoc 签名里的 cdhash 每次构建都变，校验过不了。
 */
export async function installDownloadedMacApp(): Promise<void> {
  if (installStarted) return
  installStarted = true
  try {
    const bundle = currentAppBundle()
    try {
      await access(resolve(bundle, '..'), constants.W_OK)
    } catch {
      throw new Error('没有权限替换当前应用。请把 Every Code 移到「应用程序」文件夹后再更新')
    }
    const staged = await stageApp(await newestZip(updaterPendingDir()))
    await scheduleSwap(bundle, staged)
  } catch (error) {
    installStarted = false
    throw error
  }
}
