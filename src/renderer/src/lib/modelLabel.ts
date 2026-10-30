const word: Record<string, string> = {
  deepseek: 'DeepSeek',
  gpt: 'GPT',
  chatgpt: 'ChatGPT',
  openai: 'OpenAI',
  claude: 'Claude',
  gemini: 'Gemini',
  qwen: 'Qwen',
  glm: 'GLM',
  kimi: 'Kimi',
  moonshot: 'Moonshot',
  minimax: 'MiniMax',
  llama: 'Llama',
  mistral: 'Mistral',
  grok: 'Grok',
  reasoner: 'Reasoner',
  chat: 'Chat',
  flash: 'Flash',
  pro: 'Pro',
  mini: 'Mini',
  turbo: 'Turbo',
  haiku: 'Haiku',
  sonnet: 'Sonnet',
  opus: 'Opus'
}

export function modelTitle(name: string): string {
  const raw = name.trim()
  if (!raw) return '未命名模型'
  if (/\s/.test(raw)) return raw
  let id = raw
  const colon = raw.indexOf(':')
  if (colon > 0 && colon <= 4) id = raw.slice(colon + 1)
  const slash = id.lastIndexOf('/')
  if (slash >= 0) id = id.slice(slash + 1)
  const parts = id.split(/[-_]/).filter(Boolean)
  if (parts.length <= 1) return raw
  return parts
    .map((part) => {
      if (/^\d+(\.\d+)*$/.test(part)) return part
      const known = word[part.toLowerCase()]
      if (known) return known
      if (/^v\d/i.test(part)) return part.toUpperCase()
      return part.charAt(0).toUpperCase() + part.slice(1)
    })
    .join(' ')
}

export function modelCode(name: string): string {
  const raw = name.trim()
  return modelTitle(raw) === raw ? '' : raw
}
