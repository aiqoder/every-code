<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { adeApi, watchTaskEvents } from '../../api'
import { useSessionStore, messageOf } from '../../stores/session'
import { confirm } from '../../composables/confirm'
import { toast } from '../../composables/toast'
import { usePageSurface } from '../../surfaces/host'
import { showResult } from '../../surfaces/registry'
import {
  addTaskTodo,
  answerTaskQuestion,
  removeTaskTodo,
  taskPage,
  taskSurface,
  toggleTaskTodo,
  updateTaskDescription
} from '../../surfaces/task'
import { parseAskQuestionEvent, isAskTimeoutResult } from '../../composables/askQuestion'
import TaskChat, { type TaskChatHandle } from './components/TaskChat.vue'
import Dialog from '../../components/Dialog.vue'
import FormField from '../../components/FormField.vue'
import TextArea from '../../components/TextArea.vue'
import TextField from '../../components/TextField.vue'
import AskQuestionPanel from '../../components/AskQuestionPanel.vue'
import MarkdownBody from '../../components/MarkdownBody.vue'
import type { Task, TaskEvent } from '../../../../shared/types'
import { taskDisplayName } from '../../utils/taskTitle'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const railChat = ref<TaskChatHandle | null>(null)

const task = computed(() => taskPage.task)
const progress = computed(() => {
  const current = task.value
  if (!current?.totalCount) return 0
  return Math.round((current.doneCount / current.totalCount) * 100)
})
const loading = ref(true)
const busy = computed(() => taskPage.busy)
usePageSurface(taskSurface)
const pendingAsk = computed({
  get: () => taskPage.pendingAsk,
  set: (value) => {
    taskPage.pendingAsk = value
  }
})

const editOpen = ref(false)
const editText = ref('')
const editing = ref(false)

const addOpen = ref(false)
const newTodo = reactive({ title: '', detail: '' })

type TodoExtra = { detail?: string }
type TodoTip = {
  detail: string
  top: number
  left: number
  placed: boolean
}

const todoTip = ref<TodoTip | null>(null)
const tipPanel = ref<HTMLElement | null>(null)
let tipHideTimer: ReturnType<typeof setTimeout> | null = null
let tipToken = 0

function todoExtra(td: TodoExtra): Pick<TodoTip, 'detail'> {
  return {
    detail: td.detail?.trim() ?? ''
  }
}

function hasTodoExtra(td: TodoExtra): boolean {
  return Boolean(todoExtra(td).detail)
}

function showTodoTip(event: Event, td: TodoExtra): void {
  if (tipHideTimer) {
    clearTimeout(tipHideTimer)
    tipHideTimer = null
  }
  const token = ++tipToken
  const anchor = event.currentTarget as HTMLElement
  const rect = anchor.getBoundingClientRect()
  todoTip.value = { ...todoExtra(td), top: rect.bottom + 6, left: rect.left, placed: false }
  nextTick(() => {
    const panel = tipPanel.value
    const tip = todoTip.value
    if (token !== tipToken || !panel || !tip) return
    const box = panel.getBoundingClientRect()
    let top = rect.bottom + 6
    let left = rect.left
    if (top + box.height > window.innerHeight - 8) top = Math.max(8, rect.top - 6 - box.height)
    if (left + box.width > window.innerWidth - 8) left = Math.max(8, window.innerWidth - 8 - box.width)
    todoTip.value = { ...tip, top, left, placed: true }
  })
}

function queueHideTodoTip(): void {
  if (tipHideTimer) clearTimeout(tipHideTimer)
  tipHideTimer = setTimeout(() => {
    todoTip.value = null
    tipHideTimer = null
  }, 140)
}

function holdTodoTip(): void {
  if (tipHideTimer) {
    clearTimeout(tipHideTimer)
    tipHideTimer = null
  }
}

function hideTodoTipNow(): void {
  tipToken++
  if (tipHideTimer) {
    clearTimeout(tipHideTimer)
    tipHideTimer = null
  }
  todoTip.value = null
}

