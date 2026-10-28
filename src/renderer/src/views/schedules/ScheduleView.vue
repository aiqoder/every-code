<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, toRaw, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UiMenu from '@/components/ui/UiMenu.vue'
import { useDialog, useMessage } from '@/components/ui/feedback'
import {
  listAgents,
  listModels,
  resolveAgentKey,
  type AgentItem,
  type ModelItem
} from '@/api/catalog'
import { useSessionStore } from '@/stores/sessionStore'
import {
  describeRule,
  failureHint,
  formatDateTime,
  formatDuration,
  runStatusLabel,
  type ScheduleDraft,
  type ScheduleRun,
  type ScheduleRunStatus,
  type ScheduledTask
} from '@shared/schedule'
import ScheduleCard from './components/ScheduleCard.vue'
import ScheduleEditor from './components/ScheduleEditor.vue'
import TemplatePicker from './components/TemplatePicker.vue'
import { onScheduleChanged, runScheduledTask } from './runner'
import { draftFromTemplate, scheduleTemplates, type ScheduleTemplate } from './templates'

type TaskFilter = 'all' | 'enabled' | 'paused'
type PageTab = 'tasks' | 'runs'

const session = useSessionStore()
const route = useRoute()
const router = useRouter()
const dialog = useDialog()
const message = useMessage()

const tasks = ref<ScheduledTask[]>([])
const runs = ref<ScheduleRun[]>([])
const agents = ref<AgentItem[]>([])
const models = ref<ModelItem[]>([])
const workspace = ref('')
const tab = ref<PageTab>('tasks')
const taskFilter = ref<TaskFilter>('all')
const runStatus = ref<ScheduleRunStatus | 'all'>('all')
const runTaskId = ref('all')
const showArchived = ref(false)
const query = ref('')
const searching = ref(false)
const batch = ref(false)
const selected = ref<string[]>([])
const editorOpen = ref(false)
const templateOpen = ref(false)
const saving = ref(false)
const editorDraft = ref<ScheduleDraft | null>(null)
const expandedRun = ref('')
const loadError = ref('')

const taskFilters: { id: TaskFilter; label: string }[] = [
  { id: 'all', label: '全部' },
  { id: 'enabled', label: '已启用' },
  { id: 'paused', label: '已暂停' }
]

const runFilters: { id: ScheduleRunStatus | 'all'; label: string }[] = [
  { id: 'all', label: '全部' },
  { id: 'success', label: '成功' },
  { id: 'failed', label: '失败' },
  { id: 'running', label: '运行中' },
  { id: 'waiting', label: '等待确认' }
]

const createOptions = [
  { label: '手动新建', key: 'manual' },
  { label: '从任务新建', key: 'chat' },
  { label: '从模版新建', key: 'template' }
]

const shownTasks = computed(() => {
  const text = query.value.trim().toLowerCase()
  return tasks.value
    .filter((item) => {
      if (taskFilter.value === 'enabled' && !item.enabled) return false
      if (taskFilter.value === 'paused' && item.enabled) return false
      if (!text) return true
      return item.name.toLowerCase().includes(text) || item.prompt.toLowerCase().includes(text)
    })
    .slice()
    .sort((a, b) => {
      if (a.enabled !== b.enabled) return a.enabled ? -1 : 1
      return (a.nextRunAt || '9999').localeCompare(b.nextRunAt || '9999')
    })
})

const shownRuns = computed(() => {
  const text = query.value.trim().toLowerCase()
  return runs.value.filter((item) => {
    if (!showArchived.value && item.archived) return false
    if (runStatus.value !== 'all' && item.status !== runStatus.value) return false
    if (runTaskId.value !== 'all' && item.taskId !== runTaskId.value) return false
    if (!text) return true
    return (
      item.taskName.toLowerCase().includes(text) ||
      item.summary.toLowerCase().includes(text) ||
      item.error.toLowerCase().includes(text)
    )
  })
})

const allChecked = computed(
  () =>
    shownTasks.value.length > 0 &&
    shownTasks.value.every((item) => selected.value.includes(item.id))
)

let stopWatch: () => void = () => {}

