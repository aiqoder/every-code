<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'

import type { Tool, ToolPayload } from '@/types/api'
import JsonEditor from '@/components/JsonEditor.vue'

const visible = defineModel<boolean>({ default: false })

const props = defineProps<{
  editing?: Tool | null
  submit: (payload: ToolPayload) => Promise<void>
}>()

const emit = defineEmits<{
  success: []
}>()

const formRef = ref<FormInstance>()
const submitting = ref(false)
const parametersInvalid = ref(false)

const form = reactive({
  name: '',
  displayName: '',
  description: '',
  type: 'http' as 'http' | 'builtin',
  parametersText: '{\n  "type": "object",\n  "properties": {}\n}',
  url: '',
})

const rules = computed<FormRules>(() => ({
  name: [
    { required: true, message: '请输入英文名称', trigger: 'blur' },
    {
      pattern: /^[a-zA-Z][a-zA-Z0-9_]*$/,
      message: '名称只能使用英文字母、数字和下划线，且以字母开头',
      trigger: 'blur',
    },
  ],
  displayName: [{ required: true, message: '请输入中文名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择工具类型', trigger: 'change' }],
  url:
    form.type === 'http'
      ? [{ required: true, message: '请输入 HTTP URL', trigger: 'blur' }]
      : [],
  parametersText: [
    {
      validator: (_rule, value, callback) => {
        try {
          JSON.parse(String(value || '{}'))
          parametersInvalid.value = false
          callback()
        } catch {
          parametersInvalid.value = true
          callback(new Error('参数 JSON 格式无效'))
        }
      },
      trigger: 'change',
    },
  ],
}))

const dialogTitle = computed(() => (props.editing ? '编辑工具' : '新增工具'))
const isHTTP = computed(() => form.type === 'http')
const typeHint = computed(() => {
  if (form.type === 'builtin') {
    return '系统内置工具名称须为 todowriter、Task 或 AskQuestion。'
  }
  return '请求固定为 POST，请求体自动包含英文名称、中文名称、说明、参数 JSON 及模型调用参数。'
})

function resetForm() {
  form.name = ''
  form.displayName = ''
  form.description = ''
  form.type = 'http'
  form.parametersText = '{\n  "type": "object",\n  "properties": {}\n}'
  form.url = ''
  parametersInvalid.value = false
}

function fillForm(row: Tool) {
  form.name = row.name
  form.displayName = row.displayName
  form.description = row.description || ''
  form.type = row.type === 'builtin' ? 'builtin' : 'http'
  form.parametersText = JSON.stringify(row.parameters || {}, null, 2)
  form.url = String(row.extra?.url || '')
}

function parseJSON(text: string, label: string) {
  try {
    return JSON.parse(text || '{}')
  } catch {
    throw new Error(`${label} JSON 格式无效`)
  }
}

function onParametersChange() {
  parametersInvalid.value = false
  formRef.value?.validateField('parametersText').catch(() => {})
}

async function onSubmit() {
  const ok = await formRef.value?.validate().catch(() => false)
  if (!ok) return

  submitting.value = true
  try {
    const parameters = parseJSON(form.parametersText, '参数')
    const payload: ToolPayload = {
      name: form.name.trim(),
      displayName: form.displayName.trim(),
      description: form.description.trim(),
      parameters,
      type: form.type,
      extra:
        form.type === 'http'
          ? {
              method: 'POST',
              url: form.url.trim(),
            }
          : {},
    }
    await props.submit(payload)
    ElMessage.success(props.editing ? '更新成功' : '创建成功')
    visible.value = false
    emit('success')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    submitting.value = false
  }
}

watch(visible, (v) => {
  if (!v) return
  if (props.editing) {
    fillForm(props.editing)
  } else {
    resetForm()
  }
  formRef.value?.clearValidate()
})
</script>

<template>
  <el-dialog v-model="visible" :title="dialogTitle" width="720px" destroy-on-close>
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-form-item label="工具类型" prop="type">
        <el-select v-model="form.type" :disabled="!!editing" style="width: 100%">
          <el-option label="HTTP" value="http" />
          <el-option label="系统内置" value="builtin" />
        </el-select>
      </el-form-item>
      <el-form-item label="英文名称" prop="name">
        <el-input v-model="form.name" placeholder="如 get_weather" :disabled="!!editing" />
      </el-form-item>
      <el-form-item label="中文名称" prop="displayName">
        <el-input v-model="form.displayName" placeholder="用于展示的名称" />
      </el-form-item>
      <el-form-item label="说明">
        <el-input v-model="form.description" type="textarea" :rows="2" placeholder="工具用途说明" />
      </el-form-item>
      <el-form-item label="参数 JSON" prop="parametersText">
        <JsonEditor
          v-model="form.parametersText"
          :invalid="parametersInvalid"
          @update:model-value="onParametersChange"
        />
      </el-form-item>
      <template v-if="isHTTP">
        <el-divider content-position="left">HTTP 配置</el-divider>
        <el-form-item label="URL" prop="url">
          <el-input v-model="form.url" placeholder="https://api.example.com/invoke" />
        </el-form-item>
      </template>
      <p class="hint">{{ typeHint }}</p>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="onSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.hint {
  margin: 0 0 0 100px;
  color: #909399;
  font-size: 13px;
}
</style>
