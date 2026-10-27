<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton } from 'naive-ui'
import type { Conversation } from '@/api/conversations'
import type { SocketStatus } from '@/lib/clientSocket'

const props = defineProps<{
  conversations: Conversation[]
  activeId: number | null
  userName: string
  socketStatus: SocketStatus
}>()

const groups = computed(() => {
  const order: string[] = []
  const buckets = new Map<string, Conversation[]>()
  for (const item of props.conversations) {
    const key = item.workspace?.trim() || ''
    const bucket = buckets.get(key)
    if (bucket) {
      bucket.push(item)
      continue
    }
    buckets.set(key, [item])
    order.push(key)
  }
  return order.map((key) => ({
    key: key || '__none__',
    path: key,
    label: key ? workspaceLabel(key) : '未选择工作目录',
    items: buckets.get(key) || []
  }))
})

function workspaceLabel(path: string): string {
  const parts = path.split(/[/\\]/).filter((part) => part.length > 0)
  return parts[parts.length - 1] || path
}

function conversationDate(value?: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec((value || '').trim())
  if (!match) return ''
  const month = Number(match[2])
  const day = Number(match[3])
  if (!month || !day) return ''
  return `${month}月${day}日`
}

const collapsed = ref(new Set<string>())

function expanded(key: string): boolean {
  return !collapsed.value.has(key)
}

function toggleGroup(key: string): void {
  const next = new Set(collapsed.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsed.value = next
}

const router = useRouter()
const route = useRoute()
const scheduleActive = computed(
  () => route.name === 'schedules' || route.name === 'schedule-detail'
)

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
  <aside class="flex w-72 shrink-0 flex-col border-r border-[var(--ec-border)] text-[var(--ec-fg)]">
    <div class="flex items-center justify-between px-4 pt-4">
      <div class="min-w-0">
        <p class="text-[11px] tracking-[0.18em] text-[var(--ec-accent)] uppercase">Every Code</p>
        <p class="mt-1 truncate text-sm text-[var(--ec-fg)]">{{ userName || '未命名' }}</p>
      </div>
      <n-button size="tiny" quaternary @click="emit('logout')">退出</n-button>
    </div>
    <div class="px-2 pt-3">
      <button
        type="button"
        class="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-[13px] text-[var(--ec-fg-muted)] hover:text-[var(--ec-fg)]"
        @click="emit('create')"
      >
        <svg viewBox="0 0 16 16" class="h-4 w-4 shrink-0" fill="none" aria-hidden="true">
          <circle cx="8" cy="8" r="6.15" stroke="currentColor" stroke-width="1.2" />
          <path
            d="M8 5.15v5.7M5.15 8h5.7"
            stroke="currentColor"
            stroke-width="1.2"
            stroke-linecap="round"
          />
        </svg>
        新对话
      </button>
      <button
        type="button"
        class="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-[13px]"
        :class="
          scheduleActive
            ? 'bg-[color-mix(in_srgb,var(--ec-accent)_16%,transparent)] text-[var(--ec-accent)]'
            : 'text-[var(--ec-fg-muted)] hover:text-[var(--ec-fg)]'
        "
        :aria-current="scheduleActive ? 'page' : undefined"
        @click="router.push({ name: 'schedules' })"
      >
        <svg viewBox="0 0 16 16" class="h-4 w-4 shrink-0" fill="none" aria-hidden="true">
          <circle cx="8" cy="8" r="6.15" stroke="currentColor" stroke-width="1.2" />
          <path
            d="M8 4.9V8.2l2.1 1.4"
            stroke="currentColor"
            stroke-width="1.2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
        定时任务
      </button>
    </div>
    <div class="mt-3 flex-1 overflow-auto px-2 pb-3">
      <section v-for="group in groups" :key="group.key" class="mb-3">
        <button
          type="button"
          class="flex w-full items-center gap-1 px-2 py-1 text-left text-xs text-[var(--ec-fg-muted)]"
          :title="group.path || undefined"
          :aria-expanded="expanded(group.key)"
          @click="toggleGroup(group.key)"
        >
          <svg
            viewBox="0 0 16 16"
            class="h-3 w-3 shrink-0 transition-transform"
            :class="expanded(group.key) ? 'rotate-90' : ''"
            fill="none"
            aria-hidden="true"
          >
            <path
              d="M6 4.25 10 8 6 11.75"
              stroke="currentColor"
              stroke-width="1.4"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          <svg viewBox="0 0 16 16" class="h-3.5 w-3.5 shrink-0" fill="none" aria-hidden="true">
            <path
              d="M2.2 4.6c0-.6.5-1.1 1.1-1.1h2.2l1.1 1.2h6.1c.6 0 1.1.5 1.1 1.1v5.6c0 .6-.5 1.1-1.1 1.1H3.3c-.6 0-1.1-.5-1.1-1.1V4.6Z"
              stroke="currentColor"
              stroke-width="1.2"
              stroke-linejoin="round"
            />
          </svg>
          <span class="min-w-0 truncate">{{ group.label }}</span>
        </button>
        <div v-if="expanded(group.key)">
          <div
            v-for="item in group.items"
            :key="item.id"
            class="group flex items-center gap-2 py-1 pr-1 pl-7"
          >
            <button
              type="button"
              class="min-w-0 flex-1 truncate text-left text-[13px]"
              :class="
                item.id === activeId
                  ? 'font-medium text-[var(--ec-fg)]'
                  : 'text-[var(--ec-fg-muted)]'
              "
              :title="item.title || '未命名对话'"
              @click="emit('select', item.id)"
            >
              {{ item.title || '未命名对话' }}
            </button>
            <span
              v-if="conversationDate(item.updatedAt || item.createdAt)"
              class="shrink-0 text-[11px] text-[var(--ec-fg-muted)] group-hover:hidden"
            >
              {{ conversationDate(item.updatedAt || item.createdAt) }}
            </span>
            <button
              type="button"
              title="删除"
              aria-label="删除"
              class="hidden h-6 w-6 shrink-0 place-items-center rounded-md text-[var(--ec-fg-muted)] group-hover:grid hover:text-[var(--ec-danger)]"
              @click="emit('remove', item.id)"
            >
              <svg viewBox="0 0 16 16" class="h-3.5 w-3.5" fill="none" aria-hidden="true">
                <path
                  d="M3.5 4.5h9M6.4 4.5V3.3h3.2v1.2M4.6 4.5l.5 8.1h5.8l.5-8.1"
                  stroke="currentColor"
                  stroke-width="1.3"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
          </div>
        </div>
      </section>
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
