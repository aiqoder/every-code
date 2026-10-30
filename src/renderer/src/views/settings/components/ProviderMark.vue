<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  name: string
}>()

type Glyph = 'openai' | 'gemini' | 'anthropic' | 'deepseek' | 'moonshot' | 'zhipu' | 'minimax' | 'qwen' | 'generic'

const kind = computed<Glyph>(() => glyphKind(props.name))
const letter = computed(() => {
  const ch = props.name.trim().charAt(0)
  return ch ? ch.toUpperCase() : '?'
})
const color = computed(() => {
  let hash = 0
  for (const ch of props.name) hash = (hash * 31 + ch.charCodeAt(0)) >>> 0
  return `hsl(${hash % 360} 52% 46%)`
})

function glyphKind(name: string): Glyph {
  const n = name.toLowerCase()
  if (n.includes('openai') || n.includes('gpt')) return 'openai'
  if (n.includes('gemini') || n.includes('google')) return 'gemini'
  if (n.includes('anthropic') || n.includes('claude')) return 'anthropic'
  if (n.includes('deepseek')) return 'deepseek'
  if (n.includes('moonshot') || n.includes('kimi') || n.includes('月之暗面')) return 'moonshot'
  if (n.includes('zhipu') || n.includes('智谱') || n.includes('glm')) return 'zhipu'
  if (n.includes('minimax')) return 'minimax'
  if (n.includes('qwen') || n.includes('通义') || n.includes('千问')) return 'qwen'
  return 'generic'
}
</script>

<template>
  <span
    v-if="kind === 'generic'"
    class="grid h-[22px] w-[22px] shrink-0 place-items-center rounded-[7px] text-[12px] font-bold text-white"
    :style="{ background: color }"
    aria-hidden="true"
  >
    {{ letter }}
  </span>
  <svg v-else class="h-[22px] w-[22px] shrink-0" viewBox="0 0 24 24" aria-hidden="true">
    <template v-if="kind === 'openai'">
      <path
        fill="currentColor"
        d="M12 2.2c.5 0 .9.3 1.1.7l1.5 3.1 3.4.5c.5.1.8.5.8 1s-.3.9-.7 1.1L15.4 11l.6 3.4c.1.5-.1 1-.6 1.2-.4.2-.9.2-1.3 0L12 14.1l-2.1 1.5c-.4.3-.9.2-1.3 0-.5-.2-.7-.7-.6-1.2l.6-3.4-2.7-2.4c-.4-.3-.6-.8-.5-1.2.1-.5.5-.8 1-.8l3.4-.5 1.5-3.1c.2-.4.6-.7 1.1-.7z"
      />
      <circle cx="12" cy="12" r="2.1" fill="var(--ec-canvas-subtle)" />
    </template>
    <path
      v-else-if="kind === 'gemini'"
      fill="#4d7cfe"
      d="M12 2.2 13.8 9 20.6 12 13.8 15 12 21.8 10.2 15 3.4 12 10.2 9 12 2.2z"
    />
    <path
      v-else-if="kind === 'anthropic'"
      fill="#d97757"
      d="M6.2 19.2 12 4.6l5.8 14.6h-2.4l-1.2-3.2H9.8l-1.2 3.2H6.2zm4.1-5.2h3.4L12 9.2l-1.7 4.8z"
    />
    <template v-else-if="kind === 'deepseek'">
      <path
        fill="#4c6ef5"
        d="M3.6 13.2c.4-4.2 3.8-7.2 8-7.2 2.6 0 4.4 1 5.8 2.2.7.6 1.6.7 2.4.1.5-.4 1.3-.2 1.5.5.3 1.3-.2 2.8-1.4 3.8 1.1.7 1.8 1.9 1.8 3.3 0 2.7-2.5 4.8-6.4 4.8H9.2c-3.4 0-5.8-2.4-5.6-7.5z"
      />
      <circle cx="16.2" cy="11.2" r=".9" fill="#fff" />
    </template>
    <template v-else-if="kind === 'moonshot'">
      <circle cx="12" cy="12" r="9" fill="currentColor" />
      <path fill="var(--ec-canvas-subtle)" d="M9.2 7.2h2.1l3.6 6.2V7.2H17v9.6h-2.1l-3.6-6.2v6.2H9.2V7.2z" />
    </template>
    <path
      v-else-if="kind === 'zhipu'"
      fill="#2563eb"
      d="M12 3.2 14.2 9H20l-4.6 3.4L17 18.8 12 15.4 7 18.8l1.6-6.4L4 9h5.8L12 3.2z"
    />
    <path
      v-else-if="kind === 'minimax'"
      fill="none"
      stroke="#e11d48"
      stroke-width="1.8"
      stroke-linecap="round"
      d="M4 14c1.2-4 2.2-4 3.2 0s2 4 3.2 0 2-4 3.2 0 2 4 3.2 0 2-4 3.2 0"
    />
    <path
      v-else
      fill="#7c3aed"
      d="M8 4.5h6.2a5.3 5.3 0 0 1 0 10.6H11V19.5H8V4.5zm3 3v4.6h3.1a2.3 2.3 0 0 0 0-4.6H11z"
    />
  </svg>
</template>
