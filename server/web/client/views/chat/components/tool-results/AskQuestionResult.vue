<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import type { QuestionAnswerPayload, TimelineItem } from '../../../../types'
import { OTHER_OPTION_ID } from '../../askQuestion'

const props = defineProps<{ item: TimelineItem }>()
const emit = defineEmits<{ answer: [payload: QuestionAnswerPayload] }>()

const selected = reactive<Record<string, string[]>>({})
const otherText = reactive<Record<string, string>>({})
const localError = ref('')

const question = computed(() => props.item.question)
const locked = computed(() => question.value?.status !== 'pending')

function isChecked(questionId: string, optionId: string) {
  return (selected[questionId] || []).includes(optionId)
}

function toggle(questionId: string, optionId: string, multiple: boolean) {
  if (locked.value) return
  localError.value = ''
  const cur = selected[questionId] ? [...selected[questionId]] : []
  if (!multiple) {
    selected[questionId] = [optionId]
    return
  }
  const idx = cur.indexOf(optionId)
  if (idx >= 0) cur.splice(idx, 1)
  else cur.push(optionId)
  selected[questionId] = cur
}

function submit() {
  const q = question.value
  if (!q || locked.value) return
  const answers: QuestionAnswerPayload['answers'] = []
  for (const item of q.questions) {
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
      text: picked.includes(OTHER_OPTION_ID) ? text : undefined,
    })
  }
  localError.value = ''
  emit('answer', { toolCallId: props.item.id, questionId: q.questionId, answers })
}
</script>

<template>
  <form v-if="question" class="ask" @submit.prevent="submit">
    <h3 v-if="question.title" class="title">{{ question.title }}</h3>
    <fieldset v-for="item in question.questions" :key="item.id" class="q">
      <legend>{{ item.prompt }}</legend>
      <label v-for="opt in item.options" :key="opt.id" class="opt">
        <input
          :type="item.allowMultiple ? 'checkbox' : 'radio'"
          :name="item.id"
          :checked="isChecked(item.id, opt.id)"
          :disabled="locked"
          @change="toggle(item.id, opt.id, item.allowMultiple)"
        />
        <span>{{ opt.label }}</span>
      </label>
      <label class="opt">
        <input
          :type="item.allowMultiple ? 'checkbox' : 'radio'"
          :name="item.id"
          :checked="isChecked(item.id, OTHER_OPTION_ID)"
          :disabled="locked"
          @change="toggle(item.id, OTHER_OPTION_ID, item.allowMultiple)"
        />
        <span>其他</span>
      </label>
      <input
        v-if="isChecked(item.id, OTHER_OPTION_ID)"
        v-model="otherText[item.id]"
        class="other"
        type="text"
        placeholder="请填写"
        :disabled="locked"
      />
    </fieldset>
    <p v-if="localError || question.error" class="err">{{ localError || question.error }}</p>
    <button type="submit" class="submit" :disabled="locked">
      {{
        question.status === 'submitting'
          ? '提交中…'
          : question.status === 'done'
            ? '已提交'
            : question.status === 'timeout'
              ? '已超时'
              : '提交'
      }}
    </button>
  </form>
</template>

<style scoped>
.ask {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin: 8px 0 4px;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--accent-soft);
}

.title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.q {
  margin: 0;
  padding: 0;
  border: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

legend {
  margin-bottom: 4px;
  font-size: 13px;
  color: var(--text);
}

.opt {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-muted);
}

.other {
  margin-left: 22px;
  padding: 6px 8px;
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  background: transparent;
  color: var(--text);
}

.submit {
  align-self: flex-start;
  padding: 6px 14px;
  border: 0;
  border-radius: 999px;
  background: var(--accent);
  color: #fff;
  cursor: pointer;
}

.submit:disabled {
  opacity: 0.6;
  cursor: default;
}

.err {
  margin: 0;
  font-size: 12px;
  color: var(--error);
}
</style>
