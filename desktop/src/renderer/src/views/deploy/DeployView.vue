<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { adeApi } from '../../api'
import { useSessionStore, messageOf } from '../../stores/session'
import DeployChat from './components/DeployChat.vue'
import type { DeployChatHandle } from './components/deployChat'
import { confirm } from '../../composables/confirm'
import { toast } from '../../composables/toast'
import { usePageSurface } from '../../surfaces/host'
import { deployPage, deploySurface } from '../../surfaces/deploy'
import Dialog from '../../components/Dialog.vue'
import EmptyState from '../../components/EmptyState.vue'
import FormField from '../../components/FormField.vue'
import TextArea from '../../components/TextArea.vue'
import TextField from '../../components/TextField.vue'
import type { DeployTask } from '../../../../shared/types'

const session = useSessionStore()
const railChat = ref<DeployChatHandle | null>(null)

const busy = ref(false)
const saving = ref(false)
const formErr = ref('')
usePageSurface(deploySurface)

const tasks = computed(() =>
  [...deployPage.tasks].sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
)
const streaming = computed(() => {
  const chat = railChat.value
  if (!chat || chat.state.projectId !== session.currentProjectId) return false
  return chat.state.streaming
})

async function load(): Promise<void> {
  const projectId = session.currentProjectId
  deployPage.tasks = []
  if (!projectId) return
  try {
    const data = await adeApi.listDeployTasks(projectId)
    if (session.currentProjectId !== projectId) return
    deployPage.tasks = data.list ?? []
  } catch (err) {
    toast.error(messageOf(err))
  }
}

function openForm(task?: DeployTask): void {
  formErr.value = ''
  deployPage.editingId = task?.id ?? 0
  deployPage.title = task?.title ?? ''
  deployPage.prompt = task?.prompt ?? ''
  deployPage.open = true
}

function closeForm(): void {
  if (saving.value) return
  deployPage.open = false
}

async function save(): Promise<void> {
  const projectId = session.currentProjectId
  formErr.value = ''
  if (!projectId) {
    formErr.value = '先选择一个项目'
    return
  }
  const title = deployPage.title.trim()
  const prompt = deployPage.prompt.trim()
  if (!title || !prompt) {
    formErr.value = '标题和说明都要填'
    return
  }
  saving.value = true
  try {
    if (deployPage.editingId) {
      const saved = await adeApi.updateDeployTask(deployPage.editingId, { title, prompt })
      const idx = deployPage.tasks.findIndex((item) => item.id === saved.id)
      if (idx >= 0) deployPage.tasks[idx] = saved
      else deployPage.tasks.unshift(saved)
    } else {
      const saved = await adeApi.createDeployTask(projectId, { title, prompt })
      deployPage.tasks.unshift(saved)
    }
    deployPage.open = false
    toast.success('部署任务已保存')
  } catch (err) {
    formErr.value = messageOf(err)
  } finally {
    saving.value = false
  }
}

async function remove(task: DeployTask): Promise<void> {
  const ok = await confirm({
    title: '删除部署任务',
    message: `确定删除「${task.title}」？`,
    confirmText: '删除',
    danger: true
  })
  if (!ok) return
  try {
    await adeApi.deleteDeployTask(task.id)
    deployPage.tasks = deployPage.tasks.filter((item) => item.id !== task.id)
    toast.show('部署任务已删除')
  } catch (err) {
    toast.error(messageOf(err))
  }
}

async function run(task: DeployTask): Promise<void> {
  const projectId = session.currentProjectId
  if (!projectId) {
    toast.error('先选择一个项目')
    return
  }
  if (streaming.value) {
    toast.error('上一段对话还在进行')
    return
  }
  const chat = railChat.value
  if (!chat || chat.state.projectId !== projectId) {
    toast.error('部署会话还没打开')
    return
  }
  busy.value = true
  try {
    if (chat.state.streaming) {
      toast.error('上一段对话还在进行')
      return
    }
    const modelId = session.profile.modelId || session.currentProject?.modelId || 0
    chat.fresh()
    await chat.send(task.prompt.trim(), task.title.trim(), modelId)
    toast.success('已交给部署智能体，过程在右侧')
  } catch (err) {
    toast.error(messageOf(err))
  } finally {
    busy.value = false
  }
}

