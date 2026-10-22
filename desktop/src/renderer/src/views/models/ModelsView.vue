<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import Checkbox from '../../components/Checkbox.vue'
import Dialog from '../../components/Dialog.vue'
import FormField from '../../components/FormField.vue'
import NumberField from '../../components/NumberField.vue'
import Radio from '../../components/Radio.vue'
import Switch from '../../components/Switch.vue'
import TextField from '../../components/TextField.vue'
import { llmApi, type LlmModelRow, type LlmProvider } from '../../api'
import { messageOf, useSessionStore } from '../../stores/session'
import { toast } from '../../composables/toast'
import { builtinModelProviders, matchBuiltin, type BuiltinProvider, type ProviderProtocol } from '../../lib/providerPresets'
import { isProviderEnabled, setProviderEnabled } from '../../lib/providerEnabled'

const session = useSessionStore()
const needsSetup = computed(() => session.modelGateKnown && !session.modelsConfigured)

interface ProviderRow {
  key: string
  name: string
  preset?: BuiltinProvider
  provider?: LlmProvider
}

const providers = ref<LlmProvider[]>([])
const models = ref<LlmModelRow[]>([])
const selectedKey = ref('builtin:deepseek')
const loading = ref(true)
const saving = ref(false)
const err = ref('')

const form = reactive({
  apiKey: '',
  baseUrl: '',
  protocol: 'openai' as ProviderProtocol
})

const testState = reactive<Record<number, { testing: boolean; ok?: boolean; message?: string }>>({})

const customOpen = ref(false)
const customEditing = ref<LlmProvider | null>(null)
const customSaving = ref(false)
const customErr = ref('')
const customForm = reactive({
  name: '',
  website: '',
  apiKey: '',
  protocol: 'openai' as ProviderProtocol,
  baseUrl: ''
})

const modelOpen = ref(false)
const modelSaving = ref(false)
const modelErr = ref('')
const modelTarget = ref<LlmModelRow | null>(null)
const remoteNames = ref<string[]>([])
const modelForm = reactive({
  name: '',
  attachment: false,
  reasoning: false,
  temperature: false,
  inputContextLimit: 200000,
  outputContextLimit: 0,
  asDefault: false
})

const rows = computed<ProviderRow[]>(() => {
  const used = new Set<number>()
  const builtins = builtinModelProviders.map((preset) => {
    const matches = providers.value.filter(
      (item) => item.id && !used.has(item.id) && matchBuiltin(item.name)?.key === preset.key
    )
    const provider = matches.find((item) => item.editable)
    for (const item of matches) {
      if (item.id) used.add(item.id)
    }
    return { key: `builtin:${preset.key}`, name: preset.name, preset, provider }
  })
  const custom = providers.value
    .filter((item) => item.id && !used.has(item.id))
    .map((provider) => ({ key: `custom:${provider.id}`, name: provider.name, provider }))
  return [...builtins, ...custom]
})

const selectedRow = computed(() => rows.value.find((item) => item.key === selectedKey.value) ?? rows.value[0])
const selected = computed(() => selectedRow.value?.provider)
const formEditable = computed(() => !selected.value || selected.value.editable)
const selectedModels = computed(() => models.value.filter((item) => item.providerId === selected.value?.id))
const selectedEnabled = computed(() => (selected.value?.id ? isProviderEnabled(selected.value.id) : false))
const currentTest = computed(() => (selected.value?.id ? testState[selected.value.id] : undefined))

function markOf(item: ProviderRow): string {
  return item.preset?.icon || matchBuiltin(item.name)?.icon || ''
}

function markPlate(item: ProviderRow): boolean {
  const key = item.preset?.key || matchBuiltin(item.name)?.key
  return key === 'moonshot'
}

function protocolOf(item: LlmProvider): ProviderProtocol {
  return item.anthropicBaseUrl && !item.openaiCompatibleBaseUrl ? 'anthropic' : 'openai'
}

function baseUrlOf(item: LlmProvider): string {
  return item.openaiCompatibleBaseUrl || item.anthropicBaseUrl || ''
}

function hydrate(item: LlmProvider | undefined, preset?: BuiltinProvider): void {
  if (item) {
    form.apiKey = item.editable ? item.apiKey : ''
    form.baseUrl = baseUrlOf(item)
    form.protocol = protocolOf(item)
    return
  }
  form.apiKey = ''
  form.baseUrl = preset?.baseUrl ?? ''
  form.protocol = preset?.protocol ?? 'openai'
}