onMounted(() => {
  applyRoute()
  void boot()
  stopWatch = onScheduleChanged(() => {
    void reload()
  })
})

onUnmounted(() => stopWatch())

watch(
  () => session.loggedIn,
  (loggedIn, was) => {
    if (loggedIn && !was) void boot()
  }
)

watch(
  () => route.query.task,
  () => applyRoute()
)

function applyRoute(): void {
  if (route.query.tab === 'runs') tab.value = 'runs'
  if (typeof route.query.task === 'string' && route.query.task) {
    runTaskId.value = route.query.task
    tab.value = 'runs'
  }
}

async function boot(): Promise<void> {
  if (window.api?.loop) workspace.value = (await window.api.loop.getWorkspace()) || ''
  await reload()
  if (!session.loggedIn) return
  const [agentList, modelList] = await Promise.all([
    listAgents().catch(() => []),
    listModels().catch(() => [])
  ])
  agents.value = agentList
  models.value = modelList
}

async function reload(): Promise<void> {
  if (!window.api?.schedule) {
    loadError.value = '定时任务只能在桌面端使用'
    return
  }
  loadError.value = ''
  try {
    const state = await window.api.schedule.state()
    tasks.value = state.tasks
    runs.value = state.runs
    selected.value = selected.value.filter((id) => state.tasks.some((item) => item.id === id))
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : '读取定时任务失败'
  }
}

function rememberedAgent(): { agentId: string; agentName: string } {
  const agentId = resolveAgentKey(localStorage.getItem('everycode.agentId') || '', agents.value)
  const agent = agents.value.find((item) => item.agentId === agentId)
  return { agentId: agent?.agentId || '', agentName: agent?.name || '' }
}

function rememberedModel(): number | null {
  const parsed = Number(localStorage.getItem('everycode.modelId'))
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
}

function openManual(): void {
  const agent = rememberedAgent()
  editorDraft.value = {
    name: '',
    prompt: '',
    agentId: agent.agentId,
    agentName: agent.agentName,
    modelId: rememberedModel(),
    workspace: workspace.value,
    enabled: true,
    rule: {
      repeat: 'daily',
      intervalMinutes: 60,
      time: '09:00',
      onceAt: '',
      weekdays: [1, 2, 3, 4, 5],
      monthDays: [1],
      month: 1,
      day: 1,
      startDate: '',
      endDate: ''
    }
  }
  editorOpen.value = true
}

function openTemplate(template: ScheduleTemplate): void {
  templateOpen.value = false
  const agent = rememberedAgent()
  editorDraft.value = draftFromTemplate(template, {
    ...agent,
    modelId: rememberedModel(),
    workspace: workspace.value
  })
  editorOpen.value = true
}

function openEdit(task: ScheduledTask): void {
  editorDraft.value = {
    id: task.id,
    name: task.name,
    prompt: task.prompt,
    agentId: task.agentId,
    agentName: task.agentName,
    modelId: task.modelId,
    workspace: task.workspace,
    rule: structuredClone(toRaw(task.rule)),
    enabled: task.enabled
  }
  editorOpen.value = true
}

function onCreate(key: string | number): void {
  if (key === 'manual') openManual()
  if (key === 'template') templateOpen.value = true
  if (key === 'chat') {
    void router.push({ name: 'chat', query: { draft: '帮我创建一个定时任务：' } })
  }
}

