<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { useRoute, useRouter } from 'vue-router'
import { useAnchoredPop } from '@/components/ui/useAnchoredPop'
import type { Conversation } from '@/api/conversations'
import type { SocketStatus } from '@/lib/clientSocket'
import { requestUpdateCheck } from '@/lib/updateCheck'
import brandIcon from '../../../../../../resources/icon.png'

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
const mac = window.electron?.process.platform === 'darwin'
const scheduleActive = computed(
  () => route.name === 'schedules' || route.name === 'schedule-detail'
)
const agentsActive = computed(() => route.name === 'agents')
const settingsActive = computed(() => route.name === 'settings')

function navClass(active: boolean): string {
  return active
    ? 'bg-[color-mix(in_srgb,var(--ec-accent)_16%,transparent)] text-[var(--ec-accent)]'
    : 'text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]'
}

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

const menuOpen = ref(false)
const checking = ref(false)
const appVersion = ref('')
const triggerRef = ref<HTMLElement | null>(null)
const popRef = ref<HTMLElement | null>(null)
const { style: popStyle, prepare } = useAnchoredPop(menuOpen, triggerRef, popRef, () => true)
const updaterReady = typeof window !== 'undefined' && Boolean(window.api?.updater)
const updateLabel = computed(() =>
  appVersion.value ? `检查更新（v${appVersion.value}）` : '检查更新'
)

onClickOutside(
  triggerRef,
  () => {
    menuOpen.value = false
  },
  { ignore: [popRef] }
)

onMounted(() => {
  const updater = window.api?.updater
  if (!updater) return
  void updater
    .version()
    .then((version) => {
      appVersion.value = version
    })
    .catch(() => {
      appVersion.value = ''
    })
})

function toggleMenu(): void {
  if (checking.value) return
  if (!menuOpen.value) prepare()
  menuOpen.value = !menuOpen.value
}

function logout(): void {
  menuOpen.value = false
  emit('logout')
}

async function checkUpdate(): Promise<void> {
  if (!updaterReady || checking.value) return
  checking.value = true
  try {
    await requestUpdateCheck()
  } finally {
    checking.value = false
    menuOpen.value = false
  }
}
</script>

