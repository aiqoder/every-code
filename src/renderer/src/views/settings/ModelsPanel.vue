<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import UiButton from '@/components/ui/UiButton.vue'
import UiDialog from '@/components/ui/UiDialog.vue'
import UiField from '@/components/ui/UiField.vue'
import UiSwitch from '@/components/ui/UiSwitch.vue'
import { useDialog, useMessage } from '@/components/ui/feedback'
import {
  createLlmProvider,
  deleteLlmModel,
  deleteLlmProvider,
  listAllLlmModels,
  listLlmProviders,
  listRemoteModels,
  updateLlmProvider,
  type LlmModel,
  type LlmProvider,
  type ProviderPayload
} from '@/api/llm'
import { httpUrl, requiredText } from '@/lib/formCheck'
import { modelCode, modelTitle } from '@/lib/modelLabel'
import { isProviderEnabled, setProviderEnabled } from '@/lib/providerEnabled'
import {
  builtinModelProviders,
  matchBuiltin,
  type BuiltinProvider,
  type ProviderProtocol
} from '@/lib/providerPresets'
import { useSessionStore } from '@/stores/sessionStore'
import ModelConfigDialog from './components/ModelConfigDialog.vue'
import ProviderMark from './components/ProviderMark.vue'

interface ProviderRow {
  key: string
  name: string
  preset?: BuiltinProvider
  provider?: LlmProvider
}

interface TestState {
  testing: boolean
  ok?: boolean
  message?: string
}

const session = useSessionStore()
const dialog = useDialog()
const message = useMessage()
const providers = ref<LlmProvider[]>([])
const models = ref<LlmModel[]>([])
const selectedKey = ref('builtin:deepseek')
const err = ref('')
const fieldError = ref('')
const saving = ref(false)
const saved = ref(false)
const loading = ref(true)

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
const modelProviderId = ref(0)
const modelProviderName = ref('')
const modelTarget = ref<LlmModel | null>(null)

const form = reactive({
  apiKey: '',
  baseUrl: '',
  protocol: 'openai' as ProviderProtocol
})
const testState = reactive<Record<number, TestState>>({})
let savedTimer = 0

const rows = computed<ProviderRow[]>(() => {
  const used = new Set<number>()
  const builtins = builtinModelProviders.map((preset) => {
    const provider = providers.value.find(
      (item) => !!item.id && !used.has(item.id) && matchBuiltin(item.name)?.key === preset.key
    )
    if (provider?.id) used.add(provider.id)
    return { key: `builtin:${preset.key}`, name: preset.name, preset, provider }
  })
  const custom = providers.value
    .filter((item) => !!item.id && !used.has(item.id))
    .map((provider) => ({ key: `custom:${provider.id}`, name: provider.name, provider }))
  return [...builtins, ...custom]
})

const selectedRow = computed(
  () => rows.value.find((item) => item.key === selectedKey.value) || rows.value[0] || null
)
const selected = computed(() => selectedRow.value?.provider || null)
const formSource = computed(() =>
  selected.value || (selectedRow.value?.preset ? virtualProvider(selectedRow.value.preset) : null)
)
const selectedModels = computed(() => models.value.filter((item) => item.providerId === selected.value?.id))
const selectedEnabled = computed(() => (selected.value?.id ? isProviderEnabled(selected.value.id) : false))
const formEditable = computed(() => !selected.value || selected.value.editable)
const currentTest = computed(() => (selected.value?.id ? testState[selected.value.id] : undefined))
const urlInvalid = computed(
  () => fieldError.value === '请输入接口地址' || fieldError.value === '请输入有效的 URL'
)

watch(
  () => session.loggedIn,
  (loggedIn, wasLoggedIn) => {
    if (loggedIn && !wasLoggedIn) void load()
  }
)

onMounted(() => {
  hydrate(formSource.value)
  void load()
})

onBeforeUnmount(() => {
  window.clearTimeout(savedTimer)
  void persist()
})

