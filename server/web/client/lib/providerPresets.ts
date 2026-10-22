// 常见大模型供应商预设：选中后自动填充协议类型与接口地址，用户只需补 API Key。
export type ProviderProtocol = 'openai' | 'anthropic'

export interface ProviderPreset {
  name: string
  website: string
  protocol: ProviderProtocol
  baseUrl: string
}

export const providerPresets: ProviderPreset[] = [
  {
    name: 'OpenAI',
    website: 'https://openai.com',
    protocol: 'openai',
    baseUrl: 'https://api.openai.com/v1',
  },
  {
    name: 'DeepSeek',
    website: 'https://deepseek.com',
    protocol: 'openai',
    baseUrl: 'https://api.deepseek.com/v1',
  },
  {
    name: 'OpenRouter',
    website: 'https://openrouter.ai',
    protocol: 'openai',
    baseUrl: 'https://openrouter.ai/api/v1',
  },
  {
    name: '硅基流动',
    website: 'https://siliconflow.cn',
    protocol: 'openai',
    baseUrl: 'https://api.siliconflow.cn/v1',
  },
  {
    name: '月之暗面',
    website: 'https://www.moonshot.cn',
    protocol: 'openai',
    baseUrl: 'https://api.moonshot.cn/v1',
  },
  {
    name: '智谱 GLM',
    website: 'https://www.bigmodel.cn',
    protocol: 'openai',
    baseUrl: 'https://open.bigmodel.cn/api/paas/v4',
  },
  {
    name: 'Anthropic',
    website: 'https://www.anthropic.com',
    protocol: 'anthropic',
    baseUrl: 'https://api.anthropic.com',
  },
]

export function protocolLabel(protocol: ProviderProtocol): string {
  return protocol === 'anthropic' ? 'Anthropic' : 'OpenAI 兼容'
}
