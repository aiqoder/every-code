<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AskQuestionPanel from '../../../components/AskQuestionPanel.vue'
import ChatTimeline from '../../../components/ChatTimeline.vue'
import IconHistory from '../../../components/icons/IconHistory.vue'
import IconNewChat from '../../../components/icons/IconNewChat.vue'
import TextArea from '../../../components/TextArea.vue'
import TodoFloat from '../../../components/TodoFloat.vue'
import { llmApi, type LlmModelRow } from '../../../api'
import { messageOf, useSessionStore } from '../../../stores/session'
import { useTaskChat, type TaskChatHandle } from './taskChat'
import { toast } from '../../../composables/toast'
import { isProviderEnabled } from '../../../lib/providerEnabled'

export type { TaskChatHandle }

const props = defineProps<{
  projectId: number
  taskId: number
}>()

const session = useSessionStore()
const emit = defineEmits<{ changed: [] }>()
const {
  state,
  agentName,
  activate,
  dispose,
  reloadConversations,
  openConversation,
  fresh,
  send,
  answer,
  applyModel
} = useTaskChat({
  projectId: () => props.projectId,
  taskId: () => props.taskId,
  onChanged: () => emit('changed')
})

const models = ref<LlmModelRow[]>([])
const menuOpen = ref(false)
const historyOpen = ref(false)
const modelWrap = ref<HTMLElement | null>(null)
const historyWrap = ref<HTMLElement | null>(null)

const plan = computed(() => state.plan)
const history = computed(() =>
  [...state.conversations].sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
)

watch(
  () => [props.projectId, props.taskId] as const,
  () => {
    historyOpen.value = false
    void activate().catch((err) => toast.error(messageOf(err)))
  },
  { immediate: true }
)

async function pick(id: number): Promise<void> {
  if (!state.projectId || !state.taskId) return
  try {
    await openConversation(id)
  } catch (err) {
    toast.error(messageOf(err))
  }
}

async function toggleHistory(): Promise<void> {
  historyOpen.value = !historyOpen.value
  if (!historyOpen.value || !state.projectId || !state.taskId) return
  await reloadConversations()
}

const modelOptions = computed(() =>
  models.value.filter((item) => isProviderEnabled(item.providerId) || item.id === session.profile.modelId)
)

const selectedModelId = computed(() => {
  const preferred = session.profile.modelId
  if (modelOptions.value.some((item) => item.id === preferred)) return preferred
  const projectModel = session.projects.find((item) => item.id === props.projectId)?.modelId ?? 0
  if (modelOptions.value.some((item) => item.id === projectModel)) return projectModel
  return modelOptions.value[0]?.id ?? 0
})

const selectedModel = computed(() => modelOptions.value.find((item) => item.id === selectedModelId.value) ?? null)

async function loadModels(): Promise<void> {
  try {
    const data = await llmApi.listModels()
    models.value = data.list ?? []
  } catch {
    models.value = []
  }
}

async function toggleMenu(): Promise<void> {
  if (menuOpen.value) {
    menuOpen.value = false
    return
  }
  await loadModels()
  menuOpen.value = true
}

async function pickModel(id: number): Promise<void> {
  menuOpen.value = false
  try {
    if (id !== session.profile.modelId) await session.setDefaultModel(id)
    await applyModel(id)
  } catch (err) {
    toast.error(messageOf(err))
  }
}

function onDocPointer(event: PointerEvent): void {
  const target = event.target
  if (!(target instanceof Node)) return
  if (menuOpen.value && modelWrap.value && !modelWrap.value.contains(target)) menuOpen.value = false
  if (historyOpen.value && historyWrap.value && !historyWrap.value.contains(target)) historyOpen.value = false
}

async function submit(): Promise<void> {
  if (!state.projectId || !state.taskId) return
  try {
    await send(state.draft, '', selectedModelId.value)
  } catch (err) {
    toast.error(messageOf(err))
  }
}

async function onAnswer(payload: {
  conversationId: number
  questionId: string
  answers: Array<{ questionId: string; selected: string[]; text?: string }>
}): Promise<void> {
  try {
    await answer(payload)
  } catch (err) {
    toast.error(messageOf(err))
  }
}

