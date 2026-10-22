<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { adeApi, watchTaskEvents } from '../../api'
import { useSessionStore, messageOf } from '../../stores/session'
import { toast } from '../../composables/toast'
import { usePageSurface } from '../../surfaces/host'
import { showResult } from '../../surfaces/registry'
import {
  addTaskTodo,
  answerTaskQuestion,
  focusTaskTerminal,
  removeTaskTodo,
  taskPage,
  taskSurface,
  toggleTaskTodo,
  updateTaskDescription
} from '../../surfaces/task'
import { parseAskQuestionEvent, isAskTimeoutResult } from '../../composables/askQuestion'
import TaskChat, { type TaskChatHandle } from './components/TaskChat.vue'
import StatusPill from '../../components/StatusPill.vue'
import Dialog from '../../components/Dialog.vue'
import FormField from '../../components/FormField.vue'
import TextArea from '../../components/TextArea.vue'
import TextField from '../../components/TextField.vue'
import AskQuestionPanel from '../../components/AskQuestionPanel.vue'
import MarkdownBody from '../../components/MarkdownBody.vue'
import type { TaskEvent } from '../../../../shared/types'
import { taskDisplayName } from '../../utils/taskTitle'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const railChat = ref<TaskChatHandle | null>(null)

const task = computed(() => taskPage.task)
const loading = ref(true)
const busy = computed(() => taskPage.busy)
const events = ref<TaskEvent[]>([])
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
const newTodo = reactive({ title: '', detail: '', acceptance: '', dependsOn: '' })

type TodoExtra = { detail?: string; acceptance?: string; dependsOn?: string }
type TodoTip = {
  detail: string
  acceptance: string
  dependsOn: string
  top: number
  left: number
  placed: boolean
}

const todoTip = ref<TodoTip | null>(null)
const tipPanel = ref<HTMLElement | null>(null)
let tipHideTimer: ReturnType<typeof setTimeout> | null = null
let tipToken = 0

function todoExtra(td: TodoExtra): Pick<TodoTip, 'detail' | 'acceptance' | 'dependsOn'> {
  return {
    detail: td.detail?.trim() ?? '',
    acceptance: td.acceptance?.trim() ?? '',
    dependsOn: td.dependsOn?.trim() ?? ''
  }
}

function hasTodoExtra(td: TodoExtra): boolean {
  const extra = todoExtra(td)
  return Boolean(extra.detail || extra.acceptance || extra.dependsOn)
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

/** 拆解过程里用于状态文案的工具调用，不直接铺到页面上。 */
const toolLines = computed(() => {
  const chat = railChat.value
  if (!chat || chat.state.taskId !== taskId.value) return []
  return chat.state.lines.filter((line) => line.role === 'tool' && (line.toolName || line.content.trim()))
})

/** 只展示助手正文，不把工具名堆在待办区。 */
const processLines = computed(() => {
  const chat = railChat.value
  if (!chat || chat.state.taskId !== taskId.value) return []
  return chat.state.lines.filter((line) => line.role === 'assistant' && line.content.trim())
})

const runLive = computed(() => {
  let live = false
  for (const ev of events.value) {
    if (ev.type === 'run_started') live = true
    if (ev.type === 'run_finished' || ev.type === 'error') live = false
  }
  return live
})

const replanning = ref(false)

const isPlanning = computed(() => {
  const t = task.value
  if (!t) return false
  if (!(chatStreaming.value || runLive.value)) return false
  return !t.planned || replanning.value
})

/** 待执行且已有计划内容时显示；不单靠 planned，避免标志未写上时按钮一直不出现。 */
const showReplan = computed(() => {
  const t = task.value
  if (!t || t.status !== 'pending') return false
  return Boolean(t.planned || t.name.trim() || t.todos.length)
})

const canReplan = computed(() => {
  if (!showReplan.value) return false
  const t = task.value
  if (!t) return false
  if (!(t.rawRequest || t.description).trim()) return false
  return !chatStreaming.value && !busy.value && !replanning.value
})

const streamText = computed(() => {
  return events.value
    .filter((e) => e.type === 'text_delta' && e.text)
    .map((e) => e.text)
    .join('')
})

const showProcess = computed(
  () =>
    isPlanning.value ||
    ((!task.value?.planned || replanning.value) &&
      (processLines.value.length > 0 || Boolean(streamText.value)))
)

const runningLine = computed(() => {
  const lastTool = toolLines.value.at(-1)
  if (chatStreaming.value && lastTool) return `正在调用 ${lastTool.toolName || lastTool.content}`
  if (chatStreaming.value) return replanning.value ? '正在重新拆解…' : '正在整理计划…'
  const last = [...events.value].reverse().find((e) => e.type === 'log' || e.type === 'tool_call_start')
  if (last?.type === 'tool_call_start') return `正在调用 ${last.toolName ?? '工具'}`
  if (last?.text) return last.text
  return replanning.value ? '正在重新拆解…' : '正在整理计划…'
})

watch(chatStreaming, (on, was) => {
  if (!was || on) return
  if (replanning.value) replanning.value = false
  // 拆解结束后刷新，保证 planned / 名称 / 待办写回后「重新拆解」能立刻出现
  void load()
})


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
  const t = task.value
  if (!t || t.status !== 'failed') return null
  if (t.lastError?.message) {
    return {
      kind: KIND_LABEL[t.lastError.kind] ?? t.lastError.kind,
      message: t.lastError.message,
      at: t.lastError.at
    }
  }
  return null
})

