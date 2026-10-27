<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NInput, NSelect, type SelectOption } from 'naive-ui'
import type { AgentItem, ModelItem } from '@/api/catalog'

const props = defineProps<{
  agents: AgentItem[]
  models: ModelItem[]
  agentId: string
  modelId: number | null
  workspace: string
  sending: boolean
  seed?: string
  seedKey?: number
}>()

const emit = defineEmits<{
  'update:agentId': [value: string]
  'update:modelId': [value: number]
  send: [text: string]
  stop: []
  pickWorkspace: []
}>()

const draft = ref('')

watch(
  () => props.seedKey,
  () => {
    if (props.seed) draft.value = props.seed
  }
)

const agentOptions = computed<SelectOption[]>(() =>
  props.agents
    .filter((agent) => agent.agentId)
    .map((agent) => ({
      label: agent.name,
      value: agent.agentId
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
  <form class="px-4 pt-2 pb-4" @submit.prevent="submit">
    <div
      class="composer-card ec-skin-raised mx-auto max-w-3xl rounded-2xl px-3 pt-1 pb-2 shadow-[0_8px_24px_rgba(0,0,0,0.18)]"
    >
      <n-input
        v-model:value="draft"
        type="textarea"
        :rows="2"
        :bordered="false"
        :resizable="false"
        placeholder="今天帮你做些什么？"
        class="select-text"
        @keydown="onKeydown"
      />
      <div class="mt-1 flex items-center gap-1">
        <div class="composer-select w-36 shrink-0">
          <n-select
            size="small"
            :value="agentId || null"
            :options="agentOptions"
            placeholder="智能体"
            @update:value="onAgentChange"
          />
        </div>
        <button
          type="button"
          class="max-w-40 truncate rounded-md px-2 py-1 text-xs text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
          @click="emit('pickWorkspace')"
        >
          {{ workspace || '工作目录' }}
        </button>
        <span class="flex-1" />
        <div class="composer-select w-44 shrink-0">
          <n-select
            size="small"
            :value="modelId"
            :options="modelOptions"
            placeholder="模型"
            @update:value="onModelChange"
          />
        </div>
        <button
          v-if="sending"
          type="button"
          class="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-[var(--ec-fg)] text-[var(--ec-canvas)]"
          aria-label="停止"
          @click="emit('stop')"
        >
          <span class="h-2.5 w-2.5 rounded-[2px] bg-current" />
        </button>
        <button
          v-else
          type="submit"
          class="grid h-8 w-8 shrink-0 place-items-center rounded-full disabled:cursor-default"
          :class="
            draft.trim()
              ? 'bg-[var(--ec-accent)] text-[var(--ec-on-accent)]'
              : 'bg-[color-mix(in_srgb,var(--ec-fg)_12%,transparent)] text-[var(--ec-fg-muted)]'
          "
          aria-label="发送"
          :disabled="!draft.trim()"
        >
          <svg viewBox="0 0 16 16" class="h-4 w-4" fill="none" aria-hidden="true">
            <path
              d="M8 12.5V3.5M8 3.5 4.5 7M8 3.5 11.5 7"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>
    </div>
  </form>
</template>

<style scoped>
.composer-card :deep(.n-input),
.composer-card :deep(.n-input-wrapper),
.composer-card :deep(.n-input__textarea-el),
.composer-card :deep(.n-input__placeholder),
.composer-card :deep(.n-base-selection),
.composer-card :deep(.n-base-selection-label) {
  background-color: transparent !important;
  box-shadow: none !important;
}

.composer-select :deep(.n-base-selection) {
  --n-border: transparent;
  --n-border-hover: transparent;
  --n-border-focus: transparent;
  --n-border-active: transparent;
  --n-box-shadow-focus: none;
  --n-box-shadow-active: none;
  --n-color: transparent;
  --n-color-active: transparent;
}

.composer-select :deep(.n-base-selection__border),
.composer-select :deep(.n-base-selection__state-border) {
  border-color: transparent !important;
  box-shadow: none !important;
}
</style>
