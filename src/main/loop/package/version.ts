export interface SemVer {
  major: number
  minor: number
  patch: number
  prerelease: string
}

/** 解析 major.minor.patch，允许 v 前缀和预发布后缀。 */
export function parseVersion(input: string): SemVer {
  let text = input.trim().replace(/^v/, '')
  if (!text) throw new Error('版本号为空')
  let prerelease = ''
  const dash = text.indexOf('-')
  if (dash >= 0) {
    prerelease = text.slice(dash + 1)
    text = text.slice(0, dash)
    if (!prerelease) throw new Error('预发布标识为空')
  }
  const plus = text.indexOf('+')
  if (plus >= 0) text = text.slice(0, plus)
  const parts = text.split('.')
  if (parts.length > 3) throw new Error(`版本号段数过多: ${input}`)
  const nums = [0, 0, 0]
  for (let i = 0; i < parts.length; i++) {
    if (!/^\d+$/.test(parts[i])) throw new Error(`版本号段无效: ${parts[i]}`)
    nums[i] = Number(parts[i])
  }
  return { major: nums[0], minor: nums[1], patch: nums[2], prerelease }
}

/** a<b 返回 -1，相等返回 0，a>b 返回 1。预发布版本低于同号正式版。 */
export function compareVersions(a: string, b: string): number {
  const left = parseVersion(a)
  const right = parseVersion(b)
  if (left.major !== right.major) return left.major < right.major ? -1 : 1
  if (left.minor !== right.minor) return left.minor < right.minor ? -1 : 1
  if (left.patch !== right.patch) return left.patch < right.patch ? -1 : 1
  if (left.prerelease === right.prerelease) return 0
  if (!left.prerelease) return 1
  if (!right.prerelease) return -1
  return left.prerelease < right.prerelease ? -1 : 1
}
