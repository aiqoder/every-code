<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'

import type { LlmProvider, LlmProviderPayload } from '@/types/api'

const visible = defineModel<boolean>({ default: false })

const props = defineProps<{
  editing?: LlmProvider | null
  submit: (payload: LlmProviderPayload) => Promise<void>
}>()

const emit = defineEmits<{
  success: []
}>()

const formRef = ref<FormInstance>()
const submitting = ref(false)
const form = reactive({
  name: '',
  website: '',
  apiKey: '',
  openaiCompatibleBaseUrl: '',
  anthropicBaseUrl: '',
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入供应商名称', trigger: 'blur' }],
}

const dialogTitle = ref('新建供应商')

watch(visible, (v) => {
  if (!v) {
    formRef.value?.resetFields()
    return
  }
  if (props.editing) {
    dialogTitle.value = '编辑供应商'
    form.name = props.editing.name
    form.website = props.editing.website || ''
    form.apiKey = props.editing.apiKey || ''
    form.openaiCompatibleBaseUrl = props.editing.openaiCompatibleBaseUrl || ''
    form.anthropicBaseUrl = props.editing.anthropicBaseUrl || ''
  } else {
    dialogTitle.value = '新建供应商'
    form.name = ''
    form.website = ''
    form.apiKey = ''
    form.openaiCompatibleBaseUrl = ''
    form.anthropicBaseUrl = ''
  }
})

async function onConfirm() {
  if (!formRef.value) return
  await formRef.value.validate()
  submitting.value = true
  try {
    const payload = {
      name: form.name,
      website: form.website,
      apiKey: form.apiKey,
      openaiCompatibleBaseUrl: form.openaiCompatibleBaseUrl,
      anthropicBaseUrl: form.anthropicBaseUrl,
    }
    await props.submit(payload)
    ElMessage.success(props.editing ? '更新成功' : '创建成功')
    visible.value = false
    emit('success')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <el-dialog v-model="visible" :title="dialogTitle" width="560px" destroy-on-close>
    <el-form ref="formRef" :model="form" :rules="rules" label-width="160px">
      <el-form-item label="供应商名称" prop="name">
        <el-input v-model="form.name" placeholder="如 OpenAI / Anthropic" maxlength="128" />
      </el-form-item>
      <el-form-item label="官方网站" prop="website">
        <el-input v-model="form.website" placeholder="https://..." maxlength="512" />
      </el-form-item>
      <el-form-item label="API Key" prop="apiKey">
        <el-input
          v-model="form.apiKey"
          type="password"
          show-password
          placeholder="可选，供应商 API Key"
          maxlength="512"
        />
      </el-form-item>
      <el-form-item label="OpenAI Compatible BaseUrl" prop="openaiCompatibleBaseUrl">
        <el-input
          v-model="form.openaiCompatibleBaseUrl"
          placeholder="可选，OpenAI 兼容接口地址"
          maxlength="512"
        />
      </el-form-item>
      <el-form-item label="Anthropic BaseUrl" prop="anthropicBaseUrl">
        <el-input
          v-model="form.anthropicBaseUrl"
          placeholder="可选，Anthropic 接口地址"
          maxlength="512"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="onConfirm">确定</el-button>
    </template>
  </el-dialog>
</template>
