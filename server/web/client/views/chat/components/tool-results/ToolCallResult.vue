<script setup lang="ts">
import { computed, type Component } from 'vue'
import type { TimelineItem } from '../../../../types'
import type { QuestionAnswerPayload } from '../../../../types'
import AskQuestionResult from './AskQuestionResult.vue'
import DefaultToolResult from './DefaultToolResult.vue'
import TodoWriterResult from './TodoWriterResult.vue'

const props = defineProps<{ item: TimelineItem }>()
const emit = defineEmits<{ answer: [payload: QuestionAnswerPayload] }>()

const views: Record<string, Component> = {
  todowriter: TodoWriterResult,
}

const view = computed(() => {
  if (props.item.toolName === 'AskQuestion' && props.item.question) {
    return AskQuestionResult
  }
  return views[props.item.toolName || ''] || DefaultToolResult
})
</script>

<template>
  <component :is="view" :item="item" @answer="emit('answer', $event)" />
</template>