async function load(): Promise<void> {
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
    events.value = [...events.value.slice(-500), ev]
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
    detail: newTodo.detail,
    acceptance: newTodo.acceptance,
    dependsOn: newTodo.dependsOn
  })
  showResult(result)
  if (result.error) return
  newTodo.title = ''
  newTodo.detail = ''
  newTodo.acceptance = ''
  newTodo.dependsOn = ''
  addOpen.value = false
}

async function removeTodo(id: number): Promise<void> {
  showResult(await removeTaskTodo(id))
}

async function toggleTodo(id: number, status: string): Promise<void> {
  showResult(await toggleTaskTodo(id, status))
}

async function focusTerminal(): Promise<void> {
  showResult(await focusTaskTerminal())
}

async function replan(): Promise<void> {
  const current = task.value
  const chat = railChat.value
  if (!canReplan.value || !current) return
  if (!chat?.state.ready || chat.state.taskId !== current.id) {
    toast.error('右侧对话还没准备好')
    return
  }
  if (!window.confirm('将重新整理名称、概览和待办，并覆盖当前计划。继续？')) return
  const text = (current.rawRequest || current.description || '').trim()
  chat.fresh()
  replanning.value = true
  try {
    const started = await chat.send(
      `请根据当前需求重新拆解整份计划，调用 save_plan 覆盖写回名称、概览、润色需求和全部待办。\n\n需求原文：\n${text}`,
      '重新拆解',
      session.profile.modelId
    )
    if (!started) {
      replanning.value = false
      toast.error('未能开始重新拆解')
    }
  } catch (err) {
    replanning.value = false
    toast.error(messageOf(err))
  }
}

