<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NSwitch, useDialog, useMessage } from 'naive-ui'
import { listAgents, listModels, type AgentItem, type ModelItem } from '@/api/catalog'
import { useSessionStore } from '@/stores/sessionStore'
import {
  describeRule,
  failureHint,
  formatDateTime,
  formatDuration,
  permissionLabel,
  runStatusLabel,
  validityLabel,
  type ScheduleDraft,
  type ScheduleRun,
  type ScheduledTask
} from '@shared/schedule'
import ScheduleEditor from './components/ScheduleEditor.vue'
import { onScheduleChanged, runScheduledTask } from './runner'

const session = useSessionStore()
const route = useRoute()
const router = useRouter()
const dialog = useDialog()
const message = useMessage()

const tasks = ref<ScheduledTask[]>([])
const runs = ref<ScheduleRun[]>([])
const agents = ref<AgentItem[]>([])
const models = ref<ModelItem[]>([])
const editorOpen = ref(false)
const editorDraft = ref<ScheduleDraft | null>(null)
const saving = ref(false)
const scroller = ref<HTMLElement | null>(null)

const taskId = computed(() => String(route.params.id || ''))
const task = computed(() => tasks.value.find((item) => item.id === taskId.value) || null)
const timeline = computed(() =>
  runs.value
    .filter((item) => item.taskId === taskId.value)
    .slice()
    .sort((a, b) => a.startedAt.localeCompare(b.startedAt))
)

let stopWatch: () => void = () => {}

onMounted(() => {
  void boot()
  stopWatch = onScheduleChanged(() => {
    void reload()
  })
})

onUnmounted(() => stopWatch())

watch(timeline, () => {
  void stick()
})

watch(
  () => session.loggedIn,
  (loggedIn, was) => {
    if (loggedIn && !was) void loadCatalog()
  }
)

async function boot(): Promise<void> {
  await reload()
  await loadCatalog()
  await stick()
}

async function loadCatalog(): Promise<void> {
  if (!session.loggedIn) return
  const [agentList, modelList] = await Promise.all([
    listAgents().catch(() => []),
    listModels().catch(() => [])
  ])
  agents.value = agentList
  models.value = modelList
}

async function reload(): Promise<void> {
  if (!window.api?.schedule) return
  const state = await window.api.schedule.state()
  tasks.value = state.tasks
  runs.value = state.runs
}

async function stick(): Promise<void> {
  await nextTick()
  if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
}

function back(): void {
  void router.push({ name: 'schedules' })
}

function openRecords(): void {
  void router.push({ name: 'schedules', query: { tab: 'runs', task: taskId.value } })
}

async function onRun(): Promise<void> {
  if (!task.value) return
  try {
    await runScheduledTask(task.value.id, task.value.permission === 'default')
  } catch (err) {
    message.error(err instanceof Error ? err.message : '运行失败')
  }
}

async function onToggle(enabled: boolean): Promise<void> {
  if (!task.value || !window.api?.schedule) return
  try {
    await window.api.schedule.setEnabled(task.value.id, enabled)
    await reload()
  } catch (err) {
    message.error(err instanceof Error ? err.message : '切换失败')
  }
}

function openEdit(): void {
  if (!task.value) return
  const current = task.value
  editorDraft.value = {
    id: current.id,
    name: current.name,
    prompt: current.prompt,
    agentId: current.agentId,
    agentName: current.agentName,
    modelId: current.modelId,
    workspace: current.workspace,
    permission: current.permission,
    rule: structuredClone(current.rule),
    enabled: current.enabled
  }
  editorOpen.value = true
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

function confirmRemove(): void {
  if (!task.value) return
  const id = task.value.id
  dialog.warning({
    title: '删除定时任务',
    content: '删除后任务不再执行，此操作无法撤销。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      if (!window.api?.schedule) return
      await window.api.schedule.remove([id])
      back()
    }
  })
}

