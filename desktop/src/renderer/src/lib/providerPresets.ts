import deepseekIcon from '../assets/providers/deepseek.svg'
import kimiIcon from '../assets/providers/kimi.svg'
import minimaxIcon from '../assets/providers/minimax.svg'
import qwenIcon from '../assets/providers/qwen.svg'
import zhipuIcon from '../assets/providers/zhipu.svg'

export type ProviderProtocol = 'openai' | 'anthropic'

export interface BuiltinProvider {
  key: string
  name: string
  website: string
  protocol: ProviderProtocol
  baseUrl: string
  aliases: string[]
  icon: string
}

export const builtinModelProviders: BuiltinProvider[] = [
  {
    key: 'deepseek',
    name: 'DeepSeek',
    website: 'https://deepseek.com',
    protocol: 'openai',
    baseUrl: 'https://api.deepseek.com/v1',
    aliases: ['deepseek'],
    icon: deepseekIcon
  },
  {
    key: 'zhipu',
    name: '智谱AI',
    website: 'https://www.bigmodel.cn',
    protocol: 'openai',
    baseUrl: 'https://open.bigmodel.cn/api/paas/v4',
    aliases: ['智谱ai', '智谱 glm', '智谱', 'zhipu', 'glm'],
    icon: zhipuIcon
  },
  {
    key: 'qwen',
    name: '阿里通义千问',
    website: 'https://www.aliyun.com/product/bailian',
    protocol: 'openai',
    baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    aliases: ['阿里通义千问', '通义千问', '通义', 'qwen', '千问'],
    icon: qwenIcon
  },
  {
    key: 'moonshot',
    name: 'Moonshot (Kimi)',
    website: 'https://www.moonshot.cn',
    protocol: 'openai',
    baseUrl: 'https://api.moonshot.cn/v1',
    aliases: ['moonshot (kimi)', 'moonshot', 'kimi', '月之暗面'],
    icon: kimiIcon
  },
  {
    key: 'minimax',
    name: 'MiniMax',
    website: 'https://www.minimaxi.com',
    protocol: 'openai',
    baseUrl: 'https://api.minimaxi.com/v1',
    aliases: ['minimax'],
    icon: minimaxIcon
  }
]

export function matchBuiltin(name: string): BuiltinProvider | undefined {
  const normalized = name.trim().toLowerCase()
  if (!normalized) return undefined
  return builtinModelProviders.find(
    (item) =>
      item.name.toLowerCase() === normalized ||
      item.aliases.some((alias) => alias.toLowerCase() === normalized)
  )
}