function onKey(event: KeyboardEvent): void {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    void submit()
  }
}

function startFresh(): void {
  historyOpen.value = false
  fresh()
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocPointer)
  void loadModels()
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocPointer)
  dispose()
})

defineExpose<TaskChatHandle>({ state, send, fresh })
</script>

<template>
  <section class="chat rail">
    <header class="chat-head">
      <div class="head-actions">
        <div ref="historyWrap" class="history-pick" :class="{ open: historyOpen }">
          <button
            type="button"
            class="btn ghost sm icon"
            :disabled="!state.projectId || !state.taskId"
            aria-label="历史"
            title="历史"
            :aria-expanded="historyOpen"
            @click="toggleHistory"
          >
            <IconHistory />
          </button>
          <div v-if="historyOpen" class="history-pop" role="listbox" aria-label="历史会话">
            <p v-if="!history.length" class="history-empty">还没有会话</p>
            <button
              v-for="item in history"
              :key="item.id"
              type="button"
              role="option"
              class="history-opt"
              :class="{ on: item.id === state.activeId }"
              :aria-selected="item.id === state.activeId"
              @click="historyOpen = false; pick(item.id)"
            >
              {{ item.title || '未命名' }}
            </button>
          </div>
        </div>
        <button
          class="btn ghost sm icon"
          :disabled="!state.projectId || !state.taskId || state.streaming"
          aria-label="新对话"
          title="新对话"
          @click="startFresh"
        >
          <IconNewChat />
        </button>
      </div>
    </header>

    <div class="stage">
      <div v-if="plan" class="overlays">
        <TodoFloat :plan="plan" />
      </div>
      <ChatTimeline :lines="state.lines" :agent-name="agentName" :streaming="state.streaming">
        <template #empty>
          <p v-if="!state.projectId || !state.taskId" class="empty">先选择一个任务</p>
          <p v-else class="empty">在下面写一句，发给{{ agentName }}。</p>
        </template>
      </ChatTimeline>
    </div>

    <AskQuestionPanel class="chat-ask" :question="state.pendingAsk" @answer="onAnswer" />

    <form class="composer" @submit.prevent="submit">
      <TextArea
        v-if="state.projectId && state.taskId"
        v-model="state.draft"
        :rows="3"
        placeholder="输入消息，回车发送"
        :disabled="state.streaming"
        @keydown="onKey"
      />
      <div class="composer-bar">
        <div ref="modelWrap" class="model-pick" :class="{ open: menuOpen }">
          <button
            type="button"
            class="model-trigger"
            :disabled="!modelOptions.length"
            aria-label="选择模型"
            :aria-expanded="menuOpen"
            @click="toggleMenu"
          >
            <span class="model-label" :class="{ placeholder: !selectedModel }">
              {{ selectedModel?.name || '选择模型' }}
            </span>
            <span class="chev" aria-hidden="true" />
          </button>
          <div v-if="menuOpen" class="model-pop" role="listbox" aria-label="选择模型">
            <button
              v-for="item in modelOptions"
              :key="item.id"
              type="button"
              role="option"
              class="model-opt"
              :class="{ on: item.id === selectedModelId }"
              :aria-selected="item.id === selectedModelId"
              @click="pickModel(item.id)"
            >
              <span class="opt-main">
                <strong>{{ item.name }}</strong>
                <small>{{ item.providerName }}</small>
              </span>
              <span v-if="item.id === selectedModelId" class="mark">✓</span>
            </button>
          </div>
        </div>
        <button
          class="btn primary sm send"
          :disabled="!state.projectId || !state.taskId || state.streaming || !state.draft.trim() || !selectedModelId"
        >
          {{ state.streaming ? '进行中…' : '发送' }}
        </button>
      </div>
    </form>
  </section>
</template>