const taskId = computed(() => Number(route.params.id))

let unsub: (() => void) | null = null
let poll: ReturnType<typeof setInterval> | null = null

const chatStreaming = computed(() => {
  const chat = railChat.value
  if (!chat || chat.state.taskId !== taskId.value) return false
  return chat.state.streaming
})

const analyzing = ref(false)
const replanning = ref(false)
const deleting = ref(false)
let removed = false

function hasPlan(t: Task): boolean {
  return Boolean(t.name.trim() || t.todos.length)
}

/** 还没有计划时，用需求原文做第一次分析。 */
const showAnalyze = computed(() => {
  const t = task.value
  if (!t || t.doneCount > 0) return false
  return Boolean(t.rawRequest.trim()) && !hasPlan(t)
})

const canAnalyze = computed(() => {
  if (!showAnalyze.value) return false
  return !chatStreaming.value && !busy.value && !analyzing.value && !replanning.value
})

/** 待执行且已有计划内容时显示。 */
const showReplan = computed(() => {
  const t = task.value
  if (!t || t.doneCount > 0) return false
  return hasPlan(t)
})

const canReplan = computed(() => {
  if (!showReplan.value) return false
  const t = task.value
  if (!t) return false
  if (!t.rawRequest.trim()) return false
  return !chatStreaming.value && !busy.value && !analyzing.value && !replanning.value
})

watch(chatStreaming, (on, was) => {
  if (!was || on) return
  if (analyzing.value) analyzing.value = false
  if (replanning.value) replanning.value = false
  // 分析结束后刷新，保证名称 / 待办写回后「重新分析」能立刻出现
  void load()
})


async function load(): Promise<void> {
  if (removed) return
  loading.value = true
  try {
    const loaded = await adeApi.getTask(taskId.value)
    taskPage.task = loaded
    session.upsertTask(loaded)
  } catch (err) {
    toast.error(messageOf(err))
  } finally {
    loading.value = false
  }
}

function subscribe(): void {
  unsub?.()
  const watch = watchTaskEvents(taskId.value, (raw) => {
    const ev = raw as TaskEvent
    if (!ev || typeof ev !== 'object') return
    if (ev.taskId && ev.taskId !== taskId.value) return
    const ask = parseAskQuestionEvent(ev)
    if (ask) pendingAsk.value = ask
    if (ev.type === 'tool_call_end' && ev.toolName === 'AskQuestion') {
      if (pendingAsk.value && isAskTimeoutResult(ev.result)) {
        pendingAsk.value = { ...pendingAsk.value, status: 'timeout' }
      } else if (pendingAsk.value?.status === 'submitting' || pendingAsk.value?.status === 'pending') {
        pendingAsk.value = pendingAsk.value
          ? { ...pendingAsk.value, status: 'done' }
          : null
        setTimeout(() => {
          pendingAsk.value = null
        }, 800)
      }
    }
    if (['log', 'run_finished', 'error'].includes(ev.type)) void load()
  })
  unsub = watch.close
}

async function submitAsk(payload: {
  conversationId: number
  questionId: string
  answers: Array<{ questionId: string; selected: string[]; text?: string }>
}): Promise<void> {
  showResult(await answerTaskQuestion(payload))
}

function openEdit(): void {
  if (!task.value) return
  editText.value = task.value.rawRequest || task.value.description
  editOpen.value = true
}

async function submitEdit(): Promise<void> {
  editing.value = true
  try {
    const result = await updateTaskDescription(editText.value)
    showResult(result)
    if (!result.error) editOpen.value = false
  } finally {
    editing.value = false
  }
}

async function addTodo(): Promise<void> {
  const result = await addTaskTodo({
    title: newTodo.title,
    detail: newTodo.detail
  })
  showResult(result)
  if (result.error) return
  newTodo.title = ''
  newTodo.detail = ''
  addOpen.value = false
}

async function removeTodo(id: number): Promise<void> {
  showResult(await removeTaskTodo(id))
}

