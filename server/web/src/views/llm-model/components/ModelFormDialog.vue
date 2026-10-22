<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { Plus, Delete } from '@element-plus/icons-vue'

import { fetchInputCapabilities } from '@/api/llm-model'
import type { LlmModel, LlmModelPayload, LlmProvider } from '@/types/api'

const visible = defineModel<boolean>({ default: false })

const props = defineProps<{
  editing?: LlmModel | null
  providers: LlmProvider[]
  submit: (payload: LlmModelPayload) => Promise<void>
  fetchRemoteModels: (providerId: number) => Promise<{ list: string[] }>
}>()

const emit = defineEmits<{
  success: []
}>()

const formRef = ref<FormInstance>()
const submitting = ref(false)
const capabilityOptions = ref<string[]>([])
const remoteModels = ref<string[]>([])
const modelsLoading = ref(false)
const commonReasoningOptions = ['low', 'medium', 'high']
let modelsLoadSeq = 0

const form = reactive({
  providerId: undefined as number | undefined,
  name: '',
  attachment: false,
  reasoning: false,
  reasoningOptions: [] as string[],
  temperature: false,
  inputCapabilities: [] as string[],
  inputContextLimit: 200000,
  outputContextLimit: 0,
  inputPrice: 0,
  outputPrice: 0,
  cachePrice: 0,
})

const rules: FormRules = {
  providerId: [{ required: true, message: '请选择供应商', trigger: 'change' }],
  name: [{ required: true, message: '请选择或输入模型名称', trigger: 'change' }],
}

const dialogTitle = computed(() => (props.editing ? '编辑大模型' : '新增大模型'))

const modelOptions = computed(() => {
  const set = new Set(remoteModels.value)
  const name = form.name.trim()
  if (name && !set.has(name)) {
    return [name, ...remoteModels.value]
  }
  return remoteModels.value
})

function resetForm() {
  form.providerId = undefined
  form.name = ''
  form.attachment = false
  form.reasoning = false
  form.reasoningOptions = []
  form.temperature = false
  form.inputCapabilities = []
  form.inputContextLimit = 200000
  form.outputContextLimit = 0
  form.inputPrice = 0
  form.outputPrice = 0
  form.cachePrice = 0
  remoteModels.value = []
}

function fillForm(row: LlmModel) {
  form.providerId = row.providerId
  form.name = row.name
  form.attachment = row.attachment
  form.reasoning = row.reasoning
  form.reasoningOptions = [...(row.reasoningOptions || [])]
  form.temperature = row.temperature
  form.inputCapabilities = [...(row.inputCapabilities || [])]
  form.inputContextLimit = row.inputContextLimit
  form.outputContextLimit = row.outputContextLimit
  form.inputPrice = row.inputPrice
  form.outputPrice = row.outputPrice
  form.cachePrice = row.cachePrice
}

async function loadRemoteModels(providerId?: number) {
  const seq = ++modelsLoadSeq
  if (!providerId) {
    remoteModels.value = []
    modelsLoading.value = false
    return
  }
  modelsLoading.value = true
  try {
    const data = await props.fetchRemoteModels(providerId)
    if (seq !== modelsLoadSeq) return
    remoteModels.value = data.list ?? []
  } catch {
    if (seq !== modelsLoadSeq) return
    remoteModels.value = []
  } finally {
    if (seq === modelsLoadSeq) {
      modelsLoading.value = false
    }
  }
}

watch(visible, async (v) => {
  if (!v) {
    formRef.value?.resetFields()
    remoteModels.value = []
    return
  }
  if (props.editing) {
    fillForm(props.editing)
    await loadRemoteModels(props.editing.providerId)
  } else {
    resetForm()
  }
})

async function onProviderChange(providerId: number | undefined) {
  form.name = ''
  await loadRemoteModels(providerId)
}

watch(
  () => form.attachment,
  (enabled) => {
    if (!enabled) {
      form.inputCapabilities = []
    }
  },
)

watch(
  () => form.reasoning,
  (enabled) => {
    if (!enabled) {
      form.reasoningOptions = []
    } else if (form.reasoningOptions.length === 0) {
      form.reasoningOptions = ['']
    }
  },
)

function addReasoningOption() {
  form.reasoningOptions.push('')
}

function removeReasoningOption(index: number) {
  form.reasoningOptions.splice(index, 1)
}

async function loadCapabilities() {
  const data = await fetchInputCapabilities()
  capabilityOptions.value = data.list ?? []
}