function virtualProvider(preset: BuiltinProvider): LlmProvider {
  return {
    id: 0,
    name: preset.name,
    website: preset.website,
    apiKey: '',
    openaiCompatibleBaseUrl: preset.protocol === 'openai' ? preset.baseUrl : '',
    anthropicBaseUrl: preset.protocol === 'anthropic' ? preset.baseUrl : '',
    editable: true
  }
}

function findBuiltinProvider(key: string): LlmProvider | undefined {
  return providers.value.find((item) => matchBuiltin(item.name)?.key === key)
}

function protocolOf(item: LlmProvider): ProviderProtocol {
  return item.anthropicBaseUrl && !item.openaiCompatibleBaseUrl ? 'anthropic' : 'openai'
}

function baseUrlOf(item: LlmProvider): string {
  return item.openaiCompatibleBaseUrl || item.anthropicBaseUrl || ''
}

function rowOn(row: ProviderRow): boolean {
  return !!(row.provider?.id && isProviderEnabled(row.provider.id))
}

function hydrate(item: LlmProvider | null): void {
  if (!item) return
  form.apiKey = item.apiKey || ''
  form.baseUrl = baseUrlOf(item)
  form.protocol = protocolOf(item)
  fieldError.value = ''
  saved.value = false
}

function payloadOf(item: LlmProvider): ProviderPayload {
  const base = form.baseUrl.trim()
  return {
    name: item.name,
    website: item.website || '',
    apiKey: form.apiKey.trim(),
    openaiCompatibleBaseUrl: form.protocol === 'openai' ? base : '',
    anthropicBaseUrl: form.protocol === 'anthropic' ? base : ''
  }
}

function isDirty(item: LlmProvider): boolean {
  const next = payloadOf(item)
  return (
    next.apiKey !== (item.apiKey || '') ||
    next.openaiCompatibleBaseUrl !== (item.openaiCompatibleBaseUrl || '') ||
    next.anthropicBaseUrl !== (item.anthropicBaseUrl || '')
  )
}

function flashSaved(): void {
  saved.value = true
  window.clearTimeout(savedTimer)
  savedTimer = window.setTimeout(() => {
    saved.value = false
  }, 1600)
}

function baseError(): string {
  return requiredText(form.baseUrl, '请输入接口地址') || httpUrl(form.baseUrl, '请输入有效的 URL')
}

async function refreshLists(): Promise<void> {
  const [nextProviders, nextModels] = await Promise.all([listLlmProviders(), listAllLlmModels()])
  providers.value = nextProviders
  models.value = nextModels
}

async function materialize(row: ProviderRow, enable: boolean): Promise<LlmProvider | null> {
  if (row.provider?.id) {
    setProviderEnabled(row.provider.id, enable)
    return row.provider
  }
  const preset = row.preset
  if (!preset) return null
  const fromForm = selectedKey.value === row.key
  const protocol = fromForm ? form.protocol : preset.protocol
  const baseUrl = (fromForm ? form.baseUrl : preset.baseUrl).trim()
  const apiKey = fromForm ? form.apiKey.trim() : ''
  const invalid = requiredText(baseUrl, '请输入接口地址') || httpUrl(baseUrl, '请输入有效的 URL')
  if (invalid) {
    fieldError.value = invalid
    return null
  }
  fieldError.value = ''
  saving.value = true
  err.value = ''
  try {
    await createLlmProvider({
      name: preset.name,
      website: preset.website,
      apiKey,
      openaiCompatibleBaseUrl: protocol === 'openai' ? baseUrl : '',
      anthropicBaseUrl: protocol === 'anthropic' ? baseUrl : ''
    })
    await refreshLists()
    const created = findBuiltinProvider(preset.key)
    if (!created?.id) {
      err.value = '供应商已保存，但列表里没有找到它'
      return null
    }
    setProviderEnabled(created.id, enable)
    if (selectedKey.value === row.key) hydrate(created)
    flashSaved()
    return created
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
    return null
  } finally {
    saving.value = false
  }
}

