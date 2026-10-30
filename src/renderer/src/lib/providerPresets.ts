export type ProviderProtocol = 'openai' | 'anthropic'

export interface BuiltinProvider {
  key: string
  name: string
  website: string
  protocol: ProviderProtocol
  baseUrl: string
}

/** 内置供应商。接口地址不要带 /models，服务端拉取远程模型时会自己补上。 */
export const builtinModelProviders: BuiltinProvider[] = [
  {
    key: 'deepseek',
    name: 'DeepSeek',
    website: 'https://www.deepseek.com',
    protocol: 'openai',
    baseUrl: 'https://api.deepseek.com/v1'
  },
  {
    key: 'openai',
    name: 'OpenAI',
    website: 'https://platform.openai.com',
    protocol: 'openai',
    baseUrl: 'https://api.openai.com/v1'
  },
  {
    key: 'anthropic',
    name: 'Anthropic',
    website: 'https://www.anthropic.com',
    protocol: 'anthropic',
    baseUrl: 'https://api.anthropic.com/v1'
  },
  {
    key: 'gemini',
    name: 'Gemini',
    website: 'https://ai.google.dev',
    protocol: 'openai',
    baseUrl: 'https://generativelanguage.googleapis.com/v1beta/openai'
  },
  {
    key: 'moonshot',
    name: 'Moonshot',
    website: 'https://platform.moonshot.cn',
    protocol: 'openai',
    baseUrl: 'https://api.moonshot.cn/v1'
  },
  {
    key: 'zhipu',
    name: '智谱',
    website: 'https://open.bigmodel.cn',
    protocol: 'openai',
    baseUrl: 'https://open.bigmodel.cn/api/paas/v4'
  },
  {
    key: 'minimax',
    name: 'MiniMax',
    website: 'https://platform.minimaxi.com',
    protocol: 'openai',
    baseUrl: 'https://api.minimaxi.com/v1'
  },
  {
    key: 'qwen',
    name: '通义千问',
    website: 'https://bailian.console.aliyun.com',
    protocol: 'openai',
    baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1'
  }
]

const aliases: Record<string, string[]> = {
  deepseek: ['deepseek'],
  openai: ['openai', 'chatgpt'],
  anthropic: ['anthropic', 'claude'],
  gemini: ['gemini', 'google', 'googleai'],
  moonshot: ['moonshot', 'kimi', '月之暗面'],
  zhipu: ['zhipu', '智谱', '智谱ai', 'glm', 'chatglm'],
  minimax: ['minimax'],
  qwen: ['qwen', '通义', '通义千问', '千问', 'tongyi']
}

function norm(name: string): string {
  return name.trim().toLowerCase().replace(/\s+/g, '')
}

export function matchBuiltin(name: string): BuiltinProvider | undefined {
  const key = norm(name)
  if (!key) return undefined
  return builtinModelProviders.find((preset) => (aliases[preset.key] || []).some((alias) => norm(alias) === key))
}