onMounted(async () => {
  await load()
  subscribe()
  poll = setInterval(() => {
    if (task.value && (task.value.status === 'running' || !task.value.planned)) void load()
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
      <button class="btn sm" @click="openEdit">修改需求</button>
      <button
        v-if="showReplan"
        class="btn sm"
        :disabled="!canReplan"
        @click="replan"
      >
        {{ replanning || chatStreaming ? '拆解中…' : '重新拆解' }}
      </button>
      <button v-if="task?.worktree" class="btn ghost sm" @click="focusTerminal">
        聚焦终端
      </button>
    </header>

    <div v-if="loading && !task" class="loading">正在读取任务…</div>

    <div v-else-if="task" class="plan-body">
      <div class="main" @scroll="hideTodoTipNow">
        <div class="title-block">
          <span class="proj">{{ task.projectName }} · #{{ task.id }}</span>
          <h1 class="serif">{{ taskDisplayName(task) }}</h1>
          <div class="row title-actions">
            <StatusPill :status="task.status" />
            <span v-if="task.mergedDev" class="tag"><span class="dot" style="color: var(--moss)" />已合 dev</span>
            <span v-if="task.mergedMain" class="tag"><span class="dot" style="color: var(--brass)" />已合 main</span>
            <div class="spacer" />
            <button
              v-if="showReplan"
              class="btn sm"
              :disabled="!canReplan"
              @click="replan"
            >
              {{ replanning || chatStreaming ? '拆解中…' : '重新拆解' }}
            </button>
          </div>
        </div>

        <AskQuestionPanel :question="pendingAsk" @answer="submitAsk" />

        <section v-if="task.overview" class="block">
          <h3 class="serif block-title">概览</h3>
          <MarkdownBody :content="task.overview" />
        </section>

        <section v-if="failure" class="block fail-block">
          <div class="block-head">
            <h3 class="serif block-title">执行出错 · {{ failure.kind }}</h3>
            <div class="spacer" />
            <span class="faint">{{ failure.at }}</span>
          </div>
          <pre class="fail-msg selectable">{{ failure.message }}</pre>
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

          <div v-if="showProcess" class="process-wrap">
            <p v-if="isPlanning" class="planning">
              <span class="spin" />
              {{ runningLine }}
            </p>
            <div v-if="processLines.length || streamText" class="process">
              <p v-for="line in processLines" :key="line.id" class="process-line" :class="line.role">
                {{ line.content }}
              </p>
              <pre v-if="streamText" class="stream selectable">{{ streamText }}</pre>
            </div>
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
                      aria-label="查看详情和验收"
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

            <p v-if="!task.todos.length && !showProcess" class="faint pad">还没有待办。</p>
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
        <FormField label="验收标准（可选）">
          <TextField v-model="newTodo.acceptance" placeholder="怎样算做完" />
        </FormField>
        <FormField label="依赖（可选）">
          <TextField v-model="newTodo.dependsOn" placeholder="例如：第 2 条待办" />
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
      <p v-if="todoTip.acceptance"><span class="td-tip-k">验收</span>{{ todoTip.acceptance }}</p>
      <p v-if="todoTip.dependsOn"><span class="td-tip-k">依赖</span>{{ todoTip.dependsOn }}</p>
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
.dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
  display: inline-block;
}

.block {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.fail-block {
  border: 1px solid var(--rust-dim);
  border-left: 2px solid var(--rust);
  border-radius: var(--r-md);
  background: rgba(180, 91, 74, 0.07);
  padding: 11px 13px;
}
.fail-block .block-title {
  color: var(--rust);
}
.fail-msg {
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.5;
  color: var(--text);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 280px;
  overflow-y: auto;
  margin: 0;
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

.process-wrap {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.process {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 420px;
  overflow: auto;
  padding: 12px 14px;
  background: var(--paper-900);
  border: 1px solid var(--line-soft);
  border-radius: var(--r-md);
}
.process-line {
  margin: 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-size: 14px;
  font-weight: 400;
  line-height: 1.5;
  color: var(--text);
}
.process-line.tool {
  font-family: var(--font-mono);
  color: var(--text-dim);
}
.planning {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 9px 12px;
  font-size: 14px;
  line-height: 1.5;
  color: var(--brass);
  background: rgba(200, 160, 90, 0.07);
  border: 1px solid var(--brass-dim);
  border-radius: var(--r-md);
}
.spin {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  border: 1.5px solid var(--brass-dim);
  border-top-color: var(--brass);
  flex: none;
  animation: spin 0.9s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
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
.stream {
  max-height: 260px;
  font-size: 12px;
  line-height: 1.45;
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