function selectRow(key: string): void {
  selectedKey.value = key
  const row = rows.value.find((item) => item.key === key)
  hydrate(row?.provider, row?.preset)
  err.value = ''
}

async function load(): Promise<void> {
  loading.value = true
  err.value = ''
  try {
    const [providerData, modelData] = await Promise.all([llmApi.listProviders(), llmApi.listModels()])
    providers.value = providerData.list ?? []
    models.value = modelData.list ?? []
    hydrate(selectedRow.value?.provider, selectedRow.value?.preset)
    await session.refreshModelGate()
  } catch (e) {
    err.value = messageOf(e)
  } finally {
    loading.value = false
  }
}

function payloadFrom(name: string, website: string, apiKey: string, protocol: ProviderProtocol, baseUrl: string) {
  return {
    name,
    website,
    apiKey,
    openaiCompatibleBaseUrl: protocol === 'openai' ? baseUrl : '',
    anthropicBaseUrl: protocol === 'anthropic' ? baseUrl : ''
  }
}

async function materialize(row: ProviderRow): Promise<LlmProvider | undefined> {
  if (row.provider?.id) return row.provider
  if (!row.preset) return undefined
  const created = await llmApi.createProvider(
    payloadFrom(row.preset.name, row.preset.website, form.apiKey.trim(), form.protocol, form.baseUrl.trim())
  )
  setProviderEnabled(created.id, true)
  await load()
  selectedKey.value = `builtin:${row.preset.key}`
  return providers.value.find((item) => item.id === created.id)
}

async function persist(): Promise<boolean> {
  const row = selectedRow.value
  if (!row) return false
  if (row.provider && !row.provider.editable) return true
  if (!row.provider && !form.apiKey.trim()) return true
  saving.value = true
  err.value = ''
  try {
    if (!row.provider) {
      await materialize(row)
    } else {
      await llmApi.updateProvider(
        row.provider.id,
        payloadFrom(row.provider.name, row.provider.website, form.apiKey.trim(), form.protocol, form.baseUrl.trim())
      )
      await load()
    }
    return true
  } catch (e) {
    err.value = messageOf(e)
    return false
  } finally {
    saving.value = false
  }
}

async function toggleRow(row: ProviderRow): Promise<void> {
  err.value = ''
  try {
    let provider = row.provider
    if (!provider?.id) {
      if (!form.apiKey.trim() && row.key === selectedKey.value) {
        err.value = '先填写 API Key'
        return
      }
      if (row.key !== selectedKey.value) selectRow(row.key)
      provider = await materialize(row)
    }
    if (!provider?.id) return
    setProviderEnabled(provider.id, !isProviderEnabled(provider.id))
    await session.refreshModelGate()
  } catch (e) {
    err.value = messageOf(e)
  }
}

async function testProvider(): Promise<void> {
  const row = selectedRow.value
  if (!row) return
  const ok = row.provider?.id ? await persist() : true
  if (!ok) return
  let provider = row.provider
  if (!provider?.id) provider = await materialize(row)
  if (!provider?.id) return
  testState[provider.id] = { testing: true }
  try {
    await llmApi.remoteModels(provider.id)
    testState[provider.id] = { testing: false, ok: true, message: '连接成功' }
  } catch (e) {
    testState[provider.id] = { testing: false, ok: false, message: messageOf(e) }
  }
}

function openCustom(item?: LlmProvider): void {
  customEditing.value = item ?? null
  customErr.value = ''
  customForm.name = item?.name ?? ''
  customForm.website = item?.website ?? ''
  customForm.apiKey = item?.apiKey ?? ''
  customForm.protocol = item ? protocolOf(item) : 'openai'
  customForm.baseUrl = item ? baseUrlOf(item) : ''
  customOpen.value = true
}

async function saveCustom(): Promise<void> {
  const name = customForm.name.trim()
  if (!name) {
    customErr.value = '请填写供应商名称'
    return
  }
  if (!customForm.baseUrl.trim()) {
    customErr.value = '请填写接口地址'
    return
  }
  customSaving.value = true
  customErr.value = ''
  try {
    const body = payloadFrom(name, customForm.website.trim(), customForm.apiKey.trim(), customForm.protocol, customForm.baseUrl.trim())
    if (customEditing.value?.id) {
      await llmApi.updateProvider(customEditing.value.id, body)
      selectedKey.value = `custom:${customEditing.value.id}`
    } else {
      const created = await llmApi.createProvider(body)
      setProviderEnabled(created.id, true)
      selectedKey.value = `custom:${created.id}`
    }
    customOpen.value = false
    await load()
  } catch (e) {
    customErr.value = messageOf(e)
  } finally {
    customSaving.value = false
  }
}

