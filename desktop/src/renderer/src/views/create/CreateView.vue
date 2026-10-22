<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '../../stores/session'
import { usePageSurface } from '../../surfaces/host'
import { showResult } from '../../surfaces/registry'
import { createSurface, openTaskPage, saveTaskDescription } from '../../surfaces/create'
import Dialog from '../../components/Dialog.vue'
import FormField from '../../components/FormField.vue'
import TextArea from '../../components/TextArea.vue'
import EmptyState from '../../components/EmptyState.vue'
import StatusPill from '../../components/StatusPill.vue'
import type { Task } from '../../../../shared/types'
import { clickOnSelection } from '../../utils/selection'
import { taskDisplayName } from '../../utils/taskTitle'

const session = useSessionStore()
const router = useRouter()

usePageSurface(createSurface)

const taskOpen = ref(false)
const taskBusy = ref(false)
const taskError = ref('')
const taskDescription = ref('')

const projectTasks = computed<Task[]>(() => {
  const id = session.currentProjectId
  return id ? (session.tasks[id] ?? []) : []
})

onMounted(async () => {
  if (session.currentProjectId) await session.loadTasks(session.currentProjectId)
})

watch(
  () => session.currentProjectId,
  async (id) => {
    if (id) await session.loadTasks(id)
  }
)

function openTaskForm(): void {
  if (!session.currentProjectId) return
  taskError.value = ''
  taskDescription.value = ''
  taskOpen.value = true
}

async function saveTask(): Promise<void> {
  taskError.value = ''
  taskBusy.value = true
  try {
    const result = await saveTaskDescription(taskDescription.value)
    if (result.error) {
      taskError.value = result.error
      return
    }
    taskOpen.value = false
    showResult(result)
  } finally {
    taskBusy.value = false
  }
}

function openTask(task: Task, event: MouseEvent): void {
  if (clickOnSelection(event)) return
  void openTaskPage(task.id)
}

function progressOf(task: { doneCount: number; totalCount: number }): number {
  if (!task.totalCount) return 0
  return Math.round((task.doneCount / task.totalCount) * 100)
}
</script>

<template>
  <div class="create">
    <header class="page-head">
      <div class="col">
        <span class="eyebrow">任务</span>
        <h1 class="serif">{{ session.currentProject?.name || '创建与修改任务' }}</h1>
      </div>
      <div class="head-actions">
        <button v-if="session.currentProject" class="btn primary sm" @click="openTaskForm">＋ 添加任务</button>
      </div>
    </header>

    <div v-if="!session.currentProject" class="scroll">
      <EmptyState empty-title="还没有项目" empty-hint="先到「项目概况」新建一个项目，再回来添加任务。">
        <button class="btn primary" @click="router.push('/project')">去项目概况</button>
      </EmptyState>
    </div>

    <div v-else class="scroll">
      <section class="tasks">
        <div class="tasks-head">
          <h2 class="serif">本项目任务</h2>
          <span class="faint">{{ projectTasks.length }} 条</span>
          <div class="spacer" />
          <button class="btn sm" @click="openTaskForm">＋ 添加任务</button>
        </div>

        <EmptyState
          v-if="!projectTasks.length"
          empty-title="还没有任务"
          empty-hint="点「添加任务」写下要做什么；点进一条任务可修改需求与计划。"
        />

        <div v-else class="task-list">
          <button v-for="task in projectTasks" :key="task.id" class="task-card" @click="openTask(task, $event)">
            <div class="tc-head">
              <span class="tc-name serif ellipsis">{{ taskDisplayName(task) }}</span>
              <StatusPill :status="task.status" size="sm" />
            </div>
            <p class="tc-overview">{{ task.overview || task.rawRequest }}</p>
            <div class="tc-foot">
              <span class="tc-progress">
                <span class="bar"><span class="fill" :style="{ width: progressOf(task) + '%' }" /></span>
                {{ task.doneCount }}/{{ task.totalCount || '—' }}
              </span>
              <span class="tc-time">{{ task.updatedAt }}</span>
            </div>
          </button>
        </div>
      </section>
    </div>

    <Dialog
      :open="taskOpen"
      title="添加任务"
      subtitle="写下要做什么。保存后进入详情页，可继续改需求和计划。"
      width="640px"
      :persistent="taskBusy"
      @close="taskOpen = false"
    >
      <div class="form">
        <FormField label="任务描述">
          <TextArea
            v-model="taskDescription"
            :rows="8"
            placeholder="写清要做什么、给谁用、有哪些约束。"
          />
        </FormField>
      </div>
      <p v-if="taskError" class="err">{{ taskError }}</p>
      <template #footer>
        <button class="btn ghost" type="button" :disabled="taskBusy" @click="taskOpen = false">取消</button>
        <div class="spacer" />
        <button class="btn primary" type="button" :disabled="taskBusy" @click="saveTask">
          {{ taskBusy ? '保存中…' : '保存' }}
        </button>
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
.create {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.page-head {
  display: flex;
  align-items: flex-start;
  gap: 24px;
  padding: 26px 32px 18px;
  border-bottom: 1px solid var(--line-soft);
  -webkit-app-region: drag;
}
.page-head .col {
  -webkit-app-region: no-drag;
}
.eyebrow {
  font-size: 10.5px;
  letter-spacing: 0.18em;
  color: var(--brass-dim);
  text-transform: uppercase;
  margin-bottom: 6px;
}
h1 {
  font-size: 24px;
}
.head-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-left: auto;
  padding-top: 18px;
  -webkit-app-region: no-drag;
}
.scroll {
  flex: 1;
  overflow: auto;
  padding: 22px 32px 40px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.tasks {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tasks-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
}
.tasks-head h2 {
  font-size: 16px;
}
.tasks-head .faint {
  font-size: 11.5px;
}
.task-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;
}
.task-card {
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding: 14px 16px;
  text-align: left;
  user-select: text;
  background: var(--paper-900);
  border: 1px solid var(--line-soft);
  border-radius: var(--r-lg);
  transition: border-color 0.14s ease, background 0.14s ease, transform 0.14s ease;
}
.task-card:hover {
  border-color: var(--line-strong);
  background: var(--paper-800);
  transform: translateY(-1px);
}
.tc-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.tc-name {
  flex: 1;
  font-size: 14px;
  color: var(--text-strong);
}
.tc-overview {
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-dim);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.tc-foot {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 10.5px;
  color: var(--text-faint);
}
.tc-progress {
  display: flex;
  align-items: center;
  gap: 7px;
  font-family: var(--font-mono);
}
.bar {
  width: 62px;
  height: 3px;
  border-radius: 2px;
  background: var(--ink-700);
  overflow: hidden;
}
.fill {
  display: block;
  height: 100%;
  background: var(--brass);
  transition: width 0.3s ease;
}
.tc-time {
  margin-left: auto;
}
.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.err {
  margin-top: 12px;
  font-size: 12px;
  color: #d98d7d;
  padding: 7px 10px;
  border-radius: var(--r-sm);
  background: rgba(180, 91, 74, 0.12);
  border: 1px solid var(--rust-dim);
}
</style>
