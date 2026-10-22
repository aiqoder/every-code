<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useClientPort } from '../../../ports/clientPort'
import FormField from '../../../components/FormField.vue'
import Checkbox from '../../../components/Checkbox.vue'
import NumberField from '../../../components/NumberField.vue'
import SelectDropdown from '../../../components/SelectDropdown.vue'
import SuggestField from '../../../components/SuggestField.vue'
import {
  createFieldErrors,
  hasFieldErrors,
  required,
} from '../../../lib/formValidation'
import type { LlmProvider, Model, ModelPayload } from '../../../types'
import EditorsBreadcrumb from '../components/EditorsBreadcrumb.vue'

type ModelField = 'providerId' | 'name'

const route = useRoute()
const router = useRouter()
const port = useClientPort()

const modelId = Number(route.params.id) || 0
const editing = modelId > 0

const form = reactive({
  providerId: 0,
  name: '',
  attachment: false,
  reasoning: false,
  reasoningOptions: [''] as string[],
  temperature: false,
  inputCapabilities: [] as string[],
  inputContextLimit: 200000,
  outputContextLimit: 0,
  inputPrice: 0,
  outputPrice: 0,
  cachePrice: 0,
})
const showAdvanced = ref(false)

// 编辑已有模型时，若高级项非默认值则自动展开，避免用户以为配置丢失。
function expandAdvancedIfCustomized() {
  showAdvanced.value =
    form.attachment ||
    form.reasoning ||
    form.temperature ||
    form.inputContextLimit !== 200000 ||
    form.outputContextLimit > 0 ||
    form.inputPrice > 0 ||
    form.outputPrice > 0 ||
    form.cachePrice > 0
}
const providers = ref<LlmProvider[]>([])
const capabilities = ref<string[]>([])
const remoteModels = ref<string[]>([])
const modelsLoading = ref(false)
const loading = ref(true)
const err = ref('')
const saving = ref(false)
const fieldErrors = reactive(createFieldErrors<ModelField>(['providerId', 'name']))
const commonReasoningOptions = ['low', 'medium', 'high']
let loadSeq = 0

const providerOptions = computed(() =>
  providers.value.map((p) => ({ value: p.id, label: p.name })),
)

const modelOptions = computed(() => {
  const set = new Set(remoteModels.value)
  const name = form.name.trim()
  if (name && !set.has(name)) return [name, ...remoteModels.value]
  return remoteModels.value
})

async function loadRemote(providerId: number) {
  const seq = ++loadSeq
  if (!providerId) {
    remoteModels.value = []
    return
  }
  modelsLoading.value = true
  try {
    const list = await port.listRemoteModels(providerId)
    if (seq !== loadSeq) return
    remoteModels.value = list
  } catch {
    if (seq !== loadSeq) return
    remoteModels.value = []
  } finally {
    if (seq === loadSeq) modelsLoading.value = false
  }
}

