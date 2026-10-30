<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { useRoute, useRouter } from 'vue-router'
import { useAnchoredPop } from '@/components/ui/useAnchoredPop'
import type { Conversation } from '@/api/conversations'
import type { SocketStatus } from '@/lib/clientSocket'
import { requestUpdateCheck } from '@/lib/updateCheck'
import brandIcon from '../../../../../../resources/icon.png'
import PlusIcon from './icons/PlusIcon.vue'
import FolderPlusIcon from './icons/FolderPlusIcon.vue'
import ClockIcon from './icons/ClockIcon.vue'
import AgentIcon from './icons/AgentIcon.vue'
import SettingsIcon from './icons/SettingsIcon.vue'
import ChevronRightIcon from './icons/ChevronRightIcon.vue'
import FolderIcon from './icons/FolderIcon.vue'
import TrashIcon from './icons/TrashIcon.vue'
import ChevronUpIcon from './icons/ChevronUpIcon.vue'

const props = defineProps<{
  conversations: Conversation[]
  activeId: number | null
  userName: string
  socketStatus: SocketStatus
  workspace: string
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
  const current = props.workspace.trim()
  if (!buckets.has(current)) {
    buckets.set(current, [])
    order.unshift(current)
  } else {
    const index = order.indexOf(current)
    if (index > 0) {
      order.splice(index, 1)
      order.unshift(current)
    }
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
const settingsActive = computed(
  () => route.name === 'settings' || route.name === 'settings-models'
)

function navClass(active: boolean): string {
  return active
    ? 'bg-[color-mix(in_srgb,var(--ec-accent)_16%,transparent)] text-[var(--ec-accent)]'
    : 'text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]'
}

const emit = defineEmits<{
  select: [id: number]
  create: [workspace: string]
  createWorkspace: []
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
        @click="emit('create', workspace)"
      >
        <PlusIcon class="h-4 w-4 shrink-0" />
        新对话
      </button>
      <button
        type="button"
        class="nav-item flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm"
        :class="navClass(scheduleActive)"
        :aria-current="scheduleActive ? 'page' : undefined"
        @click="router.push({ name: 'schedules' })"
      >
        <ClockIcon class="h-4 w-4 shrink-0" />
        定时任务
      </button>
      <button
        type="button"
        class="nav-item flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm"
        :class="navClass(agentsActive)"
        :aria-current="agentsActive ? 'page' : undefined"
        @click="router.push({ name: 'agents' })"
      >
        <AgentIcon class="h-4 w-4 shrink-0" />
        智能体
      </button>
      <button
        type="button"
        class="nav-item flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm"
        :class="navClass(settingsActive)"
        :aria-current="settingsActive ? 'page' : undefined"
        @click="router.push({ name: 'settings' })"
      >
        <SettingsIcon class="h-4 w-4 shrink-0" />
        设置
      </button>
    </div>
    <div class="mt-3 flex items-center justify-between px-4 py-1">
      <span class="text-xs text-[var(--ec-fg-muted)]">工作目录</span>
      <button
        type="button"
        title="新建工作目录"
        aria-label="新建工作目录"
        class="grid h-6 w-6 shrink-0 place-items-center rounded-md text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
        @click="emit('createWorkspace')"
      >
        <FolderPlusIcon class="h-3.5 w-3.5" />
      </button>
    </div>
    <div class="min-h-0 flex-1 overflow-auto px-2 pb-3">
      <section v-for="group in groups" :key="group.key" class="mb-3">
        <div class="group/folder flex items-center gap-0.5">
          <button
            type="button"
            class="flex min-w-0 flex-1 items-center gap-1 rounded-md px-2 py-1 text-left text-xs text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
            :title="group.path || undefined"
            :aria-expanded="expanded(group.key)"
            @click="toggleGroup(group.key)"
          >
            <ChevronRightIcon
              class="h-3 w-3 shrink-0 transition-transform"
              :class="expanded(group.key) ? 'rotate-90' : ''"
            />
            <FolderIcon class="h-3.5 w-3.5 shrink-0" />
            <span class="min-w-0 truncate">{{ group.label }}</span>
          </button>
          <button
            type="button"
            title="新建对话"
            aria-label="新建对话"
            class="grid h-6 w-6 shrink-0 place-items-center rounded-md text-[var(--ec-fg-muted)] opacity-0 pointer-events-none group-hover/folder:pointer-events-auto group-hover/folder:opacity-100 focus-visible:pointer-events-auto focus-visible:opacity-100 hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
            @click="emit('create', group.path)"
          >
            <PlusIcon class="h-3.5 w-3.5" />
          </button>
        </div>
        <div v-if="expanded(group.key)">
          <p
            v-if="group.items.length === 0"
            class="px-2 py-2 pl-7 text-sm text-[var(--ec-fg-muted)]"
          >
            还没有对话
          </p>
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
              <TrashIcon class="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      </section>
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
        <ChevronUpIcon
          class="h-3 w-3 shrink-0 transition-transform"
          :class="menuOpen ? 'rotate-180' : ''"
        />
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
