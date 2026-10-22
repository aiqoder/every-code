<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '../../stores/session'
import { usePageSurface } from '../../surfaces/host'
import { showResult } from '../../surfaces/registry'
import {
  executePage,
  executeSurface,
  refreshSelected,
  selectTask as runSelectTask,
  startTask as runStartTask,
  stopExecuteWatch
} from '../../surfaces/execute'
import ExecuteChat from './components/ExecuteChat.vue'
import EmptyState from '../../components/EmptyState.vue'
import Dialog from '../../components/Dialog.vue'
import type { Task } from '../../../../shared/types'
import { taskDisplayName } from '../../utils/taskTitle'

const session = useSessionStore()
const router = useRouter()

const selectedId = computed(() => executePage.selectedId)
const busy = computed(() => executePage.busy)
const detailOpen = ref(false)
usePageSurface(executeSurface)

const tasks = computed(() => session.allTasks)

const visibleTasks = computed(() =>
  [...tasks.value].sort((a, b) => b.createdAt.localeCompare(a.createdAt))
)

const selected = computed(() => tasks.value.find((t) => t.id === selectedId.value) ?? null)

const progress = computed(() => progressOf(selected.value))

const doneSteps = computed(() => tasks.value.reduce((sum, task) => sum + task.doneCount, 0))
const totalSteps = computed(() => tasks.value.reduce((sum, task) => sum + task.totalCount, 0))

function progressOf(task: { doneCount: number; totalCount: number } | null | undefined): number {
  if (!task?.totalCount) return 0
  return Math.round((task.doneCount / task.totalCount) * 100)
}

let pollTimer: ReturnType<typeof setInterval> | null = null

async function selectTask(task: Task): Promise<void> {
  await runSelectTask(task)
}

async function startTask(task: Task): Promise<void> {
  showResult(await runStartTask(task))
}

async function viewPlan(task: Task): Promise<void> {
  await selectTask(task)
  detailOpen.value = true
}

function canStart(task: Task): boolean {
  return task.totalCount > 0 && task.doneCount < task.totalCount
}

onMounted(async () => {
  if (!session.projects.length) await session.loadProjects()
  await session.loadAllTasks()
  const saved = Number(localStorage.getItem('ade:task') ?? 0)
  const candidate = tasks.value.find((t) => t.id === saved) ?? tasks.value[0]
  if (candidate) await selectTask(candidate)
  pollTimer = setInterval(() => {
    void session.loadAllTasks().then(() => {
      const current = selected.value
      if (current && current.doneCount < current.totalCount) void refreshSelected()
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
          <span class="stat"><b>{{ doneSteps }}/{{ totalSteps }}</b> 待办完成</span>
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
                <th class="col-progress">进度</th>
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
                  <span class="tr-name ellipsis">{{ taskDisplayName(task) }}</span>
                </td>
                <td class="col-progress">
                  <span class="row-progress">
                    <span class="bar"><span class="fill" :style="{ width: progressOf(task) + '%' }" /></span>
                    <span>{{ task.doneCount }}/{{ task.totalCount || '—' }}</span>
                  </span>
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
                      v-else-if="!task.totalCount"
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
          <div class="spacer" />
          <button
            v-if="canStart(selected)"
            class="btn primary sm"
            :disabled="busy"
            @click="startTask(selected)"
          >
            开始执行
          </button>
          <button class="btn sm" @click="router.push(`/tasks/${selected.id}`)">打开计划页</button>
        </div>

        <p v-if="selected.overview" class="dt-overview">{{ selected.overview }}</p>

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
.col-progress {
  width: 160px;
  white-space: nowrap;
  padding-right: 8px;
}
.row-progress {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-dim);
}
.row-progress .bar {
  width: 72px;
  height: 3px;
  border-radius: 2px;
  background: var(--ink-700);
  overflow: hidden;
}
.row-progress .fill {
  display: block;
  height: 100%;
  background: var(--brass);
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
.dt-overview {
  margin: 12px 0 0;
  font-size: 12px;
  line-height: 1.75;
  color: var(--text-dim);
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
