<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { bridge, llmApi, type LlmModelRow } from '../../api'
import { isProviderEnabled } from '../../lib/providerEnabled'
import { confirm } from '../../composables/confirm'
import { toast } from '../../composables/toast'
import { messageOf, useSessionStore } from '../../stores/session'
import { showResult } from '../../surfaces/registry'
import {
  createPage,
  createProject,
  deleteProject,
  parseEnvReport,
  persistScene,
  reloadScenes,
  removeCustomScene,
  saveEnvCheck,
  updateProject
} from '../../surfaces/create'
import EnvChat from './components/EnvChat.vue'
import type { EnvChatHandle } from './components/envChat'
import Dialog from '../../components/Dialog.vue'
import FormField from '../../components/FormField.vue'
import TextArea from '../../components/TextArea.vue'
import TextField from '../../components/TextField.vue'
import EmptyState from '../../components/EmptyState.vue'
import type { Scene } from '../../../../shared/types'

const session = useSessionStore()
const router = useRouter()
const route = useRoute()

const wizardOpen = ref(false)
const step = ref(1)
const busy = ref(false)
const error = ref('')

const form = reactive({
  name: '',
  localDir: '',
  modelId: 0,
  sceneId: 0
})

const scenesBusy = ref(false)
const detailScene = ref<Scene | null>(null)
const customOpen = ref(false)
const customBusy = ref(false)
const customError = ref('')
const customForm = reactive({
  id: 0,
  title: '',
  description: ''
})

const envResult = computed(() => createPage.envResult)
const envChatOpen = ref(false)
const envConvId = ref(0)
const envChat = ref<EnvChatHandle | null>(null)
const envStreaming = computed(() => envChat.value?.state.streaming ?? false)

const editOpen = ref(false)
const editBusy = ref(false)
const editForm = reactive({
  name: '',
  description: '',
  localDir: ''
})

const removeBusy = ref(false)

const enabledModels = ref<LlmModelRow[]>([])

const project = computed(() => session.currentProject)
const currentScene = computed(() => {
  const id = project.value?.sceneId ?? 0
  if (!id) return null
  return createPage.scenes.find((item) => item.id === id) ?? null
})

const canNext = computed(() => {
  if (step.value === 1) return Boolean(form.name.trim() && form.localDir.trim() && form.modelId > 0)
  return true
})

const envList = computed(() => (envResult.value.length ? envResult.value : (project.value?.envCheck ?? [])))
const envTone = computed(() => {
  const list = envList.value
  if (!list.length) return '未检测'
  return list.every((item) => item.installed) ? '已齐备' : '有缺失'
})

async function loadEnabledModels(): Promise<void> {
  try {
    const [providerData, modelData] = await Promise.all([llmApi.listProviders(), llmApi.listModels()])
    const enabled = new Set(
      (providerData.list ?? []).filter((item) => isProviderEnabled(item.id)).map((item) => item.id)
    )
    enabledModels.value = (modelData.list ?? []).filter((item) => enabled.has(item.providerId))
    if (!enabledModels.value.some((item) => item.id === form.modelId)) {
      const preferred = session.profile.modelId
      form.modelId = enabledModels.value.some((item) => item.id === preferred) ? preferred : 0
    }
  } catch {
    enabledModels.value = []
  }
}

onMounted(() => {
  createPage.envResult = project.value?.envCheck ?? []
  void loadSceneList()
  if (route.query.new === '1') openWizard()
  void openEnvChat()
})

watch(
  () => createPage.envChatNonce,
  () => {
    void openEnvChat()
  }
)

watch(
  () => route.query.new,
  (value) => {
    if (value === '1') {
      openWizard()
      router.replace({ path: '/project' })
    }
  }
)

watch(
  () => session.currentProjectId,
  () => {
    createPage.envResult = session.currentProject?.envCheck ?? []
  }
)

function onEnvSettled(): void {
  createPage.envBusy = false
  const chat = envChat.value
  if (!chat || !envConvId.value || chat.state.activeId !== envConvId.value) return
  const last = [...chat.state.lines].reverse().find((line) => line.role === 'assistant' && line.content.trim())
  if (!last) return
  const items = parseEnvReport(last.content)
  if (!items.length) return
  const current = createPage.envResult
  if (
    current.length === items.length &&
    current.every((item, index) => item.label === items[index]?.label && item.installed === items[index]?.installed)
  ) {
    return
  }
  void saveEnvCheck(items).then((result) => showResult(result))
}

