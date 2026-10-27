<script setup lang="ts">
import { ref } from 'vue'
import type { TimelineItem } from '../types'

defineProps<{
  items: TimelineItem[]
  sending: boolean
}>()

const openTools = ref<Record<string, boolean>>({})

function toggle(id: string): void {
  openTools.value[id] = !openTools.value[id]
}
</script>

<template>
  <div class="px-6 py-6">
    <div v-if="items.length === 0" class="mx-auto mt-24 max-w-lg text-center">
      <p class="text-xs tracking-[0.2em] text-[var(--ec-accent)] uppercase">Desk</p>
      <h2 class="mt-3 text-2xl font-semibold text-[var(--ec-fg)]">把任务交给当前工作目录</h2>
      <p class="mt-3 text-sm leading-6 text-[var(--ec-fg-muted)]">
        消息直接发给服务端会话。读文件、改代码、跑命令和技能脚本在这台电脑上执行。
      </p>
    </div>
    <div v-else class="mx-auto flex max-w-3xl flex-col gap-4">
      <article v-for="item in items" :key="item.id">
        <div
          v-if="item.kind === 'user'"
          class="ml-auto max-w-[85%] rounded-2xl bg-[var(--ec-accent)] px-4 py-3 text-sm leading-6 text-[var(--ec-on-accent)] select-text"
        >
          {{ item.text }}
        </div>
        <div
          v-else-if="item.kind === 'assistant'"
          class="max-w-[90%] text-sm leading-7 text-[var(--ec-fg)] select-text"
        >
          <p
            v-if="item.reasoning"
            class="mb-2 rounded-lg border border-[var(--ec-border)] bg-[var(--ec-canvas-inset)] px-3 py-2 text-xs leading-5 text-[var(--ec-fg-muted)]"
          >
            {{ item.reasoning }}
          </p>
          <p class="whitespace-pre-wrap">{{ item.text }}</p>
        </div>
        <div
          v-else-if="item.kind === 'tool'"
          class="rounded-xl border border-[var(--ec-border)] bg-[var(--ec-canvas-subtle)]"
        >
          <button
            type="button"
            class="flex w-full items-center gap-3 px-3 py-2 text-left text-xs text-[var(--ec-fg)]"
            @click="toggle(item.id)"
          >
            <span
              class="rounded border border-[var(--ec-border)] bg-[var(--ec-canvas-inset)] px-1.5 py-0.5 font-mono text-[var(--ec-accent)]"
              >{{ item.name }}</span
            >
            <span class="text-[var(--ec-fg-muted)]">{{ item.pending ? '执行中' : '完成' }}</span>
          </button>
          <pre
            v-if="openTools[item.id] || item.pending"
            class="max-h-64 overflow-auto border-t border-[var(--ec-border)] bg-[var(--ec-canvas-inset)] px-3 py-2 font-mono text-xs leading-5 whitespace-pre-wrap text-[var(--ec-fg)] select-text"
            >{{ item.args }}{{ item.result ? '\n' + item.result : '' }}</pre
          >
        </div>
        <p
          v-else
          class="text-sm"
          :class="item.tone === 'error' ? 'text-[var(--ec-danger)]' : 'text-[var(--ec-fg-muted)]'"
        >
          {{ item.text }}
        </p>
      </article>
      <p v-if="sending" class="text-xs text-[var(--ec-fg-muted)]">正在生成…</p>
    </div>
  </div>
</template>