async function onSave(draft: ScheduleDraft): Promise<void> {
  if (!window.api?.schedule) return
  saving.value = true
  try {
    await window.api.schedule.save(draft)
    editorOpen.value = false
    await reload()
  } catch (err) {
    message.error(err instanceof Error ? err.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function onToggle(task: ScheduledTask, enabled: boolean): Promise<void> {
  if (!window.api?.schedule) return
  try {
    await window.api.schedule.setEnabled(task.id, enabled)
    await reload()
  } catch (err) {
    message.error(err instanceof Error ? err.message : '切换失败')
  }
}

async function onRun(task: ScheduledTask): Promise<void> {
  try {
    await runScheduledTask(task.id)
  } catch (err) {
    message.error(err instanceof Error ? err.message : '运行失败')
  }
}

function confirmRemove(ids: string[]): void {
  if (ids.length === 0) return
  dialog.warning({
    title: '删除定时任务',
    content:
      ids.length === 1
        ? '删除后任务不再执行，此操作无法撤销。'
        : `删除选中的 ${ids.length} 个定时任务？删除后不再执行，此操作无法撤销。`,
    positiveText: '删除',
    negativeText: '取消',
    danger: true,
    onPositiveClick: async () => {
      if (!window.api?.schedule) return
      await window.api.schedule.remove(ids)
      selected.value = []
      batch.value = false
      await reload()
    }
  })
}

function openTask(task: ScheduledTask): void {
  void router.push({ name: 'schedule-detail', params: { id: task.id } })
}

function openRecords(task: ScheduledTask): void {
  tab.value = 'runs'
  runTaskId.value = task.id
  void router.replace({ name: 'schedules', query: { tab: 'runs', task: task.id } })
}

function toggleAll(event: Event): void {
  const checked = (event.target as HTMLInputElement).checked
  selected.value = checked ? shownTasks.value.map((item) => item.id) : []
}

function toggleOne(id: string, checked: boolean): void {
  selected.value = checked ? [...selected.value, id] : selected.value.filter((item) => item !== id)
}

function taskOf(run: ScheduleRun): ScheduledTask | undefined {
  return tasks.value.find((item) => item.id === run.taskId)
}

function openRun(run: ScheduleRun): void {
  if (run.status === 'failed' && !run.conversationId) {
    expandedRun.value = expandedRun.value === run.id ? '' : run.id
    return
  }
  if (run.conversationId) {
    const task = taskOf(run)
    void router.push({
      name: 'chat',
      query: {
        conversation: String(run.conversationId),
        ...(run.status === 'waiting' && task ? { draft: task.prompt } : {})
      }
    })
    return
  }
  expandedRun.value = expandedRun.value === run.id ? '' : run.id
}

async function removeRun(run: ScheduleRun): Promise<void> {
  if (!window.api?.schedule) return
  await window.api.schedule.removeRuns([run.id])
  await reload()
}

async function archiveRun(run: ScheduleRun, archived: boolean): Promise<void> {
  if (!window.api?.schedule) return
  await window.api.schedule.archiveRuns([run.id], archived)
  await reload()
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col text-[var(--ec-fg)]">
    <div class="window-drag flex shrink-0 items-center gap-3 border-b border-[var(--ec-border)] px-5">
      <div class="flex gap-1" role="tablist">
        <button
          type="button"
          role="tab"
          class="border-b-2 px-3 py-3 text-sm"
          :class="
            tab === 'tasks'
              ? 'border-[var(--ec-accent)] font-medium'
              : 'border-transparent text-[var(--ec-fg-muted)] hover:text-[var(--ec-fg)]'
          "
          :aria-selected="tab === 'tasks'"
          @click="tab = 'tasks'"
        >
          定时任务
        </button>
        <button
          type="button"
          role="tab"
          class="border-b-2 px-3 py-3 text-sm"
          :class="
            tab === 'runs'
              ? 'border-[var(--ec-accent)] font-medium'
              : 'border-transparent text-[var(--ec-fg-muted)] hover:text-[var(--ec-fg)]'
          "
          :aria-selected="tab === 'runs'"
          @click="tab = 'runs'"
        >
          运行记录
        </button>
      </div>
      <span class="flex-1" />
      <input
        v-if="searching"
        v-model="query"
        type="search"
        class="h-8 w-52 rounded-md border border-[var(--ec-field-border)] bg-[var(--ec-field)] px-2 text-sm text-[var(--ec-fg)] outline-none"
        :placeholder="tab === 'tasks' ? '搜索任务' : '搜索运行记录'"
        aria-label="搜索"
      />
      <button type="button" class="bar" @click="searching = !searching">搜索</button>
      <template v-if="tab === 'tasks' && batch">
        <button
          type="button"
          class="bar"
          :disabled="selected.length === 0"
          @click="confirmRemove(selected)"
        >
          删除
        </button>
        <button type="button" class="bar" @click="batch = false">完成</button>
      </template>
      <template v-else-if="tab === 'tasks'">
        <button type="button" class="bar" @click="batch = true">批量管理</button>
        <div
          class="inline-flex overflow-hidden rounded-md bg-[var(--ec-accent)] text-[var(--ec-on-accent)]"
        >
          <button type="button" class="px-3 py-1 text-sm hover:bg-white/15" @click="openManual">
            新建
          </button>
          <UiMenu :options="createOptions" @select="onCreate">
            <button
              type="button"
              class="border-l border-white/30 px-2 py-1 text-sm hover:bg-white/15"
              aria-label="更多创建方式"
            >
              ▾
            </button>
          </UiMenu>
        </div>
      </template>
    </div>

    <div v-if="tab === 'tasks'" class="flex shrink-0 items-center gap-1 px-5 pt-3">
      <label v-if="batch" class="mr-2 flex items-center gap-2 text-xs text-[var(--ec-fg-muted)]">
        <input type="checkbox" class="h-4 w-4" :checked="allChecked" @change="toggleAll" />
        全选
      </label>
      <button
        v-for="item in taskFilters"
        :key="item.id"
        type="button"
        class="rounded-md px-2.5 py-1 text-xs"
        :class="
          taskFilter === item.id
            ? 'bg-[color-mix(in_srgb,var(--ec-accent)_16%,transparent)] text-[var(--ec-accent)]'
            : 'text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)]'
        "
        @click="taskFilter = item.id"
      >
        {{ item.label }}
      </button>
    </div>

    <div v-else class="flex shrink-0 flex-wrap items-center gap-2 px-5 pt-3">
      <button
        v-for="item in runFilters"
        :key="item.id"
        type="button"
        class="rounded-md px-2.5 py-1 text-xs"
        :class="
          runStatus === item.id
            ? 'bg-[color-mix(in_srgb,var(--ec-accent)_16%,transparent)] text-[var(--ec-accent)]'
            : 'text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)]'
        "
        @click="runStatus = item.id"
      >
        {{ item.label }}
      </button>
      <select
        v-model="runTaskId"
        class="h-7 rounded-md border border-[var(--ec-border)] bg-transparent px-2 text-xs"
        aria-label="按任务筛选"
      >
        <option value="all">全部任务</option>
        <option v-for="item in tasks" :key="item.id" :value="item.id">{{ item.name }}</option>
      </select>
      <label class="flex items-center gap-2 text-xs text-[var(--ec-fg-muted)]">
        <input v-model="showArchived" type="checkbox" class="h-4 w-4" />
        显示已归档
      </label>
    </div>

    <div class="min-h-0 flex-1 overflow-auto px-5 py-4">
      <p v-if="loadError" class="text-sm text-[var(--ec-danger)]">{{ loadError }}</p>

      <template v-else-if="tab === 'tasks'">
        <div v-if="tasks.length === 0" class="mx-auto flex max-w-3xl flex-col gap-4 py-8">
          <div>
            <p class="text-sm">还没有定时任务</p>
            <p class="mt-1 text-xs text-[var(--ec-fg-muted)]">
              选一个场景开始，或从右上角新建。每条任务都要指定智能体。
            </p>
          </div>
          <div class="grid gap-3 sm:grid-cols-3">
            <button
              v-for="item in scheduleTemplates"
              :key="item.id"
              type="button"
              class="rounded-md border border-[var(--ec-border)] px-3 py-3 text-left hover:border-[var(--ec-accent)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_6%,transparent)]"
              @click="openTemplate(item)"
            >
              <span class="block text-sm">{{ item.name }}</span>
              <span class="mt-1 block text-xs leading-5 text-[var(--ec-fg-muted)]">{{
                item.description
              }}</span>
              <span class="mt-3 block text-xs text-[var(--ec-accent)]">{{
                describeRule(item.rule)
              }}</span>
            </button>
          </div>
        </div>
        <p
          v-else-if="shownTasks.length === 0"
          class="py-16 text-center text-sm text-[var(--ec-fg-muted)]"
        >
          没有符合条件的定时任务
        </p>
        <div v-else class="mx-auto flex max-w-3xl flex-col gap-2">
          <ScheduleCard
            v-for="item in shownTasks"
            :key="item.id"
            :task="item"
            :batch="batch"
            :checked="selected.includes(item.id)"
            @open="openTask(item)"
            @run="onRun(item)"
            @edit="openEdit(item)"
            @records="openRecords(item)"
            @remove="confirmRemove([item.id])"
            @toggle="onToggle(item, $event)"
            @check="toggleOne(item.id, $event)"
          />
        </div>
      </template>

      <template v-else>
        <p
          v-if="shownRuns.length === 0"
          class="py-16 text-center text-sm text-[var(--ec-fg-muted)]"
        >
          没有运行记录
        </p>
        <div v-else class="mx-auto flex max-w-3xl flex-col gap-2">
          <article
            v-for="item in shownRuns"
            :key="item.id"
            class="rounded-md border border-[var(--ec-border)] ec-skin-raised px-4 py-3"
          >
            <button
              type="button"
              class="flex w-full items-start gap-3 rounded-md text-left hover:bg-[color-mix(in_srgb,var(--ec-fg)_6%,transparent)]"
              @click="openRun(item)"
            >
              <span class="min-w-0 flex-1">
                <span class="flex items-center gap-2">
                  <span class="truncate text-sm">{{ item.taskName }}</span>
                  <span
                    class="shrink-0 text-xs"
                    :class="
                      item.status === 'failed'
                        ? 'text-[var(--ec-danger)]'
                        : item.status === 'success'
                          ? 'text-[var(--ec-success)]'
                          : item.status === 'waiting'
                            ? 'text-[var(--ec-attention)]'
                            : 'text-[var(--ec-accent)]'
                    "
                  >
                    {{ runStatusLabel(item.status) }}
                  </span>
                  <span v-if="item.archived" class="text-xs text-[var(--ec-fg-muted)]"
                    >已归档</span
                  >
                </span>
                <span class="mt-1 block text-xs text-[var(--ec-fg-muted)]">
                  {{ item.agentName }} · {{ formatDateTime(item.startedAt) }} ·
                  {{ item.status === 'running' ? '进行中' : formatDuration(item.durationMs) }} ·
                  {{ item.trigger === 'manual' ? '手动' : '计划' }}
                </span>
              </span>
            </button>
            <p
              v-if="item.summary"
              class="mt-2 line-clamp-3 text-xs leading-5 text-[var(--ec-fg-muted)]"
            >
              {{ item.summary }}
            </p>
            <div
              v-if="expandedRun === item.id"
              class="mt-2 rounded-md bg-[color-mix(in_srgb,var(--ec-danger)_10%,transparent)] px-3 py-2"
            >
              <p class="text-xs leading-5 text-[var(--ec-danger)]">
                {{ item.error || '执行失败' }}
              </p>
              <p class="mt-1 text-xs leading-5 text-[var(--ec-fg-muted)]">
                {{ failureHint(item.error) }}
              </p>
            </div>
            <div class="mt-2 flex justify-end gap-1">
              <button type="button" class="bar" @click="archiveRun(item, !item.archived)">
                {{ item.archived ? '取消归档' : '归档' }}
              </button>
              <button
                v-if="item.status !== 'running'"
                type="button"
                class="bar"
                @click="removeRun(item)"
              >
                删除
              </button>
            </div>
          </article>
        </div>
      </template>
    </div>

    <ScheduleEditor
      v-model:show="editorOpen"
      :saving="saving"
      :draft="editorDraft"
      :agents="agents"
      :models="models"
      @save="onSave"
    />
    <TemplatePicker v-model:show="templateOpen" @pick="openTemplate" />
  </div>
</template>

<style scoped>
.bar {
  border-radius: 0.375rem;
  padding: 0.25rem 0.5rem;
  font-size: var(--ec-text-meta);
  color: var(--ec-fg-muted);
}

.bar:hover:not(:disabled) {
  background: color-mix(in srgb, var(--ec-fg) 8%, transparent);
  color: var(--ec-fg);
}

.bar:disabled {
  opacity: 0.4;
}

input[type='checkbox'] {
  accent-color: var(--ec-accent);
}
</style>
