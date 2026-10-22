<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '../../stores/session'
import { toast } from '../../composables/toast'
import { usePageSurface } from '../../surfaces/host'
import { showResult } from '../../surfaces/registry'
import {
  executePage,
  executeSurface,
  focusTerminal,
  refreshSelected,
  retryTask as runRetryTask,
  selectTask as runSelectTask,
  startTask as runStartTask,
  stopExecuteWatch
} from '../../surfaces/execute'
import ExecuteChat from './components/ExecuteChat.vue'
import StatusPill from '../../components/StatusPill.vue'
import EmptyState from '../../components/EmptyState.vue'
import Dialog from '../../components/Dialog.vue'
import type { Task } from '../../../../shared/types'

const session = useSessionStore()
const router = useRouter()

const selectedId = computed(() => executePage.selectedId)
const busy = computed(() => executePage.busy)
const detailOpen = ref(false)
const failureOpen = computed({
  get: () => executePage.failureOpen,
  set: (value) => {
    executePage.failureOpen = value
  }
})
usePageSurface(executeSurface)

const tasks = computed(() => session.allTasks)

const visibleTasks = computed(() =>
  [...tasks.value].sort((a, b) => b.createdAt.localeCompare(a.createdAt))
)

const selected = computed(() => tasks.value.find((t) => t.id === selectedId.value) ?? null)

const progress = computed(() => {
  const t = selected.value
  if (!t || !t.totalCount) return 0
  return Math.round((t.doneCount / t.totalCount) * 100)
})

const runningCount = computed(() => tasks.value.filter((t) => t.status === 'running').length)

const KIND_LABEL: Record<string, string> = {
  planner: '拆解',
  advisor: '顾问',
  coder: '编码',
  verifier: '验收',
  reviewer: '复核',
  installer: '安装',
  deploy: '部署',
  task: '任务'
}

const failure = computed(() => {
  const t = selected.value
  if (!t?.lastError?.message) return null
  return {
    kind: KIND_LABEL[t.lastError.kind] ?? t.lastError.kind,
    message: t.lastError.message,
    at: t.lastError.at
  }
})

const failureLong = computed(() => (failure.value?.message.length ?? 0) > 160)
const failureText = computed(() => {
  const msg = failure.value?.message ?? ''
  if (failureOpen.value || msg.length <= 160) return msg
  return `${msg.slice(0, 160)}…`
})

let pollTimer: ReturnType<typeof setInterval> | null = null

async function selectTask(task: Task): Promise<void> {
  await runSelectTask(task)
}

async function openTaskTab(task: Task): Promise<void> {
  showResult(await focusTerminal(task))
}

async function startTask(task: Task): Promise<void> {
  showResult(await runStartTask(task))
}

async function retryTask(task: Task): Promise<void> {
  showResult(await runRetryTask(task))
}

async function viewPlan(task: Task): Promise<void> {
  await selectTask(task)
  detailOpen.value = true
}

function copyFailure(): void {
  const msg = failure.value?.message
  if (!msg) return
  void navigator.clipboard.writeText(msg).then(
    () => toast.success('报错已复制'),
    () => toast.error('复制失败')
  )
}

function canStart(task: Task): boolean {
  return task.status === 'pending' && !!task.planned
}

function canRetry(task: Task): boolean {
  return task.status === 'failed'
}

onMounted(async () => {
  if (!session.projects.length) await session.loadProjects()
  await session.loadAllTasks()
  const saved = Number(localStorage.getItem('ade:task') ?? 0)
  const running = tasks.value.find((t) => t.status === 'running')
  const candidate =
    running ?? tasks.value.find((t) => t.id === saved) ?? tasks.value[0]
  if (candidate) await selectTask(candidate)
  pollTimer = setInterval(() => {
    void session.loadAllTasks().then(() => {
      if (selected.value?.status === 'running') void refreshSelected()
    })
  }, 6000)
})

