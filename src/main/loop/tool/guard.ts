const wrappers = new Set(['env', 'exec', 'nohup', 'time', 'nice', 'stdbuf', 'xargs', 'setsid'])
const shells = new Set(['sh', 'bash', 'zsh', 'dash', 'ksh', 'ash'])

/** 拒绝提权、关机和写磁盘设备。只看命令位置上的词，不拦参数里的同名文字。 */
export function guardShellCommand(src: string): void {
  for (const segment of splitSegments(src)) {
    checkWords(tokenize(segment))
  }
}

function deny(detail: string): never {
  throw new Error(`拒绝执行危险命令：${detail}`)
}

function checkWords(words: string[]): void {
  let i = 0
  while (i < words.length && isEnvAssign(words[i])) i++
  while (i < words.length) {
    if (isEnvAssign(words[i])) {
      i++
      continue
    }
    const base = cmdBase(words[i])
    if (base === 'command') {
      checkWords(words.slice(i + 1))
      return
    }
    if (base === 'eval') {
      guardShellCommand(words.slice(i + 1).join(' '))
      return
    }
    if (wrappers.has(base)) {
      i++
      while (i < words.length && words[i].startsWith('-') && words[i] !== '--') i++
      if (words[i] === '--') i++
      continue
    }
    break
  }
  if (i >= words.length) return
  checkArgv(words[i], words.slice(i + 1))
}

function checkArgv(argv0: string, args: string[]): void {
  const base = cmdBase(argv0)
  if (
    base === 'sudo' ||
    base === 'reboot' ||
    base === 'shutdown' ||
    base === 'halt' ||
    base === 'poweroff'
  ) {
    deny(base)
  }
  if (base === 'diskutil') {
    for (const arg of args) {
      if (arg.toLowerCase() === 'erasedisk' || arg.toLowerCase() === 'erasevolume') {
        deny(`diskutil ${arg}`)
      }
    }
  }
  if (base === 'dd') {
    for (const arg of args) {
      if (!arg.startsWith('of=')) continue
      const dest = arg.slice(3)
      if (isDiskDevice(dest)) deny(`dd 写入 ${dest}`)
    }
  }
  if (shells.has(base)) {
    const script = shellDashC(args)
    if (script) guardShellCommand(script)
  }
}

function shellDashC(args: string[]): string {
  for (let i = 0; i < args.length; i++) {
    const arg = args[i]
    if (arg === '--' || (arg && !arg.startsWith('-'))) return ''
    if (arg === '-c' || (arg.startsWith('-') && !arg.startsWith('--') && arg.includes('c'))) {
      return args[i + 1] || ''
    }
  }
  return ''
}

function isEnvAssign(word: string): boolean {
  const eq = word.indexOf('=')
  if (eq <= 0) return false
  return /^[A-Za-z_][A-Za-z0-9_]*$/.test(word.slice(0, eq))
}

function cmdBase(word: string): string {
  const cleaned = word.replace(/^['"]|['"]$/g, '')
  const slash = Math.max(cleaned.lastIndexOf('/'), cleaned.lastIndexOf('\\'))
  return (slash >= 0 ? cleaned.slice(slash + 1) : cleaned).toLowerCase()
}

function isDiskDevice(dest: string): boolean {
  const path = dest.replace(/\\/g, '/')
  return /^\/dev\/(r?disk|sd[a-z]|nvme|mmcblk|xvd|vd)/.test(path)
}

function splitSegments(src: string): string[] {
  const out: string[] = []
  let current = ''
  let quote = ''
  for (let i = 0; i < src.length; i++) {
    const ch = src[i]
    if (quote) {
      current += ch
      if (ch === quote && src[i - 1] !== '\\') quote = ''
      continue
    }
    if (ch === '"' || ch === "'") {
      quote = ch
      current += ch
      continue
    }
    if (ch === '\n' || ch === ';') {
      push(out, current)
      current = ''
      continue
    }
    if ((ch === '&' && src[i + 1] === '&') || (ch === '|' && src[i + 1] === '|') || ch === '|') {
      push(out, current)
      current = ''
      if (ch !== '|' || src[i + 1] === '|') i++
      continue
    }
    current += ch
  }
  push(out, current)
  return out
}

function push(out: string[], value: string): void {
  const trimmed = value.trim()
  if (trimmed) out.push(trimmed)
}

function tokenize(segment: string): string[] {
  const words: string[] = []
  let current = ''
  let quote = ''
  for (let i = 0; i < segment.length; i++) {
    const ch = segment[i]
    if (quote) {
      if (ch === quote) quote = ''
      else current += ch
      continue
    }
    if (ch === '"' || ch === "'") {
      quote = ch
      continue
    }
    if (/\s/.test(ch)) {
      if (current) words.push(current)
      current = ''
      continue
    }
    current += ch
  }
  if (current) words.push(current)
  return words
}
