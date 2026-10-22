<script setup lang="ts">
const props = defineProps<{
  modelValue?: string | number | boolean
  value: string | number | boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string | number | boolean]
  change: [value: string | number | boolean]
}>()

function pick() {
  if (props.disabled || props.modelValue === props.value) return
  emit('update:modelValue', props.value)
  emit('change', props.value)
}
</script>

<template>
  <label class="radio" :class="{ checked: modelValue === value, disabled }" @click.prevent="pick">
    <span class="radio-dot" role="radio" :aria-checked="modelValue === value" :aria-disabled="disabled || undefined" />
    <span v-if="$slots.default" class="radio-label"><slot /></span>
  </label>
</template>

<style scoped>
.radio {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  max-width: 100%;
  cursor: pointer;
  user-select: none;
  font-size: 13px;
  color: var(--text);
}

.radio.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.radio-dot {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  margin-top: 2px;
  width: 16px;
  height: 16px;
  border: 1px solid var(--border-strong, var(--border));
  border-radius: 50%;
  background: var(--surface);
  transition:
    border-color 0.12s,
    background 0.12s;
}

.radio.checked .radio-dot {
  border-color: var(--accent);
}

.radio.checked .radio-dot::after {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
}

.radio-label {
  min-width: 0;
  line-height: 1.4;
}
</style>
