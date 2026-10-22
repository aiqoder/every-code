<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { PendingAskQuestion } from '../../../shared/types'
import Radio from './Radio.vue'
import TextField from './TextField.vue'

const OTHER = '__other__'

const props = defineProps<{
  question: PendingAskQuestion | null
}>()

const emit = defineEmits<{
  answer: [
    payload: {
      conversationId: number
      questionId: string
      answers: Array<{ questionId: string; selected: string[]; text?: string }>
    }
  ]
  dismiss: []
}>()

const selected = reactive<Record<string, string[]>>({})
const otherText = reactive<Record<string, string>>({})
const localError = ref('')
const remainSec = ref(0)
const step = ref(0)
const sending = ref(false)
let tick: ReturnType<typeof setInterval> | null = null

const locked = computed(() => !props.question || props.question.status !== 'pending')
const items = computed(() => props.question?.questions ?? [])
const current = computed(() => items.value[step.value] ?? null)
const last = computed(() => step.value >= items.value.length - 1)

watch(
  () => props.question?.questionId,
  () => {
    Object.keys(selected).forEach((k) => delete selected[k])
    Object.keys(otherText).forEach((k) => delete otherText[k])
    localError.value = ''
    step.value = 0
    sending.value = false
    restartTick()
  },
  { immediate: true }
)

function restartTick(): void {
  if (tick) clearInterval(tick)
  tick = null
  const q = props.question
  if (!q?.deadlineAt) {
    remainSec.value = 0
    return
  }
  const update = (): void => {
    remainSec.value = Math.max(0, Math.ceil((q.deadlineAt - Date.now()) / 1000))
  }
  update()
  tick = setInterval(update, 1000)
}

function isChecked(qid: string, oid: string): boolean {
  return (selected[qid] || []).includes(oid)
}

function toggle(qid: string, oid: string, multiple: boolean): void {
  if (locked.value) return
  localError.value = ''
  if (!multiple) {
    selected[qid] = [oid]
    return
  }
  const cur = selected[qid] ? [...selected[qid]] : []
  const idx = cur.indexOf(oid)
  if (idx >= 0) cur.splice(idx, 1)
  else cur.push(oid)
  selected[qid] = cur
}

function answerOf(itemId: string): { questionId: string; selected: string[]; text?: string } | null {
  const picked = selected[itemId] || []
  if (!picked.length) {
    localError.value = '请先选一项'
    return null
  }
  const text = (otherText[itemId] || '').trim()
  if (picked.includes(OTHER) && !text) {
    localError.value = '请填写「其他」的内容'
    return null
  }
  return {
    questionId: itemId,
    selected: picked,
    text: picked.includes(OTHER) ? text : undefined
  }
}

function next(): void {
  const item = current.value
  const q = props.question
  if (!item || !q || locked.value || sending.value) return
  if (!answerOf(item.id)) return
  localError.value = ''
  if (!last.value) {
    step.value += 1
    return
  }
  const answers: Array<{ questionId: string; selected: string[]; text?: string }> = []
  for (const one of q.questions) {
    const answer = answerOf(one.id)
    if (!answer) {
      step.value = q.questions.indexOf(one)
      return
    }
    answers.push(answer)
  }
  sending.value = true
  emit('answer', {
    conversationId: q.conversationId,
    questionId: q.questionId,
    answers
  })
}

watch(
  () => props.question?.status,
  (status) => {
    if (status === 'pending') sending.value = false
  }
)

function back(): void {
  if (step.value <= 0 || locked.value) return
  localError.value = ''
  step.value -= 1
}

const remainLabel = computed(() => {
  const s = remainSec.value
  if (s <= 0) return '即将超时'
  const m = Math.floor(s / 60)
  const r = s % 60
  return m > 0 ? `${m} 分 ${r} 秒` : `${r} 秒`
})
</script>

