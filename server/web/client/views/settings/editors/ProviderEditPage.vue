<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useClientPort } from '../../../ports/clientPort'
import FormField from '../../../components/FormField.vue'
import TextField from '../../../components/TextField.vue'
import {
  createFieldErrors,
  hasFieldErrors,
  optionalUrl,
  required,
} from '../../../lib/formValidation'
import {
  protocolLabel,
  providerPresets,
  type ProviderPreset,
  type ProviderProtocol,
} from '../../../lib/providerPresets'
import type { LlmProvider, ProviderPayload } from '../../../types'
import EditorsBreadcrumb from '../components/EditorsBreadcrumb.vue'

type ProviderField = 'name' | 'website' | 'baseUrl'

const route = useRoute()
const router = useRouter()
const port = useClientPort()

const providerId = Number(route.params.id) || 0
const editing = providerId > 0

const form = reactive({
  name: '',
  website: '',
  apiKey: '',
  protocol: 'openai' as ProviderProtocol,
  baseUrl: '',
})
const loading = ref(true)
const err = ref('')
const saving = ref(false)
const fieldErrors = reactive(createFieldErrors<ProviderField>(['name', 'website', 'baseUrl']))

const baseUrlPlaceholder = computed(() =>
  form.protocol === 'anthropic' ? 'https://api.anthropic.com' : 'https://api.openai.com/v1',
)

function applyPreset(preset: ProviderPreset) {
  form.name = preset.name
  form.website = preset.website
  form.protocol = preset.protocol
  form.baseUrl = preset.baseUrl
  clearFieldError('name')
  clearFieldError('baseUrl')
}

function clearFieldError(field: ProviderField) {
  fieldErrors[field] = ''
}

function validateForm(): boolean {
  fieldErrors.name = required(form.name, '请输入供应商名称')
  fieldErrors.website = optionalUrl(form.website, '请输入有效的 URL')
  fieldErrors.baseUrl = required(form.baseUrl, '请输入接口地址') || optionalUrl(form.baseUrl, '请输入有效的 URL')
  return !hasFieldErrors(fieldErrors)
}

async function load() {
  if (!editing) {
    loading.value = false
    return
  }
  loading.value = true
  err.value = ''
  try {
    const list = await port.listProviders()
    const row = (list ?? []).find((p) => p.id === providerId)
    if (row) {
      form.name = row.name
      form.website = row.website || ''
      form.apiKey = row.apiKey || ''
      if (row.anthropicBaseUrl && !row.openaiCompatibleBaseUrl) {
        form.protocol = 'anthropic'
        form.baseUrl = row.anthropicBaseUrl
      } else {
        form.protocol = 'openai'
        form.baseUrl = row.openaiCompatibleBaseUrl || ''
      }
    }
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push({ name: 'settings-models', query: route.query })
}

async function submit() {
  if (!validateForm()) return
  saving.value = true
  err.value = ''
  const payload: ProviderPayload = {
    name: form.name.trim(),
    website: form.website.trim(),
    apiKey: form.apiKey.trim(),
    openaiCompatibleBaseUrl: form.protocol === 'openai' ? form.baseUrl.trim() : '',
    anthropicBaseUrl: form.protocol === 'anthropic' ? form.baseUrl.trim() : '',
  }
  try {
    if (editing) {
      await port.updateProvider(providerId, payload)
    } else {
      await port.createProvider(payload)
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
        { label: editing ? '编辑供应商' : '新建供应商' },
      ]"
    />

    <form class="dlg" @submit.prevent="submit">
      <p v-if="err" class="err">{{ err }}</p>

      <FormField v-if="!editing" label="从模板快速创建">
        <div class="presets">
          <button
            v-for="preset in providerPresets"
            :key="preset.name"
            type="button"
            class="preset"
            :class="{ active: form.name === preset.name }"
            @click="applyPreset(preset)"
          >
            {{ preset.name }}
          </button>
        </div>
      </FormField>

      <FormField label="供应商名称" required :error="fieldErrors.name">
        <TextField
          v-model="form.name"
          placeholder="如 OpenAI / DeepSeek"
          :invalid="!!fieldErrors.name"
          @update:model-value="clearFieldError('name')"
        />
      </FormField>

      <FormField label="API 协议">
        <div class="protocols">
          <label
            v-for="p in (['openai', 'anthropic'] as ProviderProtocol[])"
            :key="p"
            class="protocol"
            :class="{ active: form.protocol === p }"
          >
            <input v-model="form.protocol" type="radio" name="protocol" :value="p" />
            {{ protocolLabel(p) }}
          </label>
        </div>
      </FormField>

      <FormField label="接口地址" required :error="fieldErrors.baseUrl">
        <TextField
          v-model="form.baseUrl"
          type="url"
          :placeholder="baseUrlPlaceholder"
          :invalid="!!fieldErrors.baseUrl"
          @update:model-value="clearFieldError('baseUrl')"
        />
      </FormField>

      <FormField label="API Key">
        <TextField
          v-model="form.apiKey"
          type="password"
          placeholder="可选，供应商 API Key"
        />
      </FormField>

      <FormField label="官方网站" :error="fieldErrors.website">
        <TextField
          v-model="form.website"
          type="url"
          placeholder="https://...（可选）"
          :invalid="!!fieldErrors.website"
          @update:model-value="clearFieldError('website')"
        />
      </FormField>

      <div class="actions">
        <button type="button" @click="goBack">取消</button>
        <button type="submit" class="go" :disabled="saving || loading">
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.editor-page {
  width: 100%;
  max-width: 560px;
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

.presets {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.preset {
  font: inherit;
  font-size: 12px;
  padding: 6px 14px;
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text-muted);
  border-radius: var(--radius-full);
}

.preset:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.preset.active {
  color: var(--accent);
  border-color: var(--accent);
}

.protocols {
  display: flex;
  gap: 8px;
}

.protocol {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-muted);
  padding: 8px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.protocol:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.protocol.active {
  color: var(--accent);
  border-color: var(--accent);
}

.protocol input {
  display: none;
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