async function toggleTodo(id: number, status: string): Promise<void> {
  showResult(await toggleTaskTodo(id, status))
}

async function removeTask(): Promise<void> {
  const current = task.value
  if (!current || deleting.value) return
  const name = taskDisplayName(current)
  deleting.value = true
  try {
    await confirm({
      title: '删除任务',
      subtitle: '删除后不能恢复。',
      message: `删除「${name}」后，这个任务及其记录会一并删除。`,
      confirmText: '删除',
      danger: true,
      busyText: '删除中…',
      action: async () => {
        await adeApi.deleteTask(current.id)
        removed = true
        session.removeTask(current.id, current.projectId)
        taskPage.task = null
        toast.success('任务已删除')
        await router.push('/create')
      }
    })
  } finally {
    deleting.value = false
  }
}

async function runAnalysis(again: boolean): Promise<void> {
  const current = task.value
  const chat = railChat.value
  if ((again ? !canReplan.value : !canAnalyze.value) || !current) return
  if (!chat?.state.ready || chat.state.taskId !== current.id) {
    toast.error('右侧对话还没准备好')
    return
  }
  const text = current.rawRequest.trim()
  if (!text) {
    toast.error('没有需求原文')
    return
  }
  chat.fresh()
  if (again) replanning.value = true
  else analyzing.value = true
  try {
    const started = await chat.send(text, text.slice(0, 50), session.profile.modelId)
    if (!started) {
      if (again) replanning.value = false
      else analyzing.value = false
      toast.error(again ? '未能开始重新分析' : '未能开始分析')
    }
  } catch (err) {
    if (again) replanning.value = false
    else analyzing.value = false
    toast.error(messageOf(err))
  }
}

function analyze(): Promise<void> {
  return runAnalysis(false)
}

function replan(): Promise<void> {
  return runAnalysis(true)
}

onMounted(async () => {
  await load()
  subscribe()
  poll = setInterval(() => {
    const current = task.value
    if (current && (!hasPlan(current) || (current.totalCount > 0 && current.doneCount < current.totalCount))) void load()
  }, 5000)
})

onBeforeUnmount(() => {
  unsub?.()
  if (poll) clearInterval(poll)
  hideTodoTipNow()
})
</script>