function openWizard(): void {
  wizardOpen.value = true
  step.value = 1
  error.value = ''
  form.name = ''
  form.localDir = ''
  form.modelId = 0
  form.sceneId = 0
  detailScene.value = null
  customOpen.value = false
  void loadEnabledModels()
  void loadSceneList()
}

async function loadSceneList(): Promise<void> {
  scenesBusy.value = true
  try {
    await reloadScenes()
  } catch (err) {
    if (wizardOpen.value) error.value = messageOf(err)
  } finally {
    scenesBusy.value = false
  }
}

function pickScene(id: number): void {
  form.sceneId = form.sceneId === id ? 0 : id
}

function openCustom(scene?: Scene): void {
  customError.value = ''
  customForm.id = scene?.id ?? 0
  customForm.title = scene?.title ?? ''
  customForm.description = scene?.description ?? ''
  customOpen.value = true
}

async function saveCustom(): Promise<void> {
  customError.value = ''
  customBusy.value = true
  try {
    const scene = await persistScene({
      id: customForm.id || undefined,
      title: customForm.title,
      description: customForm.description
    })
    form.sceneId = scene.id
    customOpen.value = false
  } catch (err) {
    customError.value = messageOf(err)
  } finally {
    customBusy.value = false
  }
}

async function removeScene(scene: Scene): Promise<void> {
  const ok = await confirm({
    title: '删除场景',
    message: `删除「${scene.title}」后，已创建的项目不受影响。`,
    confirmText: '删除',
    danger: true
  })
  if (!ok) return
  const result = await removeCustomScene(scene.id, true)
  showResult(result)
  if (!result.error && form.sceneId === scene.id) form.sceneId = 0
}

async function chooseDir(): Promise<void> {
  const dir = await bridge.pickDirectory()
  if (!dir) return
  form.localDir = dir
  if (!form.name.trim()) {
    form.name = dir.split('/').filter(Boolean).pop() ?? ''
  }
}

async function submitProject(): Promise<void> {
  error.value = ''
  if (!form.name.trim()) {
    error.value = '请填写项目名称'
    step.value = 1
    return
  }
  if (!form.localDir.trim()) {
    error.value = '请选择本地目录'
    step.value = 1
    return
  }
  busy.value = true
  try {
    const result = await createProject({
      name: form.name,
      localDir: form.localDir,
      modelId: form.modelId,
      sceneId: form.sceneId
    })
    if (result.error) {
      error.value = result.error
      return
    }
    showResult(result)
    wizardOpen.value = false
    step.value = 1
  } finally {
    busy.value = false
  }
}

async function openEnvChat(): Promise<void> {
  const nonce = createPage.envChatNonce
  if (!nonce || nonce === createPage.envChatSeen) return
  createPage.envChatSeen = nonce
  const current = session.currentProject
  if (!current?.localDir.trim()) {
    createPage.envBusy = false
    toast.error('这个项目没有本地目录')
    return
  }
  createPage.envBusy = true
  envChatOpen.value = true
  try {
    await nextTick()
    const chat = envChat.value
    if (!chat) throw new Error('检测窗口没有打开')
    envConvId.value = await chat.startEnvCheck()
    if (!chat.state.streaming) onEnvSettled()
  } catch (err) {
    createPage.envBusy = false
    envChatOpen.value = false
    toast.error(messageOf(err))
  }
}

function checkEnvironment(): void {
  const current = project.value
  if (!current?.localDir.trim()) {
    toast.error('这个项目没有本地目录')
    return
  }
  if (createPage.envBusy) {
    toast.show('环境检测还在进行')
    return
  }
  createPage.envBusy = true
  createPage.envChatNonce += 1
}

function openEdit(): void {
  const current = project.value
  if (!current) return
  editForm.name = current.name
  editForm.description = current.description || ''
  editForm.localDir = current.localDir || ''
  editOpen.value = true
}

async function pickEditDir(): Promise<void> {
  const dir = await bridge.pickDirectory()
  if (dir) editForm.localDir = dir
}

async function saveEdit(): Promise<void> {
  editBusy.value = true
  try {
    const result = await updateProject({
      name: editForm.name,
      description: editForm.description,
      localDir: editForm.localDir
    })
    showResult(result)
    if (!result.error) editOpen.value = false
  } finally {
    editBusy.value = false
  }
}