onBeforeUnmount(() => {
  stopExecuteWatch()
  if (pollTimer) clearInterval(pollTimer)
})

watch(
  () => tasks.value.length,
  () => {
    if (!selectedId.value && tasks.value.length) void selectTask(tasks.value[0])
  }
)
</script>

<template>
  <div class="page-with-chat">
    <div class="execute">
      <header class="exec-head">
        <div class="col">
          <span class="eyebrow">执行</span>
          <h1 class="serif">任务列表</h1>
        </div>
        <div class="stats">
          <span class="stat"><b>{{ tasks.length }}</b> 任务</span>
          <span class="stat" :class="{ live: runningCount > 0 }"><b>{{ runningCount }}</b> 执行中</span>
          <span class="stat"><b>{{ tasks.filter((t) => t.status === 'wait_test').length }}</b> 待测试</span>
        </div>
      </header>

      <div class="exec-body">
        <EmptyState
          v-if="!visibleTasks.length"
          empty-title="没有任务"
          empty-hint="任务页拆出任务后，会按创建时间出现在这里。"
        />
        <div v-else class="table-wrap">
          <table class="task-table">
            <thead>
              <tr>
                <th>标题</th>
                <th class="col-status">状态</th>
                <th class="col-ops">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="task in visibleTasks"
                :key="task.id"
                class="task-row"
                :class="{ active: task.id === selectedId }"
                @click="selectTask(task)"
              >
                <td>
                  <span class="tr-name ellipsis">{{ task.name || '拆解中…' }}</span>
                </td>
                <td class="col-status">
                  <StatusPill :status="task.status" size="sm" />
                </td>
                <td class="col-ops">
                  <div class="ops">
                    <button
                      v-if="canStart(task)"
                      class="btn primary sm ops-run"
                      :disabled="busy"
                      @click.stop="startTask(task)"
                    >
                      开始执行
                    </button>
                    <button
                      v-else-if="canRetry(task)"
                      class="btn primary sm ops-run"
                      :disabled="busy"
                      @click.stop="retryTask(task)"
                    >
                      重新执行
                    </button>
                    <button
                      v-else-if="task.status === 'pending'"
                      class="btn sm ops-run"
                      disabled
                      title="拆解完成后再开始"
                    >
                      开始执行
                    </button>
                    <span v-else class="ops-run ops-empty" aria-hidden="true" />
                    <button class="btn sm" @click.stop="viewPlan(task)">查看计划</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <Dialog
      :open="detailOpen"
      :title="selected?.name || '任务详情'"
      width="640px"
      @update:open="detailOpen = $event"
      @close="detailOpen = false"
    >
      <template v-if="selected">
        <div class="dt-top">
          <StatusPill :status="selected.status" />
          <div class="spacer" />
          <button
            v-if="canStart(selected)"
            class="btn primary sm"
            :disabled="busy"
            @click="startTask(selected)"
          >
            开始执行
          </button>
          <button
            v-else-if="canRetry(selected)"
            class="btn primary sm"
            :disabled="busy"
            @click="retryTask(selected)"
          >
            重新执行
          </button>
          <button class="btn sm" @click="router.push(`/tasks/${selected.id}`)">打开计划页</button>
          <button v-if="selected.worktree" class="btn ghost sm" @click="openTaskTab(selected)">
            聚焦终端
          </button>
        </div>

        <div class="dt-meta">
          <span class="dm"><span class="k">分支</span><span class="mono ellipsis">{{ selected.branch || '—' }}</span></span>
          <span class="dm"><span class="k">worktree</span><span class="mono ellipsis" :title="selected.worktree">{{ selected.worktree || '开始执行时创建' }}</span></span>
          <span class="dm"><span class="k">合入</span><span>dev {{ selected.mergedDev ? '✓' : '—' }} · main {{ selected.mergedMain ? '✓' : '—' }}</span></span>
        </div>

        <p v-if="selected.overview" class="dt-overview">{{ selected.overview }}</p>

        <div v-if="selected.status === 'failed' && failure" class="dt-fail">
          <div class="df-head">
            <span class="df-dot" />
            <span class="df-title">执行出错 · {{ failure.kind }}</span>
            <span class="spacer" />
            <span class="df-time">{{ failure.at }}</span>
          </div>
          <pre class="df-msg selectable">{{ failureText }}</pre>
          <div class="df-foot">
            <button v-if="failureLong" class="btn ghost sm" @click="failureOpen = !failureOpen">
              {{ failureOpen ? '收起' : '展开完整报错' }}
            </button>
            <button class="btn ghost sm" @click="copyFailure">复制</button>
            <div class="spacer" />
            <span class="df-hint">修好后点「重新执行」</span>
          </div>
        </div>

        <div class="dt-progress">
          <div class="row">
            <h3 class="serif">待办进度</h3>
            <span class="faint">{{ selected.doneCount }}/{{ selected.totalCount }}</span>
            <div class="spacer" />
            <span class="pct">{{ progress }}%</span>
          </div>
          <div class="pbar"><span class="pfill" :style="{ width: progress + '%' }" /></div>
        </div>

        <div class="todo-list">
          <div v-for="td in selected.todos" :key="td.id" class="todo" :class="td.status">
            <span class="todo-mark">{{
              td.status === 'done' ? '✓' : td.status === 'doing' ? '·' : ''
            }}</span>
            <div class="col todo-text">
              <span class="todo-title">{{ td.title }}</span>
              <span v-if="td.acceptance" class="todo-acc">验收：{{ td.acceptance }}</span>
            </div>
            <span v-if="td.source === 'agent'" class="lock" title="Agent 生成，不可修改">锁定</span>
          </div>
          <p v-if="!selected.todos.length" class="col-empty">还没有待办，拆解完成后再看</p>
        </div>
      </template>
    </Dialog>

    <ExecuteChat
      v-if="session.currentProjectId"
      :project-id="session.currentProjectId"
      :task-id="selectedId"
      @settled="refreshSelected"
    />
  </div>
