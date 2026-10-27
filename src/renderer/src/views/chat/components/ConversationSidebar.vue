<script setup lang="ts">
import { NButton } from 'naive-ui'
import type { Conversation } from '@/api/conversations'
import type { SocketStatus } from '@/lib/clientSocket'

defineProps<{
  conversations: Conversation[]
  activeId: number | null
  userName: string
  socketStatus: SocketStatus
}>()

const emit = defineEmits<{
  select: [id: number]
  create: []
  remove: [id: number]
  logout: []
}>()

const statusText: Record<SocketStatus, string> = {
  idle: '未连接',
  connecting: '连接中',
  open: '工具通道已接通',
  closed: '已断开'
}
</script>

<template>
  <aside
    class="ec-skin-raised flex w-64 shrink-0 flex-col border-r border-[var(--ec-border)] text-[var(--ec-fg)]"
  >
    <div class="flex items-center justify-between px-4 pt-4">
      <div class="min-w-0">
        <p class="text-[11px] tracking-[0.18em] text-[var(--ec-accent)] uppercase">Every Code</p>
        <p class="mt-1 truncate text-sm text-[var(--ec-fg)]">{{ userName || '未命名' }}</p>
      </div>
      <n-button size="tiny" quaternary @click="emit('logout')">退出</n-button>
    </div>
    <div class="px-3 pt-4">
      <n-button dashed block @click="emit('create')">新对话</n-button>
    </div>
    <div class="mt-3 flex-1 overflow-auto px-2 pb-3">
      <div v-for="item in conversations" :key="item.id" class="group mb-1 flex items-center gap-1">
        <button
          type="button"
          class="min-w-0 flex-1 truncate rounded-lg px-2 py-2 text-left text-sm"
          :class="
            item.id === activeId
              ? 'bg-[color-mix(in_srgb,var(--ec-accent)_18%,transparent)] text-[var(--ec-fg)]'
              : 'text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)]'
          "
          @click="emit('select', item.id)"
        >
          {{ item.title || '未命名对话' }}
        </button>
        <n-button
          size="tiny"
          quaternary
          class="shrink-0 opacity-0 group-hover:opacity-100"
          @click="emit('remove', item.id)"
        >
          删除
        </n-button>
      </div>
      <p v-if="conversations.length === 0" class="px-2 py-6 text-sm text-[var(--ec-fg-muted)]">
        还没有对话
      </p>
    </div>
    <p class="border-t border-[var(--ec-border)] px-4 py-3 text-xs text-[var(--ec-fg-muted)]">
      <span
        class="mr-2 inline-block h-1.5 w-1.5 rounded-full"
        :class="socketStatus === 'open' ? 'bg-[var(--ec-success)]' : 'bg-[var(--ec-fg-muted)]'"
      />
      {{ statusText[socketStatus] }}
    </p>
  </aside>
</template>