<template>
  <div class="page-with-chat">
  <div class="plan">
    <header class="plan-head">
      <button class="btn ghost sm back" @click="router.back()">← 返回</button>
      <div class="spacer" />
      <button class="btn sm" @click="openEdit">修改任务</button>
      <button
        v-if="showAnalyze"
        class="btn sm"
        :disabled="!canAnalyze"
        @click="analyze"
      >
        {{ analyzing || chatStreaming ? '分析中…' : '任务分析' }}
      </button>
      <button
        v-if="showReplan"
        class="btn sm"
        :disabled="!canReplan"
        @click="replan"
      >
        {{ replanning || chatStreaming ? '分析中…' : '重新分析' }}
      </button>
      <button class="btn sm danger" :disabled="!task || deleting" @click="removeTask">
        删除任务
      </button>
    </header>

    <div v-if="loading && !task" class="loading">正在读取任务…</div>

    <div v-else-if="task" class="plan-body">
      <div class="main" @scroll="hideTodoTipNow">
        <div class="title-block">
          <span class="proj">{{ task.projectName }} · #{{ task.id }}</span>
          <h1 class="serif">{{ taskDisplayName(task) }}</h1>
          <div class="row title-actions">
            <span class="plan-progress">
              <span class="bar"><span class="fill" :style="{ width: progress + '%' }" /></span>
              <span>{{ task.doneCount }}/{{ task.totalCount }}</span>
            </span>
          </div>
        </div>

        <AskQuestionPanel :question="pendingAsk" @answer="submitAsk" />

        <section v-if="task.overview" class="block">
          <h3 class="serif block-title">概览</h3>
          <MarkdownBody :content="task.overview" />
        </section>

        <section class="block">
          <MarkdownBody class="description" :content="task.description" />
        </section>

        <section class="block">
          <div class="block-head">
            <h3 class="serif block-title">待办</h3>
            <span class="faint">{{ task.doneCount }}/{{ task.totalCount }}</span>
            <div class="spacer" />
            <button class="btn ghost sm" @click="addOpen = true">＋ 添加待办</button>
          </div>

          <div class="todos">
            <div v-for="(td, i) in task.todos" :key="td.id" class="todo" :class="td.status">
              <span class="seq">{{ String(i + 1).padStart(2, '0') }}</span>
              <div class="col body">
                <div class="row">
                  <span class="td-title">
                    <span class="td-title-text">{{ td.title }}</span>
                    <button
                      v-if="hasTodoExtra(td)"
                      type="button"
                      class="td-help"
                      aria-label="查看详情"
                      @mouseenter="showTodoTip($event, td)"
                      @mouseleave="queueHideTodoTip"
                      @focus="showTodoTip($event, td)"
                      @blur="queueHideTodoTip"
                    >
                      ?
                    </button>
                  </span>
                  <span v-if="td.source === 'agent'" class="lock" title="Agent 生成，不可修改或删除">锁定</span>
                  <span v-else class="owned">你的</span>
                </div>
              </div>
              <div v-if="td.source === 'user'" class="td-actions">
                <button class="btn ghost sm" @click="toggleTodo(td.id, td.status)">
                  {{ td.status === 'done' ? '取消完成' : '标记完成' }}
                </button>
                <button class="btn ghost sm" @click="removeTodo(td.id)">删除</button>
              </div>
              <span v-else-if="td.status === 'done'" class="done-mark">✓</span>
            </div>

            <p v-if="!task.todos.length" class="faint pad">还没有待办。</p>
          </div>
        </section>

      </div>
    </div>

    <Dialog
      :open="editOpen"
      title="修改需求"
      subtitle="只保存需求原文。待办保持原样，计划调整在右侧对话里完成。"
      width="620px"
      :persistent="editing"
      @close="editOpen = false"
    >
      <div class="form">
        <FormField label="需求描述">
          <TextArea v-model="editText" :rows="6" />
        </FormField>
      </div>
      <template #footer>
        <button class="btn ghost" @click="editOpen = false">取消</button>
        <div class="spacer" />
        <button class="btn primary" :disabled="editing" @click="submitEdit">
          {{ editing ? '保存中…' : '保存' }}
        </button>
      </template>
    </Dialog>

    <Dialog
      :open="addOpen"
      title="添加待办"
      subtitle="你自己加的待办可以修改和删除。"
      width="540px"
      @close="addOpen = false"
    >
      <div class="form">
        <FormField label="标题">
          <TextField v-model="newTodo.title" placeholder="一句话说清这一步做什么" />
        </FormField>
        <FormField label="做法（可选）">
          <TextArea v-model="newTodo.detail" :rows="3" placeholder="改哪个模块、用什么方式" />
        </FormField>
      </div>
      <template #footer>
        <button class="btn ghost" @click="addOpen = false">取消</button>
        <div class="spacer" />
        <button class="btn primary" :disabled="busy" @click="addTodo">添加</button>
      </template>
    </Dialog>
  </div>
  <Teleport to="body">
    <div
      v-if="todoTip"
      ref="tipPanel"
      class="td-tip"
      :class="{ placed: todoTip.placed }"
      :style="{ top: todoTip.top + 'px', left: todoTip.left + 'px' }"
      role="tooltip"
      @mouseenter="holdTodoTip"
      @mouseleave="queueHideTodoTip"
    >
      <p v-if="todoTip.detail"><span class="td-tip-k">详情</span>{{ todoTip.detail }}</p>
    </div>
  </Teleport>
  <TaskChat
    v-if="session.currentProjectId"
    ref="railChat"
    :project-id="session.currentProjectId"
    :task-id="taskId"
    @changed="load"
  />
  </div>
</template>

