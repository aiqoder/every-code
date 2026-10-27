<script setup lang="ts">
import { NDropdown, NSwitch } from 'naive-ui'
import {
  describeRule,
  formatDateTime,
  runStatusLabel,
  validityLabel,
  type ScheduledTask
} from '@shared/schedule'

defineProps<{
  task: ScheduledTask
  batch: boolean
  checked: boolean
}>()

const emit = defineEmits<{
  open: []
  run: []
  edit: []
  records: []
  remove: []
  toggle: [enabled: boolean]
  check: [checked: boolean]
}>()

const menu = [
  { label: '查看运行记录', key: 'records' },
  { label: '打开任务', key: 'open' },
  { type: 'divider', key: 'divider' },
  { label: '删除', key: 'remove' }
]

function onMenu(key: string | number): void {
  if (key === 'records') emit('records')
  if (key === 'open') emit('open')
  if (key === 'remove') emit('remove')
}
</script>

<template>
  <article class="rounded-md border border-[var(--ec-border)] ec-skin-raised px-4 py-3">
    <div class="flex items-start gap-3">
      <input
        v-if="batch"
        type="checkbox"
        class="mt-1 h-4 w-4 shrink-0"
        :checked="checked"
        :aria-label="`选择 ${task.name}`"
        @change="emit('check', ($event.target as HTMLInputElement).checked)"
      />
      <button type="button" class="min-w-0 flex-1 text-left" @click="emit('open')">
        <span class="flex items-center gap-2">
          <span class="truncate text-sm">{{ task.name }}</span>
          <span
            v-if="task.lastStatus === 'failed'"
            class="shrink-0 rounded px-1.5 py-0.5 text-[11px] text-[var(--ec-danger)]"
            :title="task.lastError"
          >
            失败
          </span>
          <span
            v-else-if="task.lastStatus === 'running'"
            class="shrink-0 text-[11px] text-[var(--ec-accent)]"
          >
            运行中
          </span>
          <span
            v-else-if="task.lastStatus === 'waiting'"
            class="shrink-0 text-[11px] text-[var(--ec-attention)]"
          >
            等待确认
          </span>
        </span>
        <span class="mt-1 block truncate text-xs text-[var(--ec-fg-muted)]">
          {{ task.agentName || '未命名智能体' }} · {{ describeRule(task.rule) }}
        </span>
        <span
          v-if="validityLabel(task.rule)"
          class="mt-0.5 block text-xs text-[var(--ec-fg-muted)]"
        >
          {{ validityLabel(task.rule) }}
        </span>
        <span class="mt-1 block text-xs">
          <template v-if="!task.enabled">已暂停</template>
          <template v-else-if="task.nextRunAt"
            >下次运行 {{ formatDateTime(task.nextRunAt) }}</template
          >
          <template v-else>没有后续执行</template>
        </span>
        <span
          v-if="task.lastStatus === 'failed'"
          class="mt-1 block truncate text-xs text-[var(--ec-danger)]"
        >
          {{ task.lastError || runStatusLabel('failed') }}
        </span>
      </button>
      <div v-if="!batch" class="flex shrink-0 items-center gap-1">
        <button
          type="button"
          class="action"
          :disabled="task.lastStatus === 'running'"
          @click="emit('run')"
        >
          运行
        </button>
        <button type="button" class="action" @click="emit('edit')">编辑</button>
        <n-switch
          size="small"
          :value="task.enabled"
          :aria-label="task.enabled ? '暂停' : '开启'"
          @update:value="emit('toggle', $event)"
        />
        <n-dropdown trigger="click" :options="menu" @select="onMenu">
          <button type="button" class="action" aria-label="更多">⋯</button>
        </n-dropdown>
      </div>
    </div>
  </article>
</template>

<style scoped>
.action {
  border-radius: 0.375rem;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  color: var(--ec-fg-muted);
}

.action:hover:not(:disabled) {
  background: color-mix(in srgb, var(--ec-fg) 8%, transparent);
  color: var(--ec-fg);
}

.action:disabled {
  opacity: 0.4;
}

input[type='checkbox'] {
  accent-color: var(--ec-accent);
}
</style>