async function openFolder(): Promise<void> {
  const dir = project.value?.localDir.trim() ?? ''
  if (!dir) {
    toast.error('这个项目没有本地目录')
    return
  }
  try {
    await bridge.openExternal(dir)
  } catch (err) {
    toast.error(messageOf(err))
  }
}

async function confirmRemove(): Promise<void> {
  const current = project.value
  if (!current || removeBusy.value) return
  removeBusy.value = true
  try {
    await confirm({
      title: '删除项目',
      subtitle: '任务记录会删除，本机目录保留。',
      message: `删除「${current.name}」后，这个项目下的任务记录会删除。文件夹仍留在原处。`,
      detail: current.localDir,
      confirmText: '删除',
      danger: true,
      busyText: '删除中…',
      action: async () => {
        const result = await deleteProject(true)
        if (result.error) throw new Error(result.error)
        showResult(result)
      }
    })
  } finally {
    removeBusy.value = false
  }
}

onBeforeUnmount(() => {
  if (createPage.envBusy) createPage.envBusy = false
})
</script>

<template>
  <div class="project-page">
    <header class="page-head">
      <div class="col">
        <span class="eyebrow">项目概况</span>
        <h1 class="serif">{{ project?.name || '项目概况' }}</h1>
      </div>
      <div class="head-actions">
        <button class="btn ghost sm" @click="openWizard">新建项目</button>
        <template v-if="project">
          <button class="btn ghost sm" @click="openEdit">编辑项目</button>
          <button class="btn ghost sm" :disabled="createPage.envBusy" @click="checkEnvironment">
            {{ createPage.envBusy ? '检测中…' : '检测开发环境' }}
          </button>
          <button class="btn ghost sm" @click="openFolder">打开文件夹</button>
          <button class="btn danger sm" :disabled="removeBusy" @click="confirmRemove">删除项目</button>
        </template>
      </div>
    </header>

    <div v-if="!project" class="scroll">
      <EmptyState empty-title="还没有项目" empty-hint="先建一个项目，再在这里查看目录、介绍和环境。">
        <button class="btn primary" @click="openWizard">新建项目</button>
      </EmptyState>
    </div>

    <div v-else class="scroll">
      <section class="block">
        <h2 class="serif">介绍</h2>
        <p v-if="project.description" class="prose">{{ project.description }}</p>
        <p v-else class="faint">还没有介绍。</p>
      </section>

      <section class="block">
        <h2 class="serif">目录</h2>
        <p class="mono path">{{ project.localDir || '未设置' }}</p>
      </section>

      <section class="block">
        <h2 class="serif">场景</h2>
        <p v-if="currentScene" class="scene-name">{{ currentScene.title }}</p>
        <p v-else-if="project.scenario" class="scene-name">{{ project.scenario }}</p>
        <p v-else class="faint">未选择场景。</p>
        <p v-if="currentScene?.description" class="prose dim">{{ currentScene.description }}</p>
      </section>

      <section class="block">
        <h2 class="serif">技术</h2>
        <div v-if="project.technologies?.length" class="chips">
          <span v-for="tech in project.technologies" :key="tech" class="chip">{{ tech }}</span>
        </div>
        <p v-else class="faint">还没有记录。</p>
      </section>

      <section class="block">
        <div class="block-head">
          <h2 class="serif">环境</h2>
          <span class="tone" :class="{ ok: envTone === '已齐备' }">{{ envTone }}</span>
        </div>
        <div v-if="envList.length" class="chips">
          <span
            v-for="item in envList"
            :key="item.tech || item.label"
            class="chip"
            :class="{ ok: item.installed, miss: !item.installed }"
            :title="item.installed ? item.version || '已安装' : '未检测到'"
          >
            {{ item.label }}<template v-if="item.version"> {{ item.version }}</template>
          </span>
        </div>
        <p v-else class="faint">还没有检测。点右上角「检测开发环境」。</p>
      </section>
    </div>

    <Dialog
      :open="wizardOpen"
      title="新建项目"
      subtitle="先填基本信息。场景可以不选。"
      width="680px"
      @close="wizardOpen = false"
    >
      <div class="steps">
        <span v-for="(s, i) in ['基本信息', '选择场景']" :key="s" class="step-dot" :class="{ on: step === i + 1 }">
          <span class="n">{{ i + 1 }}</span>{{ s }}
        </span>
      </div>

      <div v-if="step === 1" class="form">
        <FormField label="项目名称">
          <TextField v-model="form.name" placeholder="例如：订单管理后台" />
        </FormField>
        <FormField label="本地目录">
          <div class="row">
            <TextField v-model="form.localDir" class="grow" placeholder="/Users/你/work/项目" spellcheck="false" />
            <button class="btn sm" @click="chooseDir">选择</button>
          </div>
          <span class="tip">还不是 Git 仓库会自动 init，并保证有 main 与 dev。</span>
        </FormField>
        <FormField label="模型">
          <select v-model.number="form.modelId">
            <option :value="0" disabled>选择已启用的模型</option>
            <option v-for="item in enabledModels" :key="item.id" :value="item.id">
              {{ item.providerName }} · {{ item.name }}
            </option>
          </select>
          <span v-if="!enabledModels.length" class="tip">还没有已启用的模型。先到侧栏「模型」里打开供应商并添加模型。</span>
        </FormField>
      </div>

      <div v-else class="form">
        <div class="scenes-bar">
          <span class="tip">场景可以不选。再点一次已选场景可取消。描述较长时点「详情」看全文，也可以保存自己的场景。</span>
          <button class="btn sm" type="button" @click="openCustom()">自定义场景</button>
        </div>
        <p v-if="scenesBusy" class="tip">正在加载场景…</p>
        <p v-else-if="!createPage.scenes.length" class="tip">还没有场景。可以自定义，或请管理员在后台添加。</p>
        <div v-else class="scene-list">
          <div
            v-for="scene in createPage.scenes"
            :key="scene.id"
            class="scene"
            :class="{ on: form.sceneId === scene.id }"
            @click="pickScene(scene.id)"
          >
            <div class="scene-head">
              <span class="scene-title ellipsis">{{ scene.title }}</span>
              <span v-if="scene.userId" class="scene-tag">我的</span>
            </div>
            <p class="scene-desc">{{ scene.description }}</p>
            <div class="scene-actions">
              <button type="button" class="link" @click.stop="detailScene = scene">详情</button>
              <button v-if="scene.userId" type="button" class="link" @click.stop="openCustom(scene)">编辑</button>
              <button v-if="scene.userId" type="button" class="link danger" @click.stop="removeScene(scene)">删除</button>
            </div>
          </div>
        </div>
      </div>

      <p v-if="error" class="err">{{ error }}</p>

      <template #footer>
        <button v-if="step > 1" class="btn ghost" @click="step--">上一步</button>
        <div class="spacer" />
        <button class="btn" @click="wizardOpen = false">取消</button>
        <button v-if="step < 2" class="btn primary" :disabled="!canNext" @click="step++">下一步</button>
        <button v-else class="btn primary" :disabled="busy || !canNext" @click="submitProject">
          {{ busy ? '创建中…' : '创建项目' }}
        </button>
      </template>
    </Dialog>

    <Dialog
      :open="!!detailScene"
      :title="detailScene?.title || '场景详情'"
      subtitle="场景描述"
      width="640px"
      @close="detailScene = null"
    >
      <p class="scene-detail">{{ detailScene?.description }}</p>
    </Dialog>

    <Dialog
      :open="customOpen"
      :title="customForm.id ? '编辑场景' : '自定义场景'"
      subtitle="标题和描述都会保留，下次新建项目还能选到。"
      width="640px"
      :persistent="customBusy"
      @close="customOpen = false"
    >
      <div class="form">
        <FormField label="标题">
          <TextField v-model="customForm.title" placeholder="例如：内部管理系统" />
        </FormField>
        <FormField label="描述">
          <TextArea
            v-model="customForm.description"
            :rows="10"
            placeholder="写清这个场景要做什么、给谁用、有哪些约束。描述可以比较长。"
          />
        </FormField>
      </div>
      <p v-if="customError" class="err">{{ customError }}</p>
      <template #footer>
        <button class="btn ghost" type="button" :disabled="customBusy" @click="customOpen = false">取消</button>
        <div class="spacer" />
        <button class="btn primary" type="button" :disabled="customBusy" @click="saveCustom">
          {{ customBusy ? '保存中…' : '保存并选用' }}
        </button>
      </template>
    </Dialog>

    <Dialog
      :open="editOpen"
      title="编辑项目"
      subtitle="修改名称、介绍和本地目录。"
      width="560px"
      :persistent="editBusy"
      @close="editOpen = false"
    >
      <div class="form">
        <FormField label="名称">
          <TextField v-model="editForm.name" />
        </FormField>
        <FormField label="详细介绍">
          <TextArea v-model="editForm.description" :rows="6" />
        </FormField>
        <FormField label="本地目录">
          <div class="row">
            <TextField v-model="editForm.localDir" class="grow" spellcheck="false" />
            <button class="btn sm" type="button" :disabled="editBusy" @click="pickEditDir">更换</button>
          </div>
        </FormField>
      </div>
      <template #footer>
        <button class="btn ghost" :disabled="editBusy" @click="editOpen = false">取消</button>
        <div class="spacer" />
        <button class="btn primary" :disabled="editBusy" @click="saveEdit">
          {{ editBusy ? '保存中…' : '保存' }}
        </button>
      </template>
    </Dialog>

    <Dialog
      :open="envChatOpen"
      title="检测开发环境"
      subtitle="智能体按场景和代码库检查这台电脑还缺什么。检查结束前不要关掉。"
      width="760px"
      :persistent="envStreaming && envChatOpen"
      @close="envChatOpen = false"
    >
      <div class="task-chat">
        <EnvChat
          v-if="envChatOpen"
          ref="envChat"
          :project-id="session.currentProjectId"
          @settled="onEnvSettled"
        />
      </div>
    </Dialog>
  </div>