</template>

<style scoped>
.execute {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.exec-head {
  display: flex;
  align-items: flex-end;
  gap: 22px;
  padding: 22px 26px 14px;
  border-bottom: 1px solid var(--line-soft);
  -webkit-app-region: drag;
  flex-wrap: wrap;
}
.exec-head .col,
.exec-head .stats {
  -webkit-app-region: no-drag;
}
.eyebrow {
  font-size: 10.5px;
  letter-spacing: 0.18em;
  color: var(--brass-dim);
  text-transform: uppercase;
  margin-bottom: 5px;
}
h1 {
  font-size: 21px;
}
.stats {
  display: flex;
  gap: 16px;
  padding-bottom: 3px;
}
.stat {
  font-size: 11.5px;
  color: var(--text-faint);
}
.stat b {
  font-family: var(--font-mono);
  color: var(--text-dim);
  font-weight: 500;
  margin-right: 3px;
}
.stat.live b {
  color: var(--brass);
}

.exec-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 14px 22px 22px;
}

.table-wrap {
  border: 1px solid var(--line);
  border-radius: var(--r-md);
  background: var(--paper-900);
  overflow: hidden;
}
.task-table {
  width: 100%;
  border-collapse: collapse;
}
.task-table th,
.task-table td {
  padding: 11px 16px;
  text-align: left;
  vertical-align: middle;
  border-bottom: 1px solid var(--line-soft);
}
.task-table th {
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.04em;
  color: var(--text-faint);
  background: var(--ink-850);
}
.task-table tbody tr:last-child td {
  border-bottom: none;
}
.task-table tbody tr.task-row {
  cursor: pointer;
}
.task-table tbody tr:hover td {
  background: var(--paper-800);
}
.task-table tbody tr.active td {
  background: rgba(200, 160, 90, 0.06);
}
.col-status {
  width: 1%;
  white-space: nowrap;
  padding-right: 8px;
}
.col-ops {
  width: 1%;
  white-space: nowrap;
  padding-left: 8px;
}
.ops {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
.ops-run {
  min-width: 76px;
  justify-content: center;
}
.ops-empty {
  display: inline-block;
  height: 1px;
}
.tr-name {
  display: block;
  min-width: 0;
  font-size: 13px;
  color: var(--text-strong);
  line-height: 1.45;
}

.dt-top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}
.dt-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  border: 1px solid var(--line-soft);
  border-radius: var(--r-sm);
  background: var(--ink-850);
}
.dm {
  display: flex;
  gap: 9px;
  font-size: 11px;
  color: var(--text-dim);
  min-width: 0;
}
.dm .k {
  width: 54px;
  flex: none;
  color: var(--text-faint);
  font-size: 10px;
  letter-spacing: 0.06em;
}
.dt-overview {
  margin: 12px 0 0;
  font-size: 12px;
  line-height: 1.75;
  color: var(--text-dim);
}
.dt-fail {
  margin-top: 12px;
  border: 1px solid var(--rust-dim);
  border-left: 2px solid var(--rust);
  border-radius: var(--r-md);
  background: rgba(180, 91, 74, 0.07);
  padding: 9px 11px 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.df-head {
  display: flex;
  align-items: center;
  gap: 7px;
}
.df-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--rust);
  flex: none;
}
.df-title {
  font-size: 11.5px;
  color: var(--rust);
  letter-spacing: 0.04em;
}
.df-time {
  font-family: var(--font-mono);
  font-size: 9.5px;
  color: var(--text-faint);
}
.df-msg {
  font-family: var(--font-mono);
  font-size: 10.5px;
  line-height: 1.65;
  color: var(--text);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 132px;
  overflow-y: auto;
  margin: 0;
}
.df-foot {
  display: flex;
  align-items: center;
  gap: 6px;
}
.df-hint {
  font-size: 10px;
  color: var(--text-faint);
}
.dt-progress {
  margin-top: 14px;
  display: flex;
  flex-direction: column;
  gap: 7px;
}
.dt-progress .row {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.dt-progress h3 {
  font-size: 12.5px;
  margin: 0;
}
.dt-progress .faint {
  font-size: 10.5px;
  color: var(--text-faint);
}
.pct {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--brass);
}
.pbar {
  height: 3px;
  border-radius: 2px;
  background: var(--ink-700);
  overflow: hidden;
}
.pfill {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, var(--brass-dim), var(--brass));
  transition: width 0.35s ease;
}
.todo-list {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.todo {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--r-sm);
  border-left: 2px solid var(--line);
}
.todo.done {
  border-left-color: var(--moss-dim);
  opacity: 0.7;
}
.todo.doing {
  border-left-color: var(--brass);
  background: rgba(200, 160, 90, 0.06);
}
.todo-mark {
  width: 12px;
  flex: none;
  font-size: 10px;
  color: var(--moss);
  line-height: 1.6;
  text-align: center;
}
.todo.doing .todo-mark {
  color: var(--brass);
}
.todo-text {
  gap: 2px;
  min-width: 0;
  flex: 1;
}
.todo-title {
  font-size: 11.5px;
  color: var(--text);
  line-height: 1.55;
}
.todo.done .todo-title {
  text-decoration: line-through;
  text-decoration-color: var(--moss-dim);
}
.todo-acc {
  font-size: 10px;
  color: var(--text-faint);
  line-height: 1.5;
}
.lock {
  font-size: 9px;
  color: var(--brass-dim);
  border: 1px solid var(--brass-dim);
  border-radius: 3px;
  padding: 0 3px;
  flex: none;
  margin-top: 1px;
}
.col-empty {
  font-size: 11px;
  color: var(--text-faint);
  padding: 12px 6px;
  line-height: 1.6;
}
</style>