<style scoped>
.chat {
  flex: 1 1 auto;
  width: auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  background: transparent;
}
.chat.rail {
  width: 340px;
  flex: none;
  border-left: 1px solid var(--line-soft);
  background: var(--ink-800);
}
.chat-head {
  position: relative;
  z-index: 3;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 14px 14px 10px;
  border-bottom: 1px solid var(--line-soft);
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: none;
}
.head-actions .btn.icon svg {
  width: 15px;
  height: 15px;
}
.history-pick {
  position: relative;
}
.history-pop {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 24;
  width: 220px;
  max-height: 280px;
  overflow: auto;
  padding: 4px;
  background: var(--ink-800);
  border: 1px solid var(--line);
  border-radius: var(--r-md);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.28);
}
.history-empty {
  padding: 8px;
  font-size: 12px;
  color: var(--text-faint);
}
.history-opt {
  display: block;
  width: 100%;
  padding: 7px 8px;
  border-radius: var(--r-sm);
  text-align: left;
  font-size: 12px;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.history-opt:hover,
.history-opt.on {
  color: var(--text-strong);
  background: var(--brass-glow);
}
.stage {
  position: relative;
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.overlays {
  position: absolute;
  top: 8px;
  left: 8px;
  right: 8px;
  z-index: 4;
  display: flex;
  flex-direction: column;
  gap: 8px;
  pointer-events: none;
}
.overlays > :deep(*) {
  pointer-events: auto;
}
.chat-ask {
  flex: none;
  min-height: 0;
  max-height: min(280px, 46%);
  margin: 0 12px 8px;
  overflow: auto;
}
.composer {
  position: relative;
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 0;
  margin: 10px 12px 12px;
  border: 1px solid var(--line);
  border-radius: var(--r-md);
  background: var(--ink-850);
  overflow: visible;
}
.composer:focus-within {
  border-color: var(--brass);
  box-shadow: 0 0 0 3px var(--brass-glow);
}
.composer :deep(textarea.field-area) {
  width: 100%;
  resize: none;
  min-height: 72px;
  max-height: 160px;
  background: transparent;
  color: var(--text-strong);
  border: 0;
  border-radius: 0;
  box-shadow: none;
  padding: 10px 12px 4px;
  font: inherit;
  font-size: 14px;
  line-height: 1.5;
}
.composer :deep(textarea.field-area:focus) {
  border-color: transparent;
  box-shadow: none;
}
.composer-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 4px 8px 8px;
}
.model-pick {
  position: relative;
  min-width: 0;
  max-width: 168px;
}
.model-trigger {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  min-height: 28px;
  padding: 2px 6px 2px 8px;
  border: 1px solid transparent;
  border-radius: var(--r-sm);
  background: transparent;
  color: var(--text-dim);
  font-size: 12px;
  font-weight: 500;
}
.model-pick.open .model-trigger,
.model-trigger:hover:not(:disabled) {
  color: var(--text-strong);
  background: var(--ink-700);
  border-color: var(--line);
}
.model-trigger:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.model-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.model-label.placeholder {
  color: var(--text-faint);
}
.chev {
  width: 0;
  height: 0;
  border-left: 3.5px solid transparent;
  border-right: 3.5px solid transparent;
  border-top: 4px solid currentColor;
  opacity: 0.7;
  flex: none;
}
.model-pick.open .chev {
  transform: rotate(180deg);
}
.model-pop {
  position: absolute;
  left: 0;
  bottom: calc(100% + 6px);
  z-index: 24;
  min-width: 220px;
  max-width: 280px;
  max-height: 240px;
  overflow: auto;
  padding: 4px;
  background: var(--ink-800);
  border: 1px solid var(--line);
  border-radius: var(--r-md);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.28);
}
.model-opt {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 8px;
  border-radius: var(--r-sm);
  text-align: left;
  color: var(--text);
}
.model-opt:hover,
.model-opt.on {
  background: var(--brass-glow);
}
.opt-main {
  min-width: 0;
  flex: 1;
  display: grid;
  gap: 1px;
}
.opt-main strong {
  font-size: 12.5px;
  font-weight: 500;
  color: var(--text-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.opt-main small {
  font-size: 11px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mark {
  color: var(--brass);
  font-size: 12px;
  flex: none;
}
.send {
  margin-left: auto;
  flex: none;
}
</style>
