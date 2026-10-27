<script setup lang="ts">
import { NModal } from 'naive-ui'
import { describeRule } from '@shared/schedule'
import { scheduleTemplates, type ScheduleTemplate } from '../templates'

defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  pick: [template: ScheduleTemplate]
}>()
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="从模版新建"
    :bordered="false"
    :style="{ width: '40rem', maxWidth: 'calc(100% - 2rem)' }"
    @update:show="emit('update:show', $event)"
  >
    <div class="grid gap-3 sm:grid-cols-3">
      <button
        v-for="item in scheduleTemplates"
        :key="item.id"
        type="button"
        class="rounded-md border border-[var(--ec-border)] px-3 py-3 text-left hover:border-[var(--ec-accent)]"
        @click="emit('pick', item)"
      >
        <span class="block text-sm">{{ item.name }}</span>
        <span class="mt-1 block text-xs leading-5 text-[var(--ec-fg-muted)]">{{
          item.description
        }}</span>
        <span class="mt-3 block text-xs text-[var(--ec-accent)]">{{
          describeRule(item.rule)
        }}</span>
      </button>
    </div>
  </n-modal>
</template>
