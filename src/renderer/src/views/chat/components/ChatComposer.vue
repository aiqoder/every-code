<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import UiAvatar from '@/components/ui/UiAvatar.vue'
import UiSelect from '@/components/ui/UiSelect.vue'
import UiTextarea from '@/components/ui/UiTextarea.vue'
import type { SelectOption } from '@/components/ui/types'
import type { AgentItem, ModelItem } from '@/api/catalog'
import type { LoopPackage } from '@shared/loop'

const props = defineProps<{
  agents: AgentItem[]
  packages: LoopPackage[]
  agentLocked: boolean
  models: ModelItem[]
  agentId: string
  modelId: number | null
  sending: boolean
  seed?: string
  seedKey?: number
}>()

const emit = defineEmits<{
  'update:agentId': [value: string]
  'update:modelId': [value: number]
  send: [text: string]
  stop: []
}>()

const draft = ref('')

watch(
  () => props.seedKey,
  () => {
    if (props.seed) draft.value = props.seed
  }
)

function avatarOf(agent: AgentItem): string {
  const pkg = props.packages.find((item) => item.remoteId != null && item.remoteId === agent.id)
  return pkg?.avatar || ''
}

const agentOptions = computed<SelectOption[]>(() =>
  props.agents
    .filter((agent) => agent.agentId)
    .map((agent) => ({
      label: agent.name,
      value: agent.agentId,
      avatar: avatarOf(agent)
    }))
)

const currentAgent = computed(
  () => agentOptions.value.find((item) => item.value === props.agentId) || null
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

function onAgentChange(value: string | number | null): void {
  if (typeof value === 'string') {
    emit('update:agentId', value)
    return
  }
  if (typeof value === 'number') emit('update:agentId', String(value))
}

function onModelChange(value: string | number | null): void {
  const parsed = typeof value === 'number' ? value : typeof value === 'string' ? Number(value) : NaN
  if (Number.isFinite(parsed)) emit('update:modelId', parsed)
}
</script>

<template>
  <form class="px-4 pt-2 pb-4" @submit.prevent="submit">
    <div
      class="composer-card mx-auto max-w-3xl rounded-2xl border px-3 pt-1 pb-2"
    >
      <UiTextarea
        v-model="draft"
        :rows="2"
        :bordered="false"
        placeholder="想干什么就告诉我？"
        class="select-text"
        @keydown="onKeydown"
      />
      <div class="mt-1 flex items-center gap-1">
        <div
          v-if="agentLocked"
          class="flex h-8 max-w-48 min-w-0 shrink-0 items-center gap-1.5 px-2.5 text-sm text-[var(--ec-fg)]"
          :title="currentAgent?.label"
        >
          <UiAvatar
            v-if="currentAgent"
            :name="currentAgent.label"
            :image="currentAgent.avatar"
          />
          <span class="truncate">{{ currentAgent?.label || '智能体' }}</span>
        </div>
        <div v-else class="min-w-0 max-w-48 shrink">
          <UiSelect
            variant="ghost"
            size="medium"
            :model-value="agentId || null"
            :options="agentOptions"
            placeholder="智能体"
            @update:model-value="onAgentChange"
          />
        </div>
        <span class="flex-1" />
        <div class="min-w-0 max-w-48 shrink">
          <UiSelect
            variant="ghost"
            size="medium"
            :model-value="modelId"
            :options="modelOptions"
            placeholder="模型"
            @update:model-value="onModelChange"
          />
        </div>
        <button
          v-if="sending"
          type="button"
          class="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-[var(--ec-fg)] text-[var(--ec-canvas)] hover:brightness-110"
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
              ? 'bg-[var(--ec-accent)] text-[var(--ec-on-accent)] hover:bg-[var(--accent-hover)]'
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
.composer-card {
  background: var(--ec-field);
  border-color: var(--ec-field-border);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.22);
}

.composer-card:focus-within {
  border-color: var(--ec-accent);
}
</style>