function shortTime(value: string): string {
  if (!value) return ''
  return value.replace('T', ' ').slice(0, 16)
}

watch(() => session.currentProjectId, () => void load(), { immediate: true })
</script>

<template>
  <div class="page-with-chat">
  <div class="deploy">
    <header class="page-head">
      <div class="col">
        <span class="eyebrow">部署</span>
        <h1 class="serif">{{ session.currentProject?.name || '当前项目' }}</h1>
        <p class="lede">先保存部署任务，再点执行。过程出现在右侧会话里，主机连接仍在侧栏「主机」。</p>
      </div>
      <div class="spacer" />
      <button class="btn primary sm" :disabled="!session.currentProjectId" @click="openForm()">新增</button>
    </header>

    <div class="scroll">
      <EmptyState
        v-if="!tasks.length"
        empty-title="还没有部署任务"
        empty-hint="新增一条：写清标题和要部署什么。"
      />
      <div v-else class="list">
        <article v-for="task in tasks" :key="task.id" class="card task">
          <div class="task-head">
            <span class="task-title serif ellipsis">{{ task.title }}</span>
            <span class="task-time">{{ shortTime(task.createdAt) }}</span>
          </div>
          <p class="task-prompt">{{ task.prompt }}</p>
          <div class="task-actions">
            <button class="btn ghost sm" @click="openForm(task)">编辑</button>
            <button class="btn ghost sm danger" @click="remove(task)">删除</button>
            <div class="spacer" />
            <button class="btn primary sm" :disabled="busy || streaming" @click="run(task)">
              {{ streaming ? '执行中…' : '执行' }}
            </button>
          </div>
        </article>
      </div>
    </div>

    <Dialog
      :open="deployPage.open"
      :title="deployPage.editingId ? '编辑部署任务' : '新增部署任务'"
      subtitle="保存后出现在列表里。执行才会发给右侧的部署智能体。"
      width="560px"
      :persistent="saving"
      @close="closeForm"
    >
      <div class="form">
        <FormField label="标题">
          <TextField v-model="deployPage.title" placeholder="例如：把当前版本发到测试机" />
        </FormField>
        <FormField label="说明">
          <TextArea v-model="deployPage.prompt" :rows="8" placeholder="要做什么、注意什么。智能体会在当前项目目录里执行。" />
        </FormField>
      </div>
      <p v-if="formErr" class="err">{{ formErr }}</p>
      <template #footer>
        <button class="btn ghost" :disabled="saving" @click="closeForm">取消</button>
        <div class="spacer" />
        <button class="btn primary" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存' }}</button>
      </template>
    </Dialog>
  </div>
  <DeployChat
    v-if="session.currentProjectId"
    ref="railChat"
    :project-id="session.currentProjectId"
  />
  </div>
</template>

<style scoped>
.deploy { flex: 1; display: flex; flex-direction: column; min-height: 0; }
.page-head {
  display: flex;
  align-items: flex-end;
  gap: 22px;
  padding: 22px 26px 14px;
  border-bottom: 1px solid var(--line-soft);
  flex-wrap: wrap;
}
.eyebrow { font-size: 10.5px; letter-spacing: 0.18em; color: var(--brass-dim); }
h1 { font-size: 21px; }
.lede { margin-top: 6px; font-size: 12px; color: var(--text-dim); max-width: 46em; line-height: 1.7; }
.scroll { flex: 1; overflow: auto; padding: 18px 26px 32px; }
.list { display: flex; flex-direction: column; gap: 10px; max-width: 760px; }
.task { display: flex; flex-direction: column; gap: 8px; padding: 14px 16px; }
.task-head, .task-actions { display: flex; align-items: center; gap: 8px; }
.task-title { flex: 1; min-width: 0; font-size: 15px; color: var(--text-strong); }
.task-time { font-family: var(--font-mono); font-size: 11px; color: var(--text-faint); flex: none; }
.task-prompt {
  font-size: 12.5px;
  line-height: 1.65;
  color: var(--text-dim);
  white-space: pre-wrap;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.task-actions { margin-top: 2px; padding-top: 8px; border-top: 1px solid var(--line-soft); }
.form { display: flex; flex-direction: column; gap: 12px; }
.err { color: var(--rust); font-size: 12px; margin-top: 8px; }
</style>
