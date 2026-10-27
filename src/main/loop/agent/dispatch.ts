import { LoopMessage, LoopType } from '../../../shared/loop'
import { readMarkdown, runScript } from '../skill/host'
import { runTool } from '../tool'

/** 处理服务端经 /clients/ws 下发的工具与技能帧，回包带上原来的 id。 */
export async function dispatch(message: LoopMessage): Promise<LoopMessage> {
  const id = message?.id || ''
  try {
    switch (message?.type) {
      case LoopType.ping:
        return { type: LoopType.pong, id }
      case LoopType.toolCall: {
        const { name, args } = decodeToolCall(message)
        return ok(
          LoopType.toolResult,
          id,
          await runTool(name, args, { agentId: (message.metadata?.agentId || '').trim() })
        )
      }
      case LoopType.skillCall: {
        const args = asRecord(message.payload)
        const file = stringField(args, 'file') || 'SKILL.md'
        return ok(LoopType.skillResult, id, await readMarkdown(folderOf(message), file))
      }
      case LoopType.skillRunner: {
        const args = asRecord(message.payload)
        const script = stringField(args, 'script')
        const stdin = stringField(args, 'stdin')
        const list = Array.isArray(args.args) ? args.args.map((item) => String(item)) : []
        return ok(
          LoopType.skillRunnerResult,
          id,
          await runScript(folderOf(message), script, list, stdin)
        )
      }
      case LoopType.skillLoad: {
        const args = asRecord(message.payload)
        const file = stringField(args, 'file') || 'SKILL.md'
        return ok(LoopType.skillContent, id, await readMarkdown(folderOf(message), file))
      }
      default:
        return fail(id, `unknown message type: ${message?.type || ''}`)
    }
  } catch (err) {
    const replyType = resultType(message?.type)
    const text = err instanceof Error ? err.message : String(err)
    return { type: replyType, id, data: '', error: text }
  }
}

function ok(type: string, id: string, data: string): LoopMessage {
  return { type, id, data, error: '' }
}

function fail(id: string, error: string): LoopMessage {
  return { type: LoopType.error, id, error }
}

function resultType(type: string | undefined): string {
  switch (type) {
    case LoopType.toolCall:
      return LoopType.toolResult
    case LoopType.skillRunner:
      return LoopType.skillRunnerResult
    case LoopType.skillLoad:
      return LoopType.skillContent
    default:
      return LoopType.skillResult
  }
}

function decodeToolCall(message: LoopMessage): { name: string; args: Record<string, unknown> } {
  const metaName = (message.metadata?.name || '').trim()
  if (metaName) {
    return { name: metaName, args: asRecord(message.payload) }
  }
  const payload = asRecord(message.payload)
  const name = stringField(payload, 'name')
  const rawArgs = payload.arguments
  if (typeof rawArgs === 'string') {
    return { name, args: asRecord(parseJson(rawArgs)) }
  }
  if (rawArgs && typeof rawArgs === 'object') {
    return { name, args: asRecord(rawArgs) }
  }
  return { name, args: payload }
}

function folderOf(message: LoopMessage): string {
  return (message.metadata?.folder || '').trim()
}

function asRecord(value: unknown): Record<string, unknown> {
  if (typeof value === 'string') {
    const parsed = parseJson(value)
    return parsed && typeof parsed === 'object' ? (parsed as Record<string, unknown>) : {}
  }
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

function parseJson(value: string): unknown {
  const text = value.trim()
  if (!text) return {}
  try {
    return JSON.parse(text)
  } catch {
    return {}
  }
}

function stringField(record: Record<string, unknown>, key: string): string {
  const value = record[key]
  return value == null ? '' : String(value).trim()
}
