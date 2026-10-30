<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { onClickOutside } from '@vueuse/core'
import UiButton from '@/components/ui/UiButton.vue'
import UiDialog from '@/components/ui/UiDialog.vue'
import UiField from '@/components/ui/UiField.vue'
import {
  createLlmModel,
  listInputCapabilities,
  listRemoteModels,
  updateLlmModel,
  type LlmModel,
  type ModelPayload
} from '@/api/llm'
import { nonNegative, requiredText } from '@/lib/formCheck'

const props = defineProps<{
  show: boolean
  providerId: number
  providerName: string
  model?: LlmModel | null
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  saved: []
}>()

const reasoningEfforts = ['low', 'medium', 'high']
const capLabel: Record<string, string> = {
  text: '文本',
  image: '图片',
  pdf: 'PDF'
}

const editing = computed(() => !!props.model?.id)
const title = computed(() => (editing.value ? '编辑模型' : '添加模型'))
const form = reactive({
  name: '',
  attachment: false,
  reasoning: false,
  reasoningOptions: [''] as string[],
  temperature: false,
  inputCapabilities: [] as string[],
  inputContextLimit: '200000',
  outputContextLimit: '0',
  inputPrice: '0',
  outputPrice: '0',
  cachePrice: '0'
})
const showAdvanced = ref(false)
const capabilities = ref<string[]>(['text', 'image', 'pdf'])
const remoteModels = ref<string[]>([])
const modelsLoading = ref(false)
const suggestOpen = ref(false)
const suggestRef = ref<HTMLElement | null>(null)
const err = ref('')
const nameError = ref('')
const saving = ref(false)
let loadSeq = 0

const suggestions = computed(() => {
  const query = form.name.trim().toLowerCase()
  const list = remoteModels.value
  const matched = query ? list.filter((item) => item.toLowerCase().includes(query)) : list
  return matched.slice(0, 12)
})

onClickOutside(suggestRef, () => {
  suggestOpen.value = false
})

watch(
  () => props.show,
  (open) => {
    if (open) void load()
  }
)

watch(
  () => form.reasoning,
  (on) => {
    if (!on) return
    const options = form.reasoningOptions.map((item) => item.trim()).filter(Boolean)
    if (!options.length) form.reasoningOptions = [...reasoningEfforts]
  }
)

function resetForm(): void {
  const row = props.model
  form.name = row?.name || ''
  form.attachment = !!row?.attachment
  form.reasoning = !!row?.reasoning
  form.reasoningOptions = row?.reasoningOptions?.length ? [...row.reasoningOptions] : ['']
  form.temperature = !!row?.temperature
  form.inputCapabilities = [...(row?.inputCapabilities || [])]
  form.inputContextLimit = String(row?.inputContextLimit ?? 200000)
  form.outputContextLimit = String(row?.outputContextLimit ?? 0)
  form.inputPrice = String(row?.inputPrice ?? 0)
  form.outputPrice = String(row?.outputPrice ?? 0)
  form.cachePrice = String(row?.cachePrice ?? 0)
  nameError.value = ''
  err.value = ''
  showAdvanced.value = false
  suggestOpen.value = false
  if (row) expandAdvancedIfCustomized()
}

function expandAdvancedIfCustomized(): void {
  showAdvanced.value =
    form.attachment ||
    form.reasoning ||
    form.temperature ||
    form.inputContextLimit !== '200000' ||
    Number(form.outputContextLimit) > 0 ||
    Number(form.inputPrice) > 0 ||
    Number(form.outputPrice) > 0 ||
    Number(form.cachePrice) > 0
}

async function load(): Promise<void> {
  resetForm()
  try {
    capabilities.value = await listInputCapabilities()
  } catch {
    capabilities.value = ['text', 'image', 'pdf']
  }
  await loadRemote(props.providerId)
}

async function loadRemote(providerId: number): Promise<void> {
  const seq = ++loadSeq
  if (!providerId) {
    remoteModels.value = []
    return
  }
  modelsLoading.value = true
  try {
    const list = await listRemoteModels(providerId)
    if (seq !== loadSeq) return
    remoteModels.value = list
  } catch {
    if (seq !== loadSeq) return
    remoteModels.value = []
  } finally {
    if (seq === loadSeq) modelsLoading.value = false
  }
}

