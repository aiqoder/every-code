<script setup lang="ts">
import UiDialog from '@/components/ui/UiDialog.vue'
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
  <UiDialog
    :show="show"
    title="从模版新建"
    width="40rem"
    @update:show="emit('update:show', $event)"
  >
    <div class="grid gap-3 sm:grid-cols-3">
      <button
        v-for="item in scheduleTemplates"
        :key="item.id"
        type="button"
        class="rounded-md border border-[var(--ec-border)] px-3 py-3 text-left hover:border-[var(--ec-accent)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_6%,transparent)]"
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
  </UiDialog>
</template>
