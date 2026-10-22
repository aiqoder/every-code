import { execFile } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { promisify } from 'node:util'

const exec = promisify(execFile)

async function git(args: string[], cwd: string): Promise<{ code: number; out: string }> {
  try {
    const { stdout, stderr } = await exec('git', args, {
      cwd,
      maxBuffer: 32 * 1024 * 1024,
      timeout: 120_000
    })
    return { code: 0, out: (stdout || '') + (stderr || '') }
  } catch (err: any) {
    return { code: err?.code ?? 1, out: `${err?.stdout ?? ''}${err?.stderr ?? ''}${err?.message ?? ''}` }
  }
}

/** 目录还不是 Git 仓库则 init，并保证有初始提交与 main、dev 分支。 */
export async function initRepo(dir: string): Promise<{ ok: boolean; output: string; created: boolean }> {
  const lines: string[] = []
  if (!fs.existsSync(dir)) {
    return { ok: false, output: `目录不存在：${dir}`, created: false }
  }
  let created = false
  if (!fs.existsSync(path.join(dir, '.git'))) {
    const init = await git(['init', '-b', 'main'], dir)
    if (init.code !== 0) {
      // 老版本 git 不认 -b。
      await git(['init'], dir)
    }
    created = true
    lines.push('已初始化 Git 仓库')
  }

  const branchList = async (): Promise<string[]> => {
    const res = await git(['for-each-ref', '--format=%(refname:short)', 'refs/heads'], dir)
    return res.out
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s && !s.includes(' '))
  }

  // 空目录 add 不到东西，commit 会失败，分支就无从创建；用 --allow-empty 兜住。
  const head = await git(['rev-parse', '--verify', 'HEAD'], dir)
  if (head.code !== 0) {
    await git(['add', '-A'], dir)
    const commit = await git(
      [
        '-c',
        'user.name=ADE',
        '-c',
        'user.email=ade@local',
        'commit',
        '--allow-empty',
        '-m',
        'init: 仓库初始化'
      ],
      dir
    )
    if (commit.code !== 0) {
      return { ok: false, output: commit.out.trim() || '创建初始提交失败', created }
    }
    lines.push('已创建初始提交')
  }

  const current = await git(['rev-parse', '--abbrev-ref', 'HEAD'], dir)
  let cur = current.out.trim()
  if (!cur || cur === 'HEAD' || cur.includes('\n')) cur = ''

  let branches = await branchList()
  if (!branches.includes('main')) {
    // 已有内容的老仓库：以 dev 或当前分支为基准建 main，不重命名用户的分支。
    const base = branches.includes('dev') ? 'dev' : cur || 'HEAD'
    const create = await git(['branch', 'main', base], dir)
    if (create.code !== 0) {
      await git(['branch', '-M', 'main'], dir)
      lines.push('已把当前分支重命名为 main')
    } else {
      lines.push('已创建 main 分支')
    }
    branches = await branchList()
  }
  if (!branches.includes('dev')) {
    const create = await git(['branch', 'dev', 'main'], dir)
    if (create.code !== 0) {
      await git(['checkout', '-b', 'dev'], dir)
      await git(['checkout', cur || 'main'], dir)
    }
    lines.push('已创建 dev 分支')
  }

  const finals = await branchList()
  const ok = finals.includes('main') && finals.includes('dev')
  return { ok, output: lines.filter(Boolean).join('\n') || '仓库已就绪', created }
}

export async function repoStatus(dir: string): Promise<{
  isRepo: boolean
  branch: string
  dirty: boolean
  ready: boolean
}> {
  if (!fs.existsSync(path.join(dir, '.git'))) {
    return { isRepo: false, branch: '', dirty: false, ready: false }
  }
  const branch = await git(['rev-parse', '--abbrev-ref', 'HEAD'], dir)
  const status = await git(['status', '--porcelain'], dir)
  const head = await git(['rev-parse', '--verify', 'HEAD'], dir)
  const branches = await git(['for-each-ref', '--format=%(refname:short)', 'refs/heads'], dir)
  const names = branches.out.split('\n').map((s) => s.trim())
  return {
    isRepo: true,
    branch: branch.code === 0 ? branch.out.trim() : '',
    dirty: status.out.trim().length > 0,
    // 有提交且有 main、dev，才认为可以直接建 worktree。
    ready: head.code === 0 && names.includes('main') && names.includes('dev')
  }
}

/** 把 from 分支合入 into，非快进合并，冲突时放弃。 */
export async function mergeBranch(opts: {
  repo: string
  from: string
  into: string
}): Promise<{ ok: boolean; output: string }> {
  const { repo, from, into } = opts
  const status = await git(['status', '--porcelain'], repo)
  if (status.out.trim()) {
    return { ok: false, output: '主仓库有未提交改动，请先提交或暂存' }
  }
  const head = await git(['rev-parse', '--abbrev-ref', 'HEAD'], repo)
  const prev = head.out.trim()
  await git(['checkout', into], repo)
  const merge = await git(
    ['-c', 'user.name=ADE', '-c', 'user.email=ade@local', 'merge', '--no-ff', from, '-m', `merge: ${from} → ${into}`],
    repo
  )
  if (merge.code !== 0) {
    await git(['merge', '--abort'], repo)
    if (prev && prev !== into) await git(['checkout', prev], repo)
    return { ok: false, output: merge.out.trim() }
  }
  if (prev && prev !== into) await git(['checkout', prev], repo)
  return { ok: true, output: merge.out.trim() }
}

export async function diffBetween(opts: {
  repo: string
  base: string
  head: string
}): Promise<{ diff: string }> {
  const res = await git(['diff', `${opts.base}...${opts.head}`], opts.repo)
  const stat = await git(['diff', '--stat', `${opts.base}...${opts.head}`], opts.repo)
  const body = res.out.trim()
  const head = `变更概览：\n${stat.out.trim()}\n\n`
  if (body.length > 60_000) {
    return { diff: head + body.slice(0, 60_000) + '\n…（diff 已截断）' }
  }
  return { diff: head + body }
}