async function load() {
  loading.value = true
  err.value = ''
  try {
    const [p, caps] = await Promise.all([
      port.listProviders(),
      port.listInputCapabilities(),
    ])
    providers.value = (p ?? []).filter((item) => item.editable)
    capabilities.value = caps ?? ['text', 'image', 'pdf']

    if (editing) {
      const models = await port.listModels()
      const row = (models ?? []).find((m) => m.id === modelId)
      if (row) {
        form.providerId = row.providerId || 0
        form.name = row.name
        form.attachment = !!row.attachment
        form.reasoning = !!row.reasoning
        form.reasoningOptions = row.reasoningOptions?.length ? [...row.reasoningOptions] : ['']
        form.temperature = !!row.temperature
        form.inputCapabilities = [...(row.inputCapabilities || [])]
        form.inputContextLimit = row.inputContextLimit ?? 200000
        form.outputContextLimit = row.outputContextLimit ?? 0
        form.inputPrice = row.inputPrice ?? 0
        form.outputPrice = row.outputPrice ?? 0
        form.cachePrice = row.cachePrice ?? 0
        expandAdvancedIfCustomized()
        await loadRemote(form.providerId)
      }
    } else {
      const presetProviderId = Number(route.query.providerId) || 0
      form.providerId =
        providers.value.find((item) => item.id === presetProviderId)?.id ??
        providers.value[0]?.id ??
        0
      if (form.providerId) await loadRemote(form.providerId)
    }
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

function clearFieldError(field: ModelField) {
  fieldErrors[field] = ''
}

function validateForm(): boolean {
  fieldErrors.providerId = form.providerId ? '' : '请选择供应商'
  fieldErrors.name = required(form.name, '请选择或输入模型名称')
  return !hasFieldErrors(fieldErrors)
}

async function onProviderSelect(id: number | null) {
  form.providerId = id ?? 0
  clearFieldError('providerId')
  form.name = ''
  clearFieldError('name')
  await loadRemote(form.providerId)
}

function setCap(opt: string, checked: boolean) {
  const i = form.inputCapabilities.indexOf(opt)
  if (checked && i < 0) form.inputCapabilities.push(opt)
  else if (!checked && i >= 0) form.inputCapabilities.splice(i, 1)
}

function addReasoningOption() {
  form.reasoningOptions.push('')
}

function removeReasoningOption(index: number) {
  form.reasoningOptions.splice(index, 1)
}

function goBack() {
  router.push({ name: 'settings-models', query: route.query })
}

async function submit() {
  if (!validateForm()) return
  saving.value = true
  err.value = ''
  const payload: ModelPayload = {
    providerId: form.providerId,
    name: form.name.trim(),
    attachment: form.attachment,
    reasoning: form.reasoning,
    reasoningOptions: form.reasoning
      ? form.reasoningOptions.map((s) => s.trim()).filter(Boolean)
      : [],
    temperature: form.temperature,
    inputCapabilities: form.attachment ? form.inputCapabilities : [],
    inputContextLimit: Number(form.inputContextLimit) || 0,
    outputContextLimit: Number(form.outputContextLimit) || 0,
    inputPrice: Number(form.inputPrice) || 0,
    outputPrice: Number(form.outputPrice) || 0,
    cachePrice: Number(form.cachePrice) || 0,
  }
  try {
    if (editing) {
      await port.updateModel(modelId, payload)
    } else {
      await port.createModel(payload)
    }
    goBack()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="editor-page">
    <EditorsBreadcrumb
      :items="[
        { label: '设置', to: { name: 'settings' } },
        { label: '模型配置', to: { name: 'settings-models' } },
        { label: editing ? '编辑大模型' : '新增大模型' },
      ]"
    />

    <form class="dlg" @submit.prevent="submit">
      <p v-if="err" class="err">{{ err }}</p>
      <p v-if="!loading && !providers.length" class="hint">
        请先创建自己的供应商，再维护模型。
      </p>
      <FormField label="供应商" required :error="fieldErrors.providerId">
        <SelectDropdown
          :model-value="form.providerId || null"
          variant="field"
          placement="bottom"
          placeholder="选择供应商"
          aria-label="选择供应商"
          :options="providerOptions"
          :disabled="!providers.length"
          :invalid="!!fieldErrors.providerId"
          @update:model-value="onProviderSelect"
        />
      </FormField>
      <FormField label="模型名称" required :error="fieldErrors.name">
        <SuggestField
          v-model="form.name"
          :options="modelOptions"
          :disabled="!form.providerId"
          :placeholder="modelsLoading ? '正在拉取远程模型…' : '选择或直接输入'"
          :invalid="!!fieldErrors.name"
          aria-label="模型名称"
          @update:model-value="clearFieldError('name')"
        />
      </FormField>
      <button
        type="button"
        class="advanced-toggle"
        :aria-expanded="showAdvanced"
        @click="showAdvanced = !showAdvanced"
      >
        <span class="chevron" :class="{ open: showAdvanced }">›</span>
        高级设置（能力、上下文、价格）
      </button>
      <div v-show="showAdvanced" class="advanced">
        <Checkbox v-model="form.attachment">支持附件</Checkbox>
        <FormField v-if="form.attachment" label="输入能力">
          <div class="caps">
            <Checkbox
              v-for="opt in capabilities"
              :key="opt"
              inline
              :model-value="form.inputCapabilities.includes(opt)"
              @update:model-value="setCap(opt, $event)"
            >
              {{ opt }}
            </Checkbox>
          </div>
        </FormField>
        <Checkbox v-model="form.reasoning">支持推理</Checkbox>
        <FormField v-if="form.reasoning" label="推理选项">
          <div class="reason">
            <div v-for="(_, index) in form.reasoningOptions" :key="index" class="reason-row">
              <SuggestField
                v-model="form.reasoningOptions[index]"
                :options="commonReasoningOptions"
                placeholder="如 low / medium / high"
                aria-label="推理选项"
              />
              <button type="button" @click="removeReasoningOption(index)">删除</button>
            </div>
            <button type="button" @click="addReasoningOption">追加选项</button>
          </div>
        </FormField>
        <Checkbox v-model="form.temperature">温度调节</Checkbox>
        <div class="nums">
          <FormField label="输入上下文">
            <NumberField v-model="form.inputContextLimit" :min="0" :step="1000" />
          </FormField>
          <FormField label="输出上下文（0 不限制）">
            <NumberField v-model="form.outputContextLimit" :min="0" :step="1000" />
          </FormField>
          <FormField label="输入价格">
            <NumberField v-model="form.inputPrice" :min="0" :step="0.000001" />
          </FormField>
          <FormField label="输出价格">
            <NumberField v-model="form.outputPrice" :min="0" :step="0.000001" />
          </FormField>
          <FormField label="缓存价格">
            <NumberField v-model="form.cachePrice" :min="0" :step="0.000001" />
          </FormField>
        </div>
      </div>
      <div class="actions">
        <button type="button" @click="goBack">取消</button>
        <button type="submit" class="go" :disabled="saving || loading || !providers.length">
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.editor-page {
  width: 100%;
  max-width: 640px;
  margin: 0 auto;
  align-self: center;
}

.dlg {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 24px;
  display: grid;
  gap: 14px;
}

.advanced-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  font: inherit;
  font-size: 13px;
  padding: 0;
  background: transparent;
  border: 0;
  color: var(--text-muted);
  justify-self: start;
}

.advanced-toggle:hover {
  color: var(--text);
  background: transparent;
}

.chevron {
  display: inline-block;
  transition: transform 0.15s ease;
}

.chevron.open {
  transform: rotate(90deg);
}

.advanced {
  display: grid;
  gap: 14px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.caps,
.reason {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.reason-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.reason-row :deep(.suggest-wrap) {
  flex: 1;
}

.nums {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.hint {
  margin: 0;
  color: var(--text-dim);
  font-size: 12px;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 4px;
}

button {
  font: inherit;
  font-size: 13px;
  padding: 8px 16px;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
  border-radius: var(--radius-sm);
}

button:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.go {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}

.go:hover {
  background: var(--accent-hover);
}

.go:disabled {
  opacity: 0.6;
}

.err {
  margin: 0;
  color: var(--error);
  font-size: 13px;
}
</style>
