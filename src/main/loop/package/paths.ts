import { existsSync, mkdirSync, writeFileSync } from 'fs'
import { homedir } from 'os'
import { join, resolve } from 'path'

const readmeName = 'README.txt'

const readme = `自定义工具只支持 .js 文件，一个文件定义一个工具，使用 CommonJS。

把文件直接放在这个目录下。子目录、.ts 和其他后缀都不会加载。
插件安装后，插件里的 tools/*.js 也会成为工具。

module.exports = {
  name: 'echo',
  description: '回显文本',
  parameters: [
    { name: 'text', type: 'string', description: '要回显的内容', required: true }
  ],
  async run(args, ctx) {
    // ctx.workspace 当前工作目录
    // ctx.env 插件环境变量
    // ctx.agentId 本次调用的智能体编号
    // ctx.packageDir 插件目录，用户工具为空字符串
    return String(args.text ?? '')
  }
}

name 以小写字母开头，只含小写字母、数字和下划线。
parameters.type 只能是 string、number、integer、boolean。
run 返回字符串，超时 150 秒。可以 require Node 内置模块，不能依赖额外 npm 包。
`

/** 本机数据根目录。测试可设置 EVERYCODE_HOME。 */
export function dataRoot(): string {
  const override = (process.env.EVERYCODE_HOME || '').trim()
  if (override) return resolve(override)
  return join(homedir(), '.every-code')
}

export function packagesRoot(): string {
  const dir = join(dataRoot(), 'packages')
  mkdirSync(dir, { recursive: true })
  return dir
}

export function packageDir(id: string, version: string): string {
  return join(packagesRoot(), id, version)
}

/** 用户自定义工具目录。首次创建时写入格式说明。 */
export function userToolsDir(): string {
  const dir = join(dataRoot(), 'tools')
  const fresh = !existsSync(dir)
  mkdirSync(dir, { recursive: true })
  const note = join(dir, readmeName)
  if (fresh || !existsSync(note)) writeFileSync(note, readme, 'utf8')
  return dir
}