function close(): void {
  if (saving.value) return
  emit('update:show', false)
}

function onShow(value: boolean): void {
  if (!value) close()
  else emit('update:show', true)
}

function pickName(name: string): void {
  form.name = name
  nameError.value = ''
  suggestOpen.value = false
}

function setCap(opt: string, checked: boolean): void {
  const index = form.inputCapabilities.indexOf(opt)
  if (checked && index < 0) form.inputCapabilities.push(opt)
  else if (!checked && index >= 0) form.inputCapabilities.splice(index, 1)
}

function addReasoningOption(): void {
  form.reasoningOptions.push('')
}

function removeReasoningOption(index: number): void {
  form.reasoningOptions.splice(index, 1)
}

function capText(opt: string): string {
  return capLabel[opt] || opt
}

async function submit(): Promise<void> {
  nameError.value = requiredText(form.name, '请选择或输入模型名称')
  if (!props.providerId) {
    err.value = '请先开启并保存供应商'
    return
  }
  const limitError =
    nonNegative(form.inputContextLimit, '输入上下文') ||
    nonNegative(form.outputContextLimit, '输出上下文') ||
    nonNegative(form.inputPrice, '输入价格') ||
    nonNegative(form.outputPrice, '输出价格') ||
    nonNegative(form.cachePrice, '缓存价格')
  if (limitError) {
    err.value = limitError
    showAdvanced.value = true
    return
  }
  if (nameError.value) return
  saving.value = true
  err.value = ''
  const payload: ModelPayload = {
    providerId: props.providerId,
    name: form.name.trim(),
    attachment: form.attachment,
    reasoning: form.reasoning,
    reasoningOptions: form.reasoning
      ? form.reasoningOptions.map((item) => item.trim()).filter(Boolean)
      : [],
    temperature: form.temperature,
    inputCapabilities: form.attachment ? form.inputCapabilities : [],
    inputContextLimit: Number(form.inputContextLimit) || 0,
    outputContextLimit: Number(form.outputContextLimit) || 0,
    inputPrice: Number(form.inputPrice) || 0,
    outputPrice: Number(form.outputPrice) || 0,
    cachePrice: Number(form.cachePrice) || 0
  }
  try {
    if (editing.value && props.model?.id) await updateLlmModel(props.model.id, payload)
    else await createLlmModel(payload)
    emit('saved')
    emit('update:show', false)
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UiDialog
    :show="show"
    :title="title"
    width="560px"
    :mask-closable="!saving"
    :close-on-esc="!saving"
    @update:show="onShow"
  >
    <form id="model-config-form" class="grid gap-3.5" @submit.prevent="submit">
      <p v-if="err" class="text-sm text-[var(--ec-danger)]">{{ err }}</p>
      <p class="text-[13px] text-[var(--ec-fg-muted)]">供应商：{{ providerName }}</p>
      <label class="grid gap-1.5">
        <span class="text-[13px] text-[var(--ec-fg-muted)]">模型名称</span>
        <span ref="suggestRef" class="relative block">
          <UiField
            v-model="form.name"
            :invalid="!!nameError"
            :placeholder="modelsLoading ? '正在拉取远程模型…' : '选择或直接输入'"
            autocomplete="off"
            @focus="suggestOpen = true"
            @update:model-value="nameError = ''"
          />
          <ul
            v-if="suggestOpen && suggestions.length"
            class="absolute z-10 mt-1 max-h-52 w-full overflow-auto rounded-[10px] border border-[var(--ec-border)] bg-[var(--ec-canvas-subtle)] py-1 shadow-[var(--shadow)]"
          >
            <li v-for="item in suggestions" :key="item">
              <button
                type="button"
                class="block w-full truncate px-3 py-1.5 text-left text-sm hover:bg-[var(--surface-hover)]"
                @click="pickName(item)"
              >
                {{ item }}
              </button>
            </li>
          </ul>
        </span>
        <span v-if="nameError" class="text-xs text-[var(--ec-danger)]">{{ nameError }}</span>
      </label>
      <button
        type="button"
        class="justify-self-start px-0 text-[13px] text-[var(--ec-fg-muted)] hover:text-[var(--ec-fg)]"
        :aria-expanded="showAdvanced"
        @click="showAdvanced = !showAdvanced"
      >
        <span class="mr-1 inline-block transition-transform" :class="showAdvanced ? 'rotate-90' : ''">›</span>
        高级设置（能力、上下文、价格）
      </button>
      <div v-show="showAdvanced" class="grid gap-3.5 rounded-[12px] border border-[var(--ec-border)] p-3.5">
        <label class="flex items-center gap-2 text-sm">
          <input v-model="form.attachment" type="checkbox" class="h-4 w-4 accent-[var(--ec-accent)]" />
          支持附件
        </label>
        <div v-if="form.attachment" class="grid gap-1.5">
          <span class="text-[13px] text-[var(--ec-fg-muted)]">输入能力</span>
          <div class="flex flex-wrap gap-x-4 gap-y-2">
            <label v-for="opt in capabilities" :key="opt" class="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                class="h-4 w-4 accent-[var(--ec-accent)]"
                :checked="form.inputCapabilities.includes(opt)"
                @change="setCap(opt, ($event.target as HTMLInputElement).checked)"
              />
              {{ capText(opt) }}
            </label>
          </div>
        </div>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="form.reasoning" type="checkbox" class="h-4 w-4 accent-[var(--ec-accent)]" />
          支持推理
        </label>
        <div v-if="form.reasoning" class="grid gap-2">
          <span class="text-[13px] text-[var(--ec-fg-muted)]">推理选项</span>
          <div v-for="(_, index) in form.reasoningOptions" :key="index" class="flex items-center gap-2">
            <UiField v-model="form.reasoningOptions[index]" placeholder="如 low / medium / high" />
            <button
              type="button"
              class="shrink-0 rounded-md px-2 py-1 text-xs text-[var(--ec-fg-muted)] hover:text-[var(--ec-danger)]"
              @click="removeReasoningOption(index)"
            >
              删除
            </button>
          </div>
          <button
            type="button"
            class="justify-self-start rounded-md border border-[var(--ec-border)] px-2 py-1 text-xs text-[var(--ec-fg-muted)]"
            @click="addReasoningOption"
          >
            追加选项
          </button>
        </div>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="form.temperature" type="checkbox" class="h-4 w-4 accent-[var(--ec-accent)]" />
          温度调节
        </label>
        <div class="grid grid-cols-2 gap-3">
          <label class="grid gap-1.5">
            <span class="text-[13px] text-[var(--ec-fg-muted)]">输入上下文</span>
            <UiField v-model="form.inputContextLimit" type="number" :min="0" :step="1000" />
          </label>
          <label class="grid gap-1.5">
            <span class="text-[13px] text-[var(--ec-fg-muted)]">输出上下文（0 不限制）</span>
            <UiField v-model="form.outputContextLimit" type="number" :min="0" :step="1000" />
          </label>
          <label class="grid gap-1.5">
            <span class="text-[13px] text-[var(--ec-fg-muted)]">输入价格</span>
            <UiField v-model="form.inputPrice" type="number" :min="0" step="0.000001" />
          </label>
          <label class="grid gap-1.5">
            <span class="text-[13px] text-[var(--ec-fg-muted)]">输出价格</span>
            <UiField v-model="form.outputPrice" type="number" :min="0" step="0.000001" />
          </label>
          <label class="grid gap-1.5">
            <span class="text-[13px] text-[var(--ec-fg-muted)]">缓存价格</span>
            <UiField v-model="form.cachePrice" type="number" :min="0" step="0.000001" />
          </label>
        </div>
      </div>
    </form>
    <template #footer>
      <UiButton :disabled="saving" @click="close">取消</UiButton>
      <UiButton
        type="primary"
        attr-type="submit"
        form="model-config-form"
        :loading="saving"
        :disabled="saving"
      >
        {{ saving ? '保存中…' : '保存' }}
      </UiButton>
    </template>
  </UiDialog>
</template>