<template>
  <aside v-if="question && current" class="ask-panel">
    <header class="ask-head">
      <h3>{{ current.prompt }}</h3>
      <span class="remain mono">
        <template v-if="items.length > 1">{{ step + 1 }}/{{ items.length }} · </template>{{ remainLabel }}
      </span>
    </header>

    <form class="ask-form" @submit.prevent="next">
      <fieldset class="q">
        <template v-if="current.allow_multiple">
          <label v-for="opt in current.options" :key="opt.id" class="opt" :class="{ on: isChecked(current.id, opt.id) }">
            <input
              type="checkbox"
              :name="current.id"
              :checked="isChecked(current.id, opt.id)"
              :disabled="locked"
              @change="toggle(current.id, opt.id, true)"
            />
            <span>{{ opt.label }}</span>
          </label>
          <label class="opt" :class="{ on: isChecked(current.id, OTHER) }">
            <input
              type="checkbox"
              :name="current.id"
              :checked="isChecked(current.id, OTHER)"
              :disabled="locked"
              @change="toggle(current.id, OTHER, true)"
            />
            <span>其他</span>
          </label>
        </template>
        <template v-else>
          <Radio
            v-for="opt in current.options"
            :key="opt.id"
            :model-value="isChecked(current.id, opt.id) ? opt.id : ''"
            :value="opt.id"
            :disabled="locked"
            @update:model-value="toggle(current.id, opt.id, false)"
          >
            {{ opt.label }}
          </Radio>
          <Radio
            :model-value="isChecked(current.id, OTHER) ? OTHER : ''"
            :value="OTHER"
            :disabled="locked"
            @update:model-value="toggle(current.id, OTHER, false)"
          >
            其他
          </Radio>
        </template>
        <TextField
          v-if="isChecked(current.id, OTHER)"
          v-model="otherText[current.id]"
          class="other"
          placeholder="请填写"
          :disabled="locked"
        />
      </fieldset>

      <p v-if="localError || question.error" class="err">{{ localError || question.error }}</p>

      <div class="ask-actions">
        <button v-if="step > 0" type="button" class="btn ghost sm" :disabled="locked" @click="back">上一题</button>
        <button type="submit" class="btn primary sm" :disabled="locked">
          {{
            question.status === 'submitting'
              ? '提交中…'
              : question.status === 'done'
                ? '已提交'
                : question.status === 'timeout'
                  ? '已超时'
                  : last
                    ? '提交'
                    : '继续'
          }}
        </button>
      </div>
    </form>
  </aside>
</template>

<style scoped>
.ask-panel {
  margin: 0;
  padding: 10px 10px 8px;
  border: 1px solid var(--line);
  border-radius: var(--r-md);
  background: var(--ink-850);
}

.ask-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.ask-head h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
  color: var(--text-strong);
}

.remain {
  flex: none;
  font-size: 11px;
  color: var(--text-faint);
}

.ask-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.q {
  margin: 0;
  padding: 0;
  border: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

legend {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin-bottom: 4px;
  font-size: 14px;
  font-weight: 400;
  line-height: 1.5;
  color: var(--text-strong);
}

.q-index {
  flex: none;
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-faint);
}

.opt,
.q :deep(.radio) {
  align-items: flex-start;
  gap: 8px;
  margin: 0;
  padding: 5px 8px;
  border-radius: var(--r-sm);
  font-size: 14px;
  font-weight: 400;
  line-height: 1.5;
  color: var(--text);
}

.opt {
  display: flex;
  cursor: pointer;
}

.opt:hover,
.q :deep(.radio:hover) {
  background: var(--ink-750);
}

.opt.on,
.q :deep(.radio.checked) {
  background: var(--brass-glow);
}

.opt input {
  margin-top: 2px;
}

.q :deep(.radio-label) {
  overflow-wrap: anywhere;
}

.q .other {
  margin: 4px 0 2px 28px;
}

.err {
  margin: 0;
  font-size: 12px;
  color: var(--rust);
}

.ask-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