async function removeProvider(item: LlmProvider): Promise<void> {
  if (!item.editable || matchBuiltin(item.name)) return
  if (!window.confirm(`删除供应商「${item.name}」？其下的模型也会一并删除。`)) return
  try {
    await llmApi.deleteProvider(item.id)
    selectedKey.value = 'builtin:deepseek'
    await load()
  } catch (e) {
    err.value = messageOf(e)
  }
}

async function openModel(item?: LlmModelRow): Promise<void> {
  const provider = selected.value
  if (!provider?.id || !provider.editable) {
    err.value = '请先启用并可编辑的供应商'
    return
  }
  modelTarget.value = item ?? null
  modelErr.value = ''
  modelForm.name = item?.name ?? ''
  modelForm.attachment = item?.attachment ?? false
  modelForm.reasoning = item?.reasoning ?? false
  modelForm.temperature = item?.temperature ?? false
  modelForm.inputContextLimit = item?.inputContextLimit || 200000
  modelForm.outputContextLimit = item?.outputContextLimit || 0
  const currentDefault = session.profile.modelId
  const hasDefault = models.value.some((row) => row.id === currentDefault)
  modelForm.asDefault = !hasDefault || item?.id === currentDefault
  modelOpen.value = true
  remoteNames.value = []
  try {
    remoteNames.value = (await llmApi.remoteModels(provider.id)).list ?? []
  } catch {
    remoteNames.value = []
  }
}

async function saveModel(): Promise<void> {
  const provider = selected.value
  if (!provider?.id) return
  if (!modelForm.name.trim()) {
    modelErr.value = '请填写模型 id'
    return
  }
  modelSaving.value = true
  modelErr.value = ''
  const body = {
    providerId: provider.id,
    name: modelForm.name.trim(),
    attachment: modelForm.attachment,
    reasoning: modelForm.reasoning,
    temperature: modelForm.temperature,
    inputContextLimit: modelForm.inputContextLimit,
    outputContextLimit: modelForm.outputContextLimit
  }
  try {
    const saved = modelTarget.value?.id
      ? await llmApi.updateModel(modelTarget.value.id, body)
      : await llmApi.createModel(body)
    if (modelForm.asDefault) await session.setDefaultModel(saved.id)
    else if (session.profile.modelId === saved.id) await session.setDefaultModel(0)
    modelOpen.value = false
    await load()
    toast.success('模型已保存')
  } catch (e) {
    modelErr.value = messageOf(e)
  } finally {
    modelSaving.value = false
  }
}

async function makeDefault(item: LlmModelRow): Promise<void> {
  try {
    await session.setDefaultModel(item.id)
    toast.success('已设为默认模型')
  } catch (e) {
    err.value = messageOf(e)
  }
}

async function removeModel(item: LlmModelRow): Promise<void> {
  if (!item.editable) return
  if (!window.confirm(`删除模型「${item.name}」？`)) return
  try {
    await llmApi.deleteModel(item.id)
    if (session.profile.modelId === item.id) await session.setDefaultModel(0)
    await load()
  } catch (e) {
    err.value = messageOf(e)
  }
}

onMounted(load)
</script>