async function persist(): Promise<boolean> {
  const row = rows.value.find((item) => item.key === selectedKey.value)
  if (!row) return true
  if (!row.provider) {
    const preset = row.preset
    if (!preset) return true
    if (!isDirty(virtualProvider(preset))) return true
    return !!(await materialize(row, false))
  }
  const item = row.provider
  if (!item.editable || !item.id) return true
  if (!isDirty(item)) return true
  const invalid = baseError()
  if (invalid) {
    fieldError.value = invalid
    return false
  }
  fieldError.value = ''
  saving.value = true
  err.value = ''
  try {
    const payload = payloadOf(item)
    await updateLlmProvider(item.id, payload)
    item.apiKey = payload.apiKey
    item.openaiCompatibleBaseUrl = payload.openaiCompatibleBaseUrl
    item.anthropicBaseUrl = payload.anthropicBaseUrl
    flashSaved()
    return true
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
    return false
  } finally {
    saving.value = false
  }
}

async function load(): Promise<void> {
  if (!session.loggedIn) {
    loading.value = false
    err.value = '请先登录后再配置模型'
    return
  }
  const dirty = formSource.value ? isDirty(formSource.value) : false
  err.value = ''
  loading.value = true
  try {
    await refreshLists()
    if (!rows.value.some((item) => item.key === selectedKey.value)) {
      selectedKey.value = rows.value[0]?.key || 'builtin:deepseek'
    }
    if (!dirty) hydrate(formSource.value)
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

async function selectRow(key: string): Promise<void> {
  if (key === selectedKey.value) return
  const ok = await persist()
  if (!ok) return
  selectedKey.value = key
  hydrate(formSource.value)
}

async function toggleRow(row: ProviderRow): Promise<void> {
  if (row.key !== selectedKey.value) {
    const ok = await persist()
    if (!ok) return
    selectedKey.value = row.key
    hydrate(formSource.value)
  }
  const current = rows.value.find((item) => item.key === row.key) || row
  const enable = !rowOn(current)
  if (!current.provider) {
    if (enable) await materialize(current, true)
    return
  }
  if (current.provider.id) setProviderEnabled(current.provider.id, enable)
}

function openCustom(item?: LlmProvider): void {
  customEditing.value = item ?? null
  customForm.name = item?.name || ''
  customForm.website = item?.website || ''
  customForm.apiKey = item?.apiKey || ''
  customForm.protocol = item ? protocolOf(item) : 'openai'
  customForm.baseUrl = item ? baseUrlOf(item) : ''
  customErr.value = ''
  customOpen.value = true
}

function closeCustom(): void {
  if (customSaving.value) return
  customOpen.value = false
}

function onCustomShow(value: boolean): void {
  if (!value) closeCustom()
  else customOpen.value = true
}

async function saveCustom(): Promise<void> {
  const name = customForm.name.trim()
  if (!name) {
    customErr.value = '请输入供应商名称'
    return
  }
  const builtin = matchBuiltin(name)
  const editingBuiltin = customEditing.value ? matchBuiltin(customEditing.value.name) : undefined
  if (builtin && builtin.key !== editingBuiltin?.key) {
    customErr.value = '该名称是内置供应商，请直接在左侧列表中开启'
    return
  }
  const duplicate = providers.value.find(
    (item) => item.name.trim().toLowerCase() === name.toLowerCase() && item.id !== customEditing.value?.id
  )
  if (duplicate) {
    customErr.value = '已有同名供应商'
    return
  }
  const invalid =
    requiredText(customForm.baseUrl, '请输入接口地址') || httpUrl(customForm.baseUrl, '请输入有效的 URL')
  if (invalid) {
    customErr.value = invalid
    return
  }
  const websiteInvalid = httpUrl(customForm.website, '请输入有效的网站地址')
  if (websiteInvalid) {
    customErr.value = websiteInvalid
    return
  }
  customSaving.value = true
  customErr.value = ''
  const payload: ProviderPayload = {
    name,
    website: customForm.website.trim(),
    apiKey: customForm.apiKey.trim(),
    openaiCompatibleBaseUrl: customForm.protocol === 'openai' ? customForm.baseUrl.trim() : '',
    anthropicBaseUrl: customForm.protocol === 'anthropic' ? customForm.baseUrl.trim() : ''
  }
  try {
    if (customEditing.value?.id) {
      await updateLlmProvider(customEditing.value.id, payload)
      await refreshLists()
      selectedKey.value = `custom:${customEditing.value.id}`
    } else {
      await createLlmProvider(payload)
      await refreshLists()
      const created = [...providers.value]
        .reverse()
        .find((item) => item.name === name && !matchBuiltin(item.name))
      if (created?.id) {
        setProviderEnabled(created.id, true)
        selectedKey.value = `custom:${created.id}`
      }
    }
    hydrate(formSource.value)
    customOpen.value = false
  } catch (e) {
    customErr.value = e instanceof Error ? e.message : String(e)
  } finally {
    customSaving.value = false
  }
}

function removeProvider(item: LlmProvider): void {
  if (!item.editable || !item.id || matchBuiltin(item.name)) return
  dialog.warning({
    title: '删除供应商',
    content: `删除供应商「${item.name}」？其下的模型也会一并删除。`,
    positiveText: '删除',
    negativeText: '取消',
    danger: true,
    onPositiveClick: async () => {
      try {
        await deleteLlmProvider(item.id)
        if (selected.value?.id === item.id) selectedKey.value = 'builtin:deepseek'
        await load()
      } catch (e) {
        const text = e instanceof Error ? e.message : String(e)
        err.value = text
        message.error(text)
      }
    }
  })
}

async function onProtocol(next: ProviderProtocol): Promise<void> {
  form.protocol = next
  fieldError.value = ''
  await persist()
}

async function testProvider(): Promise<void> {
  const row = rows.value.find((item) => item.key === selectedKey.value)
  if (!row) return
  let provider = row.provider
  if (!provider?.id) {
    if (!form.apiKey.trim()) {
      fieldError.value = '请先填写 API Key'
      return
    }
    provider = (await materialize(row, false)) || undefined
    if (!provider?.id) return
  } else if (!provider.editable) {
    fieldError.value = '平台供应商由服务端维护，不能在这里测试'
    return
  } else {
    const ok = await persist()
    if (!ok) return
  }
  testState[provider.id] = { testing: true }
  try {
    await listRemoteModels(provider.id)
    testState[provider.id] = { testing: false, ok: true, message: '连接成功' }
  } catch (e) {
    testState[provider.id] = {
      testing: false,
      ok: false,
      message: e instanceof Error ? e.message : String(e)
    }
  }
}

async function openModelDialog(model?: LlmModel): Promise<void> {
  const row = rows.value.find((item) => item.key === selectedKey.value)
  if (!row) return
  const provider = row.provider
  if (!provider?.id) {
    fieldError.value = '请先开启该供应商'
    return
  }
  if (!provider.editable) return
  const ok = await persist()
  if (!ok) return
  modelProviderId.value = provider.id
  modelProviderName.value = row.name
  modelTarget.value = model ?? null
  modelOpen.value = true
}

function removeModel(item: LlmModel): void {
  if (!item.editable || !item.id) return
  dialog.warning({
    title: '删除模型',
    content: `删除模型「${item.name}」？`,
    positiveText: '删除',
    negativeText: '取消',
    danger: true,
    onPositiveClick: async () => {
      try {
        await deleteLlmModel(item.id)
        await load()
      } catch (e) {
        const text = e instanceof Error ? e.message : String(e)
        err.value = text
        message.error(text)
      }
    }
  })
}
</script>

<template>
  <div class="flex h-full min-h-0">
    <aside class="flex w-[272px] shrink-0 flex-col border-r border-[var(--ec-border)]">
      <h2 class="shrink-0 px-4 pt-4 pb-3 text-sm font-semibold">模型提供商</h2>
      <ul class="min-h-0 flex-1 space-y-2 overflow-auto px-3 pb-4">
        <li
          v-for="item in rows"
          :key="item.key"
          class="flex items-center gap-1 rounded-xl border px-1.5 py-1.5"
          :class="
            item.key === selectedRow?.key
              ? 'border-transparent bg-[color-mix(in_srgb,var(--ec-accent)_16%,transparent)]'
              : 'border-[var(--ec-border)] bg-[var(--ec-canvas)]'
          "
        >
          <button
            type="button"
            class="flex min-w-0 flex-1 items-center gap-2.5 rounded-lg px-1.5 py-1 text-left"
            @click="selectRow(item.key)"
          >
            <ProviderMark :name="item.name" />
            <span class="truncate text-sm font-medium">{{ item.name }}</span>
          </button>
          <UiSwitch
            size="small"
            :value="rowOn(item)"
            :disabled="saving"
            :aria-label="`${rowOn(item) ? '停用' : '启用'} ${item.name}`"
            @update:value="toggleRow(item)"
          />
        </li>
        <li>
          <button
            type="button"
            class="flex w-full items-center gap-2.5 rounded-xl border border-dashed border-[var(--border-strong)] px-3 py-2.5 text-left text-sm font-medium"
            @click="openCustom()"
          >
            <span
              class="grid h-[22px] w-[22px] place-items-center rounded-[7px] bg-[var(--ec-canvas-inset)] text-[var(--ec-fg-muted)]"
              >+</span
            >
            自定义
          </button>
        </li>
      </ul>
    </aside>

    <section v-if="selectedRow" class="min-w-0 flex-1 overflow-auto px-8 py-4">
      <header class="flex items-center justify-between gap-3">
        <h2 class="text-lg font-semibold tracking-tight">{{ selectedRow.name }} 提供商设置</h2>
        <span class="flex shrink-0 gap-1.5">
          <em
            v-if="selected && !selected.editable"
            class="rounded-full bg-[var(--ec-canvas-inset)] px-2 py-0.5 text-xs font-medium text-[var(--ec-fg-muted)] not-italic"
            >平台</em
          >
          <em
            class="rounded-full px-2 py-0.5 text-xs font-medium not-italic"
            :class="
              selectedEnabled
                ? 'bg-[color-mix(in_srgb,var(--ec-success)_16%,transparent)] text-[var(--ec-success)]'
                : 'bg-[var(--ec-canvas-inset)] text-[var(--ec-fg-muted)]'
            "
            >{{ selectedEnabled ? '已启用' : '已停用' }}</em
          >
        </span>
      </header>

      <p v-if="err" class="mt-3 text-sm text-[var(--ec-danger)]">{{ err }}</p>
      <p v-else-if="fieldError" class="mt-3 text-sm text-[var(--ec-danger)]">{{ fieldError }}</p>
      <p v-else-if="saving" class="mt-3 text-xs text-[var(--ec-fg-muted)]">保存中…</p>
      <p v-else-if="saved" class="mt-3 text-xs text-[var(--ec-fg-muted)]">已保存</p>
      <p v-else-if="loading" class="mt-3 text-xs text-[var(--ec-fg-muted)]">正在加载供应商…</p>

      <label class="mt-4 grid gap-1.5">
        <span class="text-[13px] text-[var(--ec-fg-muted)]">API Key</span>
        <UiField
          v-model="form.apiKey"
          type="password"
          autocomplete="new-password"
          placeholder="输入 API Key"
          :disabled="!formEditable"
          @change="persist"
        />
      </label>

      <label class="mt-4 grid gap-1.5">
        <span class="text-[13px] text-[var(--ec-fg-muted)]">API Base URL</span>
        <UiField
          v-model="form.baseUrl"
          type="text"
          inputmode="url"
          autocomplete="off"
          placeholder="https://api.example.com/v1"
          :disabled="!formEditable"
          :invalid="urlInvalid"
          @change="persist"
        />
      </label>

      <fieldset class="mt-4 grid gap-1.5 border-0 p-0" :disabled="!formEditable">
        <legend class="text-[13px] text-[var(--ec-fg-muted)]">API 格式</legend>
        <div class="mt-1 flex flex-wrap gap-x-5 gap-y-2">
          <label class="inline-flex items-center gap-1.5 text-sm">
            <input
              type="radio"
              name="provider-protocol"
              class="h-3.5 w-3.5 accent-[var(--ec-accent)]"
              value="anthropic"
              :checked="form.protocol === 'anthropic'"
              @change="onProtocol('anthropic')"
            />
            Anthropic 兼容
          </label>
          <label class="inline-flex items-center gap-1.5 text-sm">
            <input
              type="radio"
              name="provider-protocol"
              class="h-3.5 w-3.5 accent-[var(--ec-accent)]"
              value="openai"
              :checked="form.protocol === 'openai'"
              @change="onProtocol('openai')"
            />
            OpenAI 兼容
          </label>
        </div>
        <p class="text-xs text-[var(--text-dim)]">请选择 API 协议兼容格式：Anthropic 兼容或 OpenAI 兼容</p>
      </fieldset>

      <div class="mt-3.5 flex min-w-0 items-center gap-3">
        <button
          type="button"
          class="inline-flex shrink-0 items-center gap-1.5 rounded-[10px] border border-[var(--border-strong)] bg-[var(--ec-canvas)] px-3.5 py-2 text-[13px] disabled:opacity-60"
          :disabled="currentTest?.testing || (!!selected && !selected.editable)"
          @click="testProvider"
        >
          <svg viewBox="0 0 24 24" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" aria-hidden="true">
            <path d="M4.6 9.6a10 10 0 0 1 14.8 0" />
            <path d="M7.4 12.6a6 6 0 0 1 9.2 0" />
            <path d="M10 15.5a2.8 2.8 0 0 1 4 0" />
            <circle cx="12" cy="18.5" r="1.15" fill="currentColor" stroke="none" />
          </svg>
          {{ currentTest?.testing ? '测试中…' : '测试连接' }}
        </button>
        <p
          v-if="currentTest && !currentTest.testing && currentTest.message"
          class="min-w-0 flex-1 truncate text-[13px]"
          :class="currentTest.ok ? 'text-[var(--ec-success)]' : 'text-[var(--ec-danger)]'"
          :title="currentTest.message"
        >
          {{ currentTest.ok ? '连接成功' : currentTest.message }}
        </p>
      </div>

      <div class="mt-6 mb-2.5 flex items-center justify-between gap-3">
        <h3 class="text-sm font-semibold">可用模型列表</h3>
        <button
          v-if="formEditable"
          type="button"
          class="inline-flex items-center gap-1 px-0.5 text-[13px] text-[var(--ec-accent)]"
          @click="openModelDialog()"
        >
          <svg viewBox="0 0 24 24" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true">
            <path d="M12 5v14M5 12h14" />
          </svg>
          添加模型
        </button>
      </div>

      <ul class="space-y-2">
        <li v-for="item in selectedModels" :key="item.id">
          <div
            class="group flex items-center gap-2.5 rounded-xl border border-[var(--ec-border)] bg-[var(--ec-canvas)] px-3 py-2.5"
            :class="item.editable ? 'cursor-pointer hover:border-[var(--border-strong)]' : ''"
            @click="item.editable && openModelDialog(item)"
          >
            <i class="h-2 w-2 shrink-0 rounded-full bg-[var(--ec-success)]" />
            <span class="min-w-0 flex-1 truncate text-sm">{{ modelTitle(item.name) }}</span>
            <button
              v-if="item.editable"
              type="button"
              class="shrink-0 rounded-md px-1 text-xs text-[var(--text-dim)] opacity-0 group-hover:opacity-100 focus-visible:opacity-100 hover:text-[var(--ec-danger)]"
              @click.stop="removeModel(item)"
            >
              删除
            </button>
            <code
              v-if="modelCode(item.name)"
              class="max-w-[46%] truncate rounded-lg bg-[var(--ec-canvas-inset)] px-2 py-0.5 font-mono text-xs text-[var(--ec-fg-muted)]"
              >{{ modelCode(item.name) }}</code
            >
          </div>
        </li>
        <li
          v-if="!selectedModels.length"
          class="rounded-xl border border-dashed border-[var(--ec-border)] px-3 py-2.5 text-center text-[13px] text-[var(--text-dim)]"
        >
          {{ selected ? '还没有模型' : '开启供应商后可以添加模型' }}
        </li>
      </ul>

      <div v-if="selected?.editable && !selectedRow.preset" class="mt-4 flex gap-4">
        <button type="button" class="px-0 text-xs text-[var(--text-dim)] hover:text-[var(--ec-fg)]" @click="openCustom(selected)">
          编辑名称与网站
        </button>
        <button type="button" class="px-0 text-xs text-[var(--text-dim)] hover:text-[var(--ec-danger)]" @click="removeProvider(selected)">
          删除供应商
        </button>
      </div>
    </section>

    <section v-else class="min-w-0 flex-1 px-8 py-4 text-sm text-[var(--text-dim)]">
      {{ loading ? '正在加载供应商…' : '请选择左侧的供应商' }}
    </section>

    <UiDialog
      :show="customOpen"
      :title="customEditing ? '编辑供应商' : '自定义供应商'"
      width="480px"
      :mask-closable="!customSaving"
      :close-on-esc="!customSaving"
      @update:show="onCustomShow"
    >
      <form id="custom-provider-form" class="grid gap-3" @submit.prevent="saveCustom">
        <p v-if="customErr" class="text-sm text-[var(--ec-danger)]">{{ customErr }}</p>
        <label class="grid gap-1.5">
          <span class="text-[13px] text-[var(--ec-fg-muted)]">供应商名称</span>
          <UiField v-model="customForm.name" autocomplete="off" placeholder="例如 内部网关" />
        </label>
        <label class="grid gap-1.5">
          <span class="text-[13px] text-[var(--ec-fg-muted)]">API Key</span>
          <UiField v-model="customForm.apiKey" type="password" autocomplete="new-password" placeholder="输入 API Key" />
        </label>
        <label class="grid gap-1.5">
          <span class="text-[13px] text-[var(--ec-fg-muted)]">API Base URL</span>
          <UiField
            v-model="customForm.baseUrl"
            inputmode="url"
            autocomplete="off"
            placeholder="https://api.example.com/v1"
          />
        </label>
        <fieldset class="grid gap-1.5 border-0 p-0">
          <legend class="text-[13px] text-[var(--ec-fg-muted)]">API 格式</legend>
          <div class="flex flex-wrap gap-x-5 gap-y-2">
            <label class="inline-flex items-center gap-1.5 text-sm">
              <input v-model="customForm.protocol" type="radio" class="h-3.5 w-3.5 accent-[var(--ec-accent)]" value="anthropic" />
              Anthropic 兼容
            </label>
            <label class="inline-flex items-center gap-1.5 text-sm">
              <input v-model="customForm.protocol" type="radio" class="h-3.5 w-3.5 accent-[var(--ec-accent)]" value="openai" />
              OpenAI 兼容
            </label>
          </div>
        </fieldset>
        <label class="grid gap-1.5">
          <span class="text-[13px] text-[var(--ec-fg-muted)]">官方网站</span>
          <UiField v-model="customForm.website" inputmode="url" autocomplete="off" placeholder="https://...（可选）" />
        </label>
      </form>
      <template #footer>
        <UiButton :disabled="customSaving" @click="closeCustom">取消</UiButton>
        <UiButton
          type="primary"
          attr-type="submit"
          form="custom-provider-form"
          :loading="customSaving"
          :disabled="customSaving"
        >
          {{ customSaving ? '保存中…' : '保存' }}
        </UiButton>
      </template>
    </UiDialog>

    <ModelConfigDialog
      :show="modelOpen"
      :provider-id="modelProviderId"
      :provider-name="modelProviderName"
      :model="modelTarget"
      @update:show="modelOpen = $event"
      @saved="load"
    />
  </div>
</template>