async function onConfirm() {
  if (!formRef.value) return
  await formRef.value.validate()
  if (!form.providerId) return

  const name = form.name.trim()
  if (!name) {
    ElMessage.warning('请选择或输入模型名称')
    return
  }

  const reasoningOptions = form.reasoning
    ? form.reasoningOptions.map((s) => s.trim()).filter(Boolean)
    : []

  submitting.value = true
  try {
    const payload = {
      providerId: form.providerId,
      name,
      attachment: form.attachment,
      reasoning: form.reasoning,
      reasoningOptions,
      temperature: form.temperature,
      inputCapabilities: form.attachment ? form.inputCapabilities : [],
      inputContextLimit: form.inputContextLimit,
      outputContextLimit: form.outputContextLimit,
      inputPrice: form.inputPrice,
      outputPrice: form.outputPrice,
      cachePrice: form.cachePrice,
    }
    await props.submit(payload)
    ElMessage.success(props.editing ? '更新成功' : '创建成功')
    visible.value = false
    emit('success')
  } finally {
    submitting.value = false
  }
}

onMounted(loadCapabilities)
</script>

<template>
  <el-dialog v-model="visible" :title="dialogTitle" width="680px" destroy-on-close>
    <el-form ref="formRef" :model="form" :rules="rules" label-width="140px">
      <el-form-item label="供应商" prop="providerId">
        <el-select
          v-model="form.providerId"
          placeholder="请选择供应商"
          filterable
          style="width: 100%"
          @change="onProviderChange"
        >
          <el-option
            v-for="p in providers"
            :key="p.id"
            :label="p.name"
            :value="p.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="模型名称" prop="name">
        <el-select
          v-model="form.name"
          filterable
          allow-create
          default-first-option
          clearable
          :disabled="!form.providerId"
          :loading="modelsLoading"
          placeholder="选择供应商后可下拉选择，也可直接输入"
          style="width: 100%"
        >
          <el-option
            v-for="m in modelOptions"
            :key="m"
            :label="m"
            :value="m"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="支持附件">
        <el-switch v-model="form.attachment" />
      </el-form-item>
      <el-form-item v-if="form.attachment" label="输入能力">
        <el-checkbox-group v-model="form.inputCapabilities">
          <el-checkbox
            v-for="opt in capabilityOptions"
            :key="opt"
            :label="opt"
            :value="opt"
          >
            {{ opt }}
          </el-checkbox>
        </el-checkbox-group>
      </el-form-item>
      <el-form-item label="支持推理">
        <el-switch v-model="form.reasoning" />
      </el-form-item>
      <el-form-item v-if="form.reasoning" label="推理选项">
        <div class="reasoning-options">
          <div
            v-for="(_, index) in form.reasoningOptions"
            :key="index"
            class="reasoning-row"
          >
            <el-select
              v-model="form.reasoningOptions[index]"
              filterable
              allow-create
              default-first-option
              clearable
              placeholder="选择或输入，如 low / medium / high"
              style="flex: 1"
            >
              <el-option
                v-for="opt in commonReasoningOptions"
                :key="opt"
                :label="opt"
                :value="opt"
              />
            </el-select>
            <el-button
              :icon="Delete"
              circle
              @click="removeReasoningOption(index)"
            />
          </div>
          <el-button type="primary" link :icon="Plus" @click="addReasoningOption">
            追加选项
          </el-button>
        </div>
      </el-form-item>
      <el-form-item label="温度调节">
        <el-switch v-model="form.temperature" />
      </el-form-item>
      <el-form-item label="输入上下文限制">
        <el-input-number
          v-model="form.inputContextLimit"
          :min="0"
          :step="1000"
          controls-position="right"
          style="width: 220px"
        />
        <span class="field-hint">token，默认 200000</span>
      </el-form-item>
      <el-form-item label="输出上下文限制">
        <el-input-number
          v-model="form.outputContextLimit"
          :min="0"
          :step="1000"
          controls-position="right"
          style="width: 220px"
        />
        <span class="field-hint">0 表示不限制</span>
      </el-form-item>
      <el-form-item label="输入价格">
        <el-input-number
          v-model="form.inputPrice"
          :min="0"
          :precision="8"
          :step="0.000001"
          controls-position="right"
          style="width: 220px"
        />
      </el-form-item>
      <el-form-item label="输出价格">
        <el-input-number
          v-model="form.outputPrice"
          :min="0"
          :precision="8"
          :step="0.000001"
          controls-position="right"
          style="width: 220px"
        />
      </el-form-item>
      <el-form-item label="缓存价格">
        <el-input-number
          v-model="form.cachePrice"
          :min="0"
          :precision="8"
          :step="0.000001"
          controls-position="right"
          style="width: 220px"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="onConfirm">确定</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.reasoning-options {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.reasoning-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.field-hint {
  margin-left: 12px;
  color: #909399;
  font-size: 12px;
}
</style>
