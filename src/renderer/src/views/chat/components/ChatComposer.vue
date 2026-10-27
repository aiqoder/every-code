<script setup lang="ts">
import { computed, ref } from 'vue'
import { NButton, NInput, NSelect, type SelectOption } from 'naive-ui'
import type { AgentItem, ModelItem } from '@/api/catalog'

const props = defineProps<{
  agents: AgentItem[]
  models: ModelItem[]
  agentId: string
  modelId: number | null
  workspace: string
  sending: boolean
}>()

const emit = defineEmits<{
  'update:agentId': [value: string]
  'update:modelId': [value: number]
  send: [text: string]
  stop: []
  pickWorkspace: []
}>()

const draft = ref('')

const agentOptions = computed<SelectOption[]>(() =>
  props.agents.map((agent) => ({
    label: agent.name,
    value: String(agent.id)
  }))
)

const modelOptions = computed<SelectOption[]>(() =>
  props.models.map((model) => ({
    label: model.providerName ? `${model.providerName} / ${model.name}` : model.name,
    value: model.id
  }))
)

function submit(): void {
  const text = draft.value.trim()
  if (!text || props.sending) return
  emit('send', text)
  draft.value = ''
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    submit()
  }
}

function onAgentChange(value: string | number | Array<string | number> | null): void {
  if (typeof value === 'string') {
    emit('update:agentId', value)
    return
  }
  if (typeof value === 'number') emit('update:agentId', String(value))
}

function onModelChange(value: string | number | Array<string | number> | null): void {
  const parsed = typeof value === 'number' ? value : typeof value === 'string' ? Number(value) : NaN
  if (Number.isFinite(parsed)) emit('update:modelId', parsed)
}
</script>

<template>
  <form
    class="border-t border-[var(--ec-border)] bg-[var(--ec-canvas-subtle)] px-4 py-3"
    @submit.prevent="submit"
  >
    <div class="mx-auto max-w-3xl">
      <n-input
        v-model:value="draft"
        type="textarea"
        :rows="3"
        :resizable="false"
        placeholder="描述你要做的事。Enter 发送，Shift+Enter 换行"
        class="select-text"
        @keydown="onKeydown"
      />
      <div class="mt-2 flex flex-wrap items-center gap-2">
        <div class="w-40 shrink-0">
          <n-select
            size="small"
            :value="agentId || null"
            :options="agentOptions"
            placeholder="选择智能体"
            @update:value="onAgentChange"
          />
        </div>
        <div class="w-56 shrink-0">
          <n-select
            size="small"
            :value="modelId"
            :options="modelOptions"
            placeholder="选择模型"
            @update:value="onModelChange"
          />
        </div>
        <n-button size="small" class="max-w-56" @click="emit('pickWorkspace')">
          <span class="inline-block max-w-52 truncate align-bottom">
            {{ workspace || '选择工作目录' }}
          </span>
        </n-button>
        <span class="flex-1" />
        <n-button v-if="sending" size="small" @click="emit('stop')">停止</n-button>
        <n-button
          v-else
          size="small"
          type="primary"
          attr-type="submit"
          :disabled="!draft.trim()"
        >
          发送
        </n-button>
      </div>
    </div>
  </form>
</template>