function openRun(run: ScheduleRun): void {
  if (!run.conversationId) return
  void router.push({
    name: 'chat',
    query: {
      conversation: String(run.conversationId),
      ...(run.status === 'waiting' && task.value ? { draft: task.value.prompt } : {})
    }
  })
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col text-[var(--ec-fg)]">
    <header class="flex shrink-0 items-start gap-3 border-b border-[var(--ec-border)] px-5 py-3">
      <button type="button" class="mt-0.5 text-sm text-[var(--ec-fg-muted)]" @click="back">
        返回
      </button>
      <div v-if="task" class="min-w-0 flex-1">
        <p class="truncate text-sm">{{ task.name }}</p>
        <p class="mt-1 truncate text-xs text-[var(--ec-fg-muted)]">
          {{ task.agentName || '未命名智能体' }} · {{ describeRule(task.rule) }} ·
          {{ permissionLabel(task.permission) }}
        </p>
        <p class="mt-0.5 text-xs text-[var(--ec-fg-muted)]">
          <template v-if="validityLabel(task.rule)">{{ validityLabel(task.rule) }} · </template>
          <template v-if="!task.enabled">已暂停</template>
          <template v-else-if="task.nextRunAt"
            >下次运行 {{ formatDateTime(task.nextRunAt) }}</template
          >
          <template v-else>没有后续执行</template>
        </p>
      </div>
      <div v-if="task" class="flex shrink-0 items-center gap-1">
        <button type="button" class="bar" :disabled="task.lastStatus === 'running'" @click="onRun">
          运行
        </button>
        <button type="button" class="bar" @click="openEdit">编辑</button>
        <button type="button" class="bar" @click="openRecords">运行记录</button>
        <button type="button" class="bar" @click="confirmRemove">删除</button>
        <n-switch size="small" :value="task.enabled" @update:value="onToggle" />
      </div>
    </header>

    <div ref="scroller" class="min-h-0 flex-1 overflow-auto px-5 py-4">
      <p v-if="!task" class="py-16 text-center text-sm text-[var(--ec-fg-muted)]">
        这个定时任务已经删除了
      </p>
      <p
        v-else-if="timeline.length === 0"
        class="py-16 text-center text-sm text-[var(--ec-fg-muted)]"
      >
        还没有运行。可以先运行一次，确认结果再放到计划里。
      </p>
      <ol v-else class="mx-auto max-w-3xl">
        <li v-for="item in timeline" :key="item.id" class="relative pb-6 pl-6">
          <span
            class="absolute top-1.5 left-0 h-2.5 w-2.5 rounded-full"
            :class="
              item.status === 'failed'
                ? 'bg-[var(--ec-danger)]'
                : item.status === 'success'
                  ? 'bg-[var(--ec-success)]'
                  : item.status === 'waiting'
                    ? 'bg-[var(--ec-attention)]'
                    : 'bg-[var(--ec-accent)]'
            "
          />
          <span
            class="rail absolute top-4 left-[4px] h-[calc(100%-0.5rem)] w-px bg-[var(--ec-border)]"
          />
          <button type="button" class="w-full text-left" @click="openRun(item)">
            <span class="flex items-baseline gap-2">
              <span class="text-sm">{{ runStatusLabel(item.status) }}</span>
              <span class="text-xs text-[var(--ec-fg-muted)]">{{
                formatDateTime(item.startedAt)
              }}</span>
              <span class="text-xs text-[var(--ec-fg-muted)]">
                {{ item.status === 'running' ? '进行中' : formatDuration(item.durationMs) }}
              </span>
              <span class="text-xs text-[var(--ec-fg-muted)]">{{
                item.trigger === 'manual' ? '手动' : '计划'
              }}</span>
              <span v-if="item.archived" class="text-xs text-[var(--ec-fg-muted)]">已归档</span>
            </span>
            <span v-if="item.summary" class="mt-1 block text-sm leading-6 whitespace-pre-wrap">{{
              item.summary
            }}</span>
            <span v-if="item.error" class="mt-1 block text-xs leading-5 text-[var(--ec-danger)]">{{
              item.error
            }}</span>
            <span v-if="item.error" class="mt-1 block text-xs leading-5 text-[var(--ec-fg-muted)]">
              {{ failureHint(item.error) }}
            </span>
          </button>
        </li>
      </ol>
    </div>

    <ScheduleEditor
      v-model:show="editorOpen"
      :saving="saving"
      :draft="editorDraft"
      :agents="agents"
      :models="models"
      @save="onSave"
    />
  </div>
</template>

<style scoped>
li:last-child .rail {
  display: none;
}

.bar {
  border-radius: 0.375rem;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  color: var(--ec-fg-muted);
}

.bar:hover:not(:disabled) {
  background: color-mix(in srgb, var(--ec-fg) 8%, transparent);
  color: var(--ec-fg);
}

.bar:disabled {
  opacity: 0.4;
}
</style>
