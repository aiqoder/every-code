<script setup lang="ts">
import { ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue?: number
    min?: number
    max?: number
    step?: number
    disabled?: boolean
    invalid?: boolean
    placeholder?: string
  }>(),
  {
    modelValue: 0,
    disabled: false,
    invalid: false,
    placeholder: '',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

const draft = ref(formatValue(props.modelValue))

watch(
  () => props.modelValue,
  (value) => {
    const next = formatValue(value)
    if (Number(draft.value) !== value || draft.value === '') {
      draft.value = next
    }
  },
)

function formatValue(value: number): string {
  return Number.isFinite(value) ? String(value) : '0'
}

function clamp(value: number): number {
  let next = value
  if (props.min != null && next < props.min) next = props.min
  if (props.max != null && next > props.max) next = props.max
  return next
}

function commit(raw: string) {
  const trimmed = raw.trim()
  if (!trimmed) {
    const fallback = props.min ?? 0
    draft.value = String(fallback)
    emit('update:modelValue', fallback)
    return
  }
  const parsed = Number(trimmed)
  if (Number.isNaN(parsed)) {
    draft.value = formatValue(props.modelValue)
    return
  }
  const next = clamp(parsed)
  draft.value = formatValue(next)
  emit('update:modelValue', next)
}

function onInput(event: Event) {
  draft.value = (event.target as HTMLInputElement).value
}

function onBlur() {
  commit(draft.value)
}
</script>

<template>
  <input
    class="field-input"
    :class="{ invalid }"
    type="text"
    inputmode="decimal"
    :value="draft"
    :placeholder="placeholder"
    :disabled="disabled"
    @input="onInput"
    @blur="onBlur"
  />
</template>

<style scoped>
.field-input {
  width: 100%;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
  border-radius: 10px;
  padding: 10px 12px;
  font: inherit;
  font-size: 14px;
}

.field-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.field-input.invalid {
  border-color: var(--error);
}

.field-input:disabled {
  opacity: 0.5;
}

.field-input::placeholder {
  color: var(--text-dim);
}
</style>