</template>

<style scoped>
.project-page {
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
  min-width: 0;
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
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
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
  gap: 22px;
}
.block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.block h2 {
  font-size: 16px;
}
.block-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
}
.prose {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--text);
  white-space: pre-wrap;
}
.prose.dim {
  color: var(--text-dim);
  font-size: 13px;
}
.scene-name {
  margin: 0;
  font-size: 14px;
  color: var(--text-strong);
}
.path {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
  overflow-wrap: anywhere;
}
.faint {
  margin: 0;
  font-size: 14px;
  color: var(--text-faint);
}
.tone {
  font-size: 12px;
  color: var(--text-dim);
}
.tone.ok {
  color: var(--moss);
}
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 8px;
}
.chip {
  font-size: 12px;
  line-height: 1.5;
  color: var(--text);
  padding: 1px 8px;
  border: 1px solid var(--line);
  border-radius: 999px;
}
.chip.ok {
  color: var(--moss);
  border-color: var(--moss-dim);
}
.chip.miss {
  color: var(--rust);
  border-color: var(--rust-dim);
}
.steps {
  display: flex;
  gap: 18px;
  margin-bottom: 18px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--line-soft);
}
.step-dot {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-faint);
}
.step-dot.on {
  color: var(--brass);
}
.n {
  width: 17px;
  height: 17px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  border: 1px solid currentColor;
  font-family: var(--font-mono);
  font-size: 9.5px;
}
.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.row .grow {
  flex: 1;
  min-width: 0;
  width: auto;
}
.tip {
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.6;
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
.task-chat {
  height: min(62vh, 640px);
  margin: -12px -16px -16px;
  display: flex;
  min-width: 0;
  min-height: 0;
}
.scenes-bar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.scenes-bar .tip {
  flex: 1;
}
.scene-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 360px;
  overflow: auto;
}
.scene {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px;
  text-align: left;
  border: 1px solid var(--line);
  border-radius: var(--r-md);
  background: var(--ink-800);
  cursor: pointer;
}
.scene.on {
  border-color: var(--brass-dim);
  background: rgba(200, 160, 90, 0.09);
}
.scene-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.scene-title {
  font-size: 13px;
  color: var(--text-strong);
}
.scene-tag {
  flex: none;
  font-size: 10px;
  color: var(--brass-dim);
  border: 1px solid var(--brass-dim);
  border-radius: 3px;
  padding: 0 4px;
}
.scene-desc {
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-dim);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  white-space: pre-wrap;
}
.scene-actions {
  display: flex;
  gap: 12px;
}
.link {
  padding: 0;
  font-size: 12px;
  color: var(--brass);
}
.link:hover {
  color: var(--text-strong);
}
.link.danger {
  color: #d98d7d;
}
.scene-detail {
  margin: 0;
  white-space: pre-wrap;
  font-size: 13px;
  line-height: 1.75;
  color: var(--text);
  max-height: 60vh;
  overflow: auto;
}
</style>
