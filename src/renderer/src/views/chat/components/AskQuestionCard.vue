<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue'
import { OTHER_OPTION_ID, isAskTimeoutResult } from '../askQuestion'
import type { QuestionAnswerPayload, ToolTimelineItem } from '../types'

const props = defineProps<{ item: ToolTimelineItem }>()
const emit = defineEmits<{
  answer: [payload: QuestionAnswerPayload]
}>()

const selected = reactive<Record<string, string[]>>({})
const otherText = reactive<Record<string, string>>({})
const localError = ref('')
const countdown = ref('')
let timer: ReturnType<typeof setInterval> | undefined

const question = computed(() => props.item.question)
const locked = computed(() => question.value?.status !== 'pending')
const buttonLabel = computed(() => {
  switch (question.value?.status) {
    case 'submitting':
      return '提交中…'
    case 'done':
      return '已提交'
    case 'timeout':
      return '已超时'
    default:
      return '提交'
  }
})
const shownError = computed(() => localError.value || question.value?.error || '')
const idleLabel = computed(() => {
  if (props.item.pending) return '正在准备问题…'
  if (isAskTimeoutResult(props.item.result)) return '已超时'
  return '已提交'
})

watch(
  () => [props.item.result, question.value?.status] as const,
  () => {
    const status = question.value?.status
    if (status === 'pending' || status === 'submitting') return
    restoreAnswers(props.item.result)
  },
  { immediate: true }
)

watch(
  () => [question.value?.deadlineAt, question.value?.status] as const,
  () => {
    tick()
    if (timer) clearInterval(timer)
    timer = undefined
    const status = question.value?.status
    if (
      typeof question.value?.deadlineAt === 'number' &&
      (status === 'pending' || status === 'submitting')
    ) {
      timer = setInterval(tick, 1000)
    }
  },
  { immediate: true }
)

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

function tick(): void {
  const deadline = question.value?.deadlineAt
  const status = question.value?.status
  if (typeof deadline !== 'number' || (status !== 'pending' && status !== 'submitting')) {
    countdown.value = ''
    return
  }
  const sec = Math.max(0, Math.ceil((deadline - Date.now()) / 1000))
  const minutes = Math.floor(sec / 60)
  const seconds = sec % 60
  countdown.value = `${minutes}:${String(seconds).padStart(2, '0')}`
}

function restoreAnswers(result: string): void {
  if (!result.trim()) return
  let parsed: unknown
  try {
    parsed = JSON.parse(result)
  } catch {
    return
  }
  if (!parsed || typeof parsed !== 'object') return
  const answers = (parsed as { answers?: unknown }).answers
  if (!Array.isArray(answers)) return
  for (const answer of answers) {
    if (!answer || typeof answer !== 'object') continue
    const row = answer as { questionId?: unknown; selected?: unknown; text?: unknown }
    const id = typeof row.questionId === 'string' ? row.questionId : ''
    if (!id || !Array.isArray(row.selected)) continue
    selected[id] = row.selected.filter((option): option is string => typeof option === 'string')
    if (typeof row.text === 'string') otherText[id] = row.text
  }
}

function isChecked(questionId: string, optionId: string): boolean {
  return (selected[questionId] || []).includes(optionId)
}

function toggle(questionId: string, optionId: string, multiple: boolean): void {
  if (locked.value) return
  localError.value = ''
  const current = selected[questionId] ? [...selected[questionId]] : []
  if (!multiple) {
    selected[questionId] = [optionId]
    return
  }
  const index = current.indexOf(optionId)
  if (index >= 0) current.splice(index, 1)
  else current.push(optionId)
  selected[questionId] = current
}

function submit(): void {
  const current = question.value
  if (!current || locked.value) return
  const answers: QuestionAnswerPayload['answers'] = []
  for (const item of current.questions) {
    const picked = selected[item.id] || []
    if (!picked.length) {
      localError.value = '请完成所有题目'
      return
    }
    const text = (otherText[item.id] || '').trim()
    if (picked.includes(OTHER_OPTION_ID) && !text) {
      localError.value = '请填写「其他」的内容'
      return
    }
    answers.push({
      questionId: item.id,
      selected: picked,
      text: picked.includes(OTHER_OPTION_ID) ? text : undefined
    })
  }
  localError.value = ''
  emit('answer', { toolCallId: props.item.id, questionId: current.questionId, answers })
}
</script>

<template>
  <form
    v-if="question && question.questions.length"
    class="ec-skin-raised rounded-xl border border-[var(--ec-border)] px-3.5 py-3"
    @submit.prevent="submit"
  >
    <div class="flex items-baseline justify-between gap-3">
      <h3 class="min-w-0 text-sm font-semibold text-[var(--ec-fg)]">
        {{ question.title || '需要你确认' }}
      </h3>
      <span v-if="countdown" class="shrink-0 font-mono text-xs text-[var(--ec-fg-muted)]"
        >剩余 {{ countdown }}</span
      >
    </div>
    <fieldset v-for="entry in question.questions" :key="entry.id" class="mt-3 border-0 p-0">
      <legend class="mb-1.5 text-[13px] text-[var(--ec-fg)]">{{ entry.prompt }}</legend>
      <div class="grid gap-1.5">
        <label
          v-for="opt in entry.options"
          :key="opt.id"
          class="flex items-center gap-2 text-[13px] text-[var(--ec-fg-muted)]"
        >
          <input
            :type="entry.allowMultiple ? 'checkbox' : 'radio'"
            :name="`${item.id}-${entry.id}`"
            :checked="isChecked(entry.id, opt.id)"
            :disabled="locked"
            @change="toggle(entry.id, opt.id, entry.allowMultiple)"
          />
          <span>{{ opt.label }}</span>
        </label>
        <label class="flex items-center gap-2 text-[13px] text-[var(--ec-fg-muted)]">
          <input
            :type="entry.allowMultiple ? 'checkbox' : 'radio'"
            :name="`${item.id}-${entry.id}`"
            :checked="isChecked(entry.id, OTHER_OPTION_ID)"
            :disabled="locked"
            @change="toggle(entry.id, OTHER_OPTION_ID, entry.allowMultiple)"
          />
          <span>其他</span>
        </label>
        <input
          v-if="isChecked(entry.id, OTHER_OPTION_ID)"
          v-model="otherText[entry.id]"
          type="text"
          class="ec-skin-inset ml-6 rounded-lg border border-[var(--ec-border)] px-2 py-1 text-[13px] text-[var(--ec-fg)] outline-none disabled:opacity-60"
          placeholder="请填写"
          :disabled="locked"
          @input="localError = ''"
        />
      </div>
    </fieldset>
    <p v-if="shownError" class="mt-2 text-xs text-[var(--ec-danger)]">{{ shownError }}</p>
    <button
      type="submit"
      class="mt-3 rounded-full bg-[var(--ec-accent)] px-3.5 py-1.5 text-xs text-[var(--ec-on-accent)] disabled:cursor-default disabled:opacity-60"
      :disabled="locked"
    >
      {{ buttonLabel }}
    </button>
  </form>
  <div
    v-else
    class="ec-skin-raised rounded-xl border border-[var(--ec-border)] px-3 py-2 text-xs text-[var(--ec-fg-muted)]"
  >
    AskQuestion · {{ idleLabel }}
  </div>
</template>