<style scoped>
.plan {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.plan-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 14px 24px;
  border-bottom: 1px solid var(--line-soft);
  background: var(--ink-850);
  -webkit-app-region: drag;
  flex: none;
}
.plan-head .btn {
  -webkit-app-region: no-drag;
  flex: none;
}
.title-actions {
  width: 100%;
}
.plan-progress {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-dim);
}
.plan-progress .bar {
  width: 72px;
  height: 3px;
  border-radius: 2px;
  background: var(--ink-700);
  overflow: hidden;
}
.plan-progress .fill {
  display: block;
  height: 100%;
  background: var(--brass);
}
.back {
  font-size: 11.5px;
  padding-left: 0;
}
.loading {
  padding: 40px 24px;
  font-size: 14px;
  line-height: 1.5;
  color: var(--text-faint);
}

.plan-body {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
}
.main {
  flex: 1;
  overflow-y: auto;
  padding: 26px 32px 48px;
  display: flex;
  flex-direction: column;
  gap: 26px;
  min-width: 0;
}
.title-block {
  display: flex;
  flex-direction: column;
  gap: 9px;
}
.proj {
  font-size: 12px;
  font-weight: 400;
  letter-spacing: normal;
  color: var(--text-dim);
  text-transform: none;
}
.title-block h1 {
  font-size: 20px;
  font-weight: 600;
  line-height: 1.25;
  letter-spacing: normal;
}
.title-block .row {
  flex-wrap: wrap;
  gap: 7px;
  margin-top: 2px;
}

.block {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.block-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
}
.block-title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
  color: var(--text-strong);
}
.block-head .faint {
  font-size: 11px;
}
.description {
  color: var(--text);
}

.todos {
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.todo {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 4px 0;
}
.todo.doing .seq {
  color: var(--brass);
}
.seq {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-faint);
  flex: none;
  padding-top: 2px;
  width: 16px;
}
.body {
  flex: 1;
  min-width: 0;
  gap: 3px;
}
.td-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-strong);
  line-height: 1.5;
  min-width: 0;
}
.td-title-text {
  color: inherit;
}
.todo.done .td-title-text {
  text-decoration: line-through;
  text-decoration-color: var(--moss-dim);
  color: var(--text-faint);
}
.td-help {
  width: 15px;
  height: 15px;
  margin-left: 6px;
  border-radius: 50%;
  border: 1px solid var(--line-strong);
  color: var(--text-faint);
  font-size: 11px;
  font-weight: 600;
  line-height: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  vertical-align: 1px;
  background: transparent;
  cursor: help;
  padding: 0;
}
.td-help:hover,
.td-help:focus-visible {
  color: var(--brass-bright);
  border-color: var(--brass-dim);
  outline: none;
}
.td-tip {
  position: fixed;
  z-index: 50;
  width: max-content;
  max-width: 320px;
  padding: 8px 10px;
  background: var(--paper-800);
  border: 1px solid var(--line);
  border-radius: var(--r-md);
  box-shadow: var(--shadow-panel);
  display: flex;
  flex-direction: column;
  gap: 6px;
  visibility: hidden;
}
.td-tip.placed {
  visibility: visible;
}
.td-tip p {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.td-tip-k {
  display: block;
  margin-bottom: 1px;
  font-size: 11px;
  color: var(--text-faint);
}
.lock,
.owned {
  font-size: 9px;
  border-radius: 3px;
  padding: 0 3px;
  flex: none;
  margin-left: 6px;
}
.lock {
  color: var(--brass-dim);
  border: 1px solid var(--brass-dim);
}
.owned {
  color: var(--text-faint);
  border: 1px solid var(--line-strong);
}
.td-actions {
  display: flex;
  gap: 2px;
  flex: none;
}
.done-mark {
  color: var(--moss);
  font-size: 10px;
  flex: none;
  padding-top: 2px;
}
.pad {
  padding: 10px 2px;
  font-size: 14px;
  line-height: 1.5;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 13px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.k {
  font-size: 11.5px;
  color: var(--text-dim);
}
.tip {
  font-size: 10.5px;
  color: var(--text-faint);
  line-height: 1.6;
}
</style>