<template>
  <div class="models">
    <header class="page-head">
      <div class="col">
        <span class="eyebrow">模型</span>
        <h1 class="serif">供应商与模型</h1>
        <p v-if="needsSetup" class="lede">还没有可用模型。按下面三步配好之后，才能创建项目和让 Agent 开工。</p>
        <ol v-if="needsSetup" class="steps">
          <li>在左侧启用一个供应商。</li>
          <li>填写 API Key 和接口地址，离开输入框即保存。</li>
          <li>在右侧添加至少一个模型。</li>
        </ol>
        <p v-else class="lede">密钥存在你的账号下。平台供应商只展示，不能改。</p>
      </div>
    </header>

    <div class="studio">
      <aside class="list">
        <div class="list-head"><h2>模型提供商</h2></div>
        <p v-if="loading" class="quiet">正在加载…</p>
        <ul class="items">
          <li v-for="item in rows" :key="item.key" :class="{ active: item.key === selectedRow?.key }">
            <button type="button" class="row" @click="selectRow(item.key)">
              <img v-if="markOf(item)" class="mark" :class="{ plate: markPlate(item) }" :src="markOf(item)" alt="" />
              <span v-else class="glyph">{{ item.name.slice(0, 1) }}</span>
              <span class="pname ellipsis">{{ item.name }}</span>
            </button>
            <Switch
              :model-value="!!(item.provider?.id && isProviderEnabled(item.provider.id))"
              :label="`${item.provider?.id && isProviderEnabled(item.provider.id) ? '停用' : '启用'} ${item.name}`"
              @update:model-value="toggleRow(item)"
            />
          </li>
          <li>
            <button type="button" class="custom-add" @click="openCustom()">＋ 自定义</button>
          </li>
        </ul>
      </aside>

      <section v-if="selectedRow" class="detail">
        <header class="detail-head">
          <h2>{{ selectedRow.name }}</h2>
          <span class="chips">
            <em v-if="selected && !selected.editable" class="badge">平台</em>
            <em class="badge" :class="{ off: !selectedEnabled }">{{ selectedEnabled ? '已启用' : '已停用' }}</em>
          </span>
        </header>
        <p v-if="err" class="err">{{ err }}</p>
        <p v-else-if="saving" class="quiet">保存中…</p>

        <FormField label="API Key">
          <TextField
            v-model="form.apiKey"
            type="password"
            autocomplete="off"
            placeholder="输入 API Key"
            spellcheck="false"
            :disabled="!formEditable"
            @blur="persist"
          />
        </FormField>
        <FormField label="API Base URL">
          <TextField v-model="form.baseUrl" spellcheck="false" :disabled="!formEditable" @blur="persist" />
        </FormField>
        <FormField label="API 格式">
          <div class="radios">
            <Radio v-model="form.protocol" value="anthropic" :disabled="!formEditable" @change="persist">Anthropic 兼容</Radio>
            <Radio v-model="form.protocol" value="openai" :disabled="!formEditable" @change="persist">OpenAI 兼容</Radio>
          </div>
        </FormField>

        <div class="test-row">
          <button class="btn sm" :disabled="currentTest?.testing" @click="testProvider">
            {{ currentTest?.testing ? '测试中…' : '测试连接' }}
          </button>
          <span v-if="currentTest && !currentTest.testing" :class="currentTest.ok ? 'ok' : 'fail'">
            {{ currentTest.ok ? '连接成功' : currentTest.message }}
          </span>
        </div>

        <div class="models-head">
          <h3>可用模型</h3>
          <button v-if="formEditable" class="btn ghost sm" @click="openModel()">添加模型</button>
        </div>
        <ul class="model-list">
          <li v-for="item in selectedModels" :key="item.id" class="model">
            <button class="model-name" :disabled="!item.editable" @click="item.editable && openModel(item)">
              {{ item.name }}
              <em v-if="item.id === session.profile.modelId" class="badge">默认</em>
            </button>
            <button
              v-if="item.id !== session.profile.modelId"
              class="btn ghost sm"
              @click="makeDefault(item)"
            >
              设为默认
            </button>
            <button v-if="item.editable" class="btn ghost sm" @click="removeModel(item)">删除</button>
          </li>
          <li v-if="!selectedModels.length" class="quiet">还没有模型</li>
        </ul>

        <div v-if="selected?.editable && !selectedRow.preset" class="extra">
          <button class="btn ghost sm" @click="openCustom(selected)">编辑名称</button>
          <button class="btn ghost sm danger" @click="removeProvider(selected)">删除供应商</button>
        </div>
      </section>
    </div>

    <Dialog :open="customOpen" :title="customEditing ? '编辑供应商' : '自定义供应商'" @close="customOpen = false">
      <div class="form">
        <p v-if="customErr" class="err">{{ customErr }}</p>
        <FormField label="名称"><TextField v-model="customForm.name" /></FormField>
        <FormField label="API Key"><TextField v-model="customForm.apiKey" type="password" /></FormField>
        <FormField label="API Base URL"><TextField v-model="customForm.baseUrl" spellcheck="false" /></FormField>
        <FormField label="API 格式">
          <div class="radios">
            <Radio v-model="customForm.protocol" value="anthropic">Anthropic 兼容</Radio>
            <Radio v-model="customForm.protocol" value="openai">OpenAI 兼容</Radio>
          </div>
        </FormField>
        <FormField label="网站"><TextField v-model="customForm.website" spellcheck="false" /></FormField>
      </div>
      <template #footer>
        <button class="btn ghost" @click="customOpen = false">取消</button>
        <div class="spacer" />
        <button class="btn primary" :disabled="customSaving" @click="saveCustom">保存</button>
      </template>
    </Dialog>

    <Dialog :open="modelOpen" :title="modelTarget ? '编辑模型' : '添加模型'" @close="modelOpen = false">
      <div class="form">
        <p v-if="modelErr" class="err">{{ modelErr }}</p>
        <FormField label="模型 id">
          <TextField v-model="modelForm.name" list="remote-models" spellcheck="false" />
          <datalist id="remote-models">
            <option v-for="name in remoteNames" :key="name" :value="name" />
          </datalist>
        </FormField>
        <Checkbox v-model="modelForm.attachment">支持附件</Checkbox>
        <Checkbox v-model="modelForm.reasoning">支持推理</Checkbox>
        <Checkbox v-model="modelForm.temperature">支持温度</Checkbox>
        <FormField label="输入上下文">
          <NumberField v-model="modelForm.inputContextLimit" :min="0" />
        </FormField>
        <FormField label="输出上下文">
          <NumberField v-model="modelForm.outputContextLimit" :min="0" />
        </FormField>
        <Checkbox v-model="modelForm.asDefault">设为默认模型</Checkbox>
      </div>
      <template #footer>
        <button class="btn ghost" @click="modelOpen = false">取消</button>
        <div class="spacer" />
        <button class="btn primary" :disabled="modelSaving" @click="saveModel">保存</button>
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
.models {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.page-head {
  padding: 22px 26px 14px;
  border-bottom: 1px solid var(--line-soft);
}
.eyebrow {
  font-size: 10.5px;
  letter-spacing: 0.18em;
  color: var(--brass-dim);
}
h1 { font-size: 21px; }
.lede { margin-top: 6px; font-size: 12px; color: var(--text-dim); }
.steps {
  margin: 10px 0 0;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--text);
}
.studio {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
}
.list {
  border-right: 1px solid var(--line-soft);
  overflow: auto;
  padding: 14px 10px 20px;
}
.list-head h2, .models-head h3 { font-size: 13px; margin: 0; }
.items { list-style: none; margin: 10px 0 0; padding: 0; display: flex; flex-direction: column; gap: 2px; }
.items li, .row, .custom-add { display: flex; align-items: center; gap: 8px; }
.items li.active { background: var(--ink-700); border-radius: var(--r-sm); }
.row, .custom-add {
  flex: 1;
  min-width: 0;
  height: 34px;
  padding: 0 8px;
  color: var(--text);
  text-align: left;
}
.mark {
  width: 18px;
  height: 18px;
  flex: none;
  object-fit: contain;
}
.mark.plate {
  width: 22px;
  height: 22px;
  padding: 2px;
  border-radius: 5px;
  background: #111318;
  box-sizing: border-box;
}
.glyph {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  display: grid;
  place-items: center;
  background: var(--ink-600);
  color: var(--text-strong);
  font-size: 12px;
}
.items :deep(.switch) { margin-right: 6px; }
.detail { overflow: auto; padding: 18px 26px 32px; display: flex; flex-direction: column; gap: 12px; }
.detail-head, .models-head, .test-row, .extra { display: flex; align-items: center; gap: 10px; }
.badge {
  font-style: normal;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--moss-dim);
  color: var(--moss);
}
.badge.off { color: var(--text-faint); border-color: var(--line); }
.field { display: flex; flex-direction: column; gap: 6px; }
.k { font-size: 11px; color: var(--text-faint); }
input, textarea {
  background: var(--ink-850);
  color: var(--text-strong);
  border: 1px solid var(--line);
  border-radius: var(--r-sm);
  padding: 7px 9px;
  font: inherit;
  font-size: 12.5px;
}
.radios, .check { display: flex; gap: 14px; font-size: 12.5px; color: var(--text); }
.quiet { color: var(--text-faint); font-size: 12px; }
.err { color: var(--rust); font-size: 12px; }
.ok { color: var(--moss); font-size: 12px; }
.fail { color: var(--rust); font-size: 12px; }
.model-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; }
.model { display: flex; align-items: center; gap: 8px; }
.model-name {
  flex: 1;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  text-align: left;
  color: var(--text-strong);
  font-family: var(--font-mono);
  font-size: 12px;
}
.form { display: flex; flex-direction: column; gap: 10px; }
</style>