<template>
  <aside class="ec-nav flex w-72 shrink-0 flex-col border-r border-[var(--ec-border)] text-[var(--ec-fg)]">
    <div
      class="flex shrink-0 items-center gap-2"
      :class="mac ? 'window-drag h-12 pr-3 pl-20' : 'px-4 pt-4 pb-2'"
    >
      <img :src="brandIcon" alt="" class="h-5 w-5 shrink-0 rounded-[5px]" />
      <span class="min-w-0 truncate text-sm font-semibold tracking-tight">Every Code</span>
    </div>
    <div class="px-2" :class="mac ? 'pt-1' : ''">
      <button
        type="button"
        class="nav-item flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
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
        class="nav-item flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm"
        :class="navClass(scheduleActive)"
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
      <button
        type="button"
        class="nav-item flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm"
        :class="navClass(agentsActive)"
        :aria-current="agentsActive ? 'page' : undefined"
        @click="router.push({ name: 'agents' })"
      >
        <svg viewBox="0 0 16 16" class="h-4 w-4 shrink-0" fill="none" aria-hidden="true">
          <rect
            x="3"
            y="4.25"
            width="10"
            height="8"
            rx="2"
            stroke="currentColor"
            stroke-width="1.2"
          />
          <path d="M8 2.4v1.85" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
          <circle cx="6.3" cy="8.15" r="0.7" fill="currentColor" />
          <circle cx="9.7" cy="8.15" r="0.7" fill="currentColor" />
        </svg>
        智能体
      </button>
      <button
        type="button"
        class="nav-item flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm"
        :class="navClass(settingsActive)"
        :aria-current="settingsActive ? 'page' : undefined"
        @click="router.push({ name: 'settings' })"
      >
        <svg viewBox="0 0 16 16" class="h-4 w-4 shrink-0" fill="none" aria-hidden="true">
          <path
            d="M2.75 4.75h1.9M8.15 4.75h5.1M2.75 11.25h5.1M12.15 11.25h1.1"
            stroke="currentColor"
            stroke-width="1.2"
            stroke-linecap="round"
          />
          <circle cx="6" cy="4.75" r="1.45" stroke="currentColor" stroke-width="1.2" />
          <circle cx="10" cy="11.25" r="1.45" stroke="currentColor" stroke-width="1.2" />
        </svg>
        设置
      </button>
    </div>
    <div class="mt-3 flex-1 overflow-auto px-2 pb-3">
      <section v-for="group in groups" :key="group.key" class="mb-3">
        <button
          type="button"
          class="flex w-full items-center gap-1 rounded-md px-2 py-1 text-left text-xs text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
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
            class="conv-row group flex w-full items-center gap-2 rounded-md py-1 pr-1 pl-7 hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)]"
          >
            <button
              type="button"
              class="conv-hit min-w-0 flex-1 truncate text-left text-sm"
              :class="
                item.id === activeId
                  ? 'conv-active font-medium text-[var(--ec-fg)]'
                  : 'text-[var(--ec-fg-muted)]'
              "
              :title="item.title || '未命名对话'"
              @click="emit('select', item.id)"
            >
              {{ item.title || '未命名对话' }}
            </button>
            <span
              v-if="conversationDate(item.updatedAt || item.createdAt)"
              class="shrink-0 text-xs text-[var(--ec-fg-muted)] group-hover:hidden"
            >
              {{ conversationDate(item.updatedAt || item.createdAt) }}
            </span>
            <button
              type="button"
              title="删除"
              aria-label="删除"
              class="conv-hit hidden h-6 w-6 shrink-0 place-items-center rounded-md text-[var(--ec-fg-muted)] group-hover:grid hover:text-[var(--ec-danger)]"
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
    <div class="border-t border-[var(--ec-border)] p-2">
      <button
        ref="triggerRef"
        type="button"
        class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
        :aria-expanded="menuOpen"
        aria-haspopup="menu"
        @click="toggleMenu"
      >
        <span
          class="inline-block h-1.5 w-1.5 shrink-0 rounded-full"
          :class="socketStatus === 'open' ? 'bg-[var(--ec-success)]' : 'bg-[var(--ec-fg-muted)]'"
        />
        <span class="min-w-0 flex-1 truncate">{{ statusText[socketStatus] }}</span>
        <svg
          viewBox="0 0 16 16"
          class="h-3 w-3 shrink-0 transition-transform"
          :class="menuOpen ? 'rotate-180' : ''"
          fill="none"
          aria-hidden="true"
        >
          <path
            d="M4 10.25 8 6.25 12 10.25"
            stroke="currentColor"
            stroke-width="1.4"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
      <Teleport to="body">
        <div
          v-if="menuOpen"
          ref="popRef"
          class="account-pop ec-stage"
          role="menu"
          :style="popStyle"
          @click.stop
        >
          <button
            type="button"
            class="item server"
            role="menuitem"
            :title="userName || '未设置'"
            @click="menuOpen = false"
          >
            <span class="kicker">当前服务器</span>
            <span class="host">{{ userName || '未设置' }}</span>
          </button>
          <button
            type="button"
            class="item"
            role="menuitem"
            :disabled="checking || !updaterReady"
            @click="checkUpdate"
          >
            {{ checking ? '正在检查更新' : updateLabel }}
          </button>
          <button type="button" class="item danger" role="menuitem" @click="logout">退出登录</button>
        </div>
      </Teleport>
    </div>
  </aside>
</template>

<style scoped>
.nav-item:hover,
.conv-hit:hover {
  background-image: none;
}

.account-pop {
  box-sizing: border-box;
  overflow: auto;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
}

.item {
  display: flex;
  width: 100%;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--ec-text-body);
  line-height: 1.35;
  text-align: left;
  cursor: pointer;
}

.item:hover:not(:disabled) {
  background: var(--surface-hover);
}

.item:disabled {
  cursor: default;
  opacity: 0.6;
}

.item.server {
  flex-direction: row;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.item.danger {
  color: var(--error);
}

.item.danger:hover:not(:disabled) {
  background: color-mix(in srgb, var(--error) 12%, transparent);
}

.kicker {
  flex-shrink: 0;
  font-size: var(--ec-text-meta);
  line-height: 1.3;
  color: var(--text-muted);
}

.host {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
