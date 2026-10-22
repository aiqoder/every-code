<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue?: boolean
    disabled?: boolean
    label?: string
  }>(),
  {
    modelValue: false,
    disabled: false,
    label: '',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

function toggle() {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
}
</script>

<template>
  <button
    type="button"
    class="switch"
    :class="{ on: modelValue, disabled }"
    role="switch"
    :aria-checked="modelValue"
    :aria-label="label || undefined"
    :disabled="disabled"
    @click="toggle"
  >
    <span class="knob" />
  </button>
</template>

<style scoped>
.switch {
  width: 32px;
  height: 18px;
  flex-shrink: 0;
  border: 0;
  border-radius: 999px;
  background: var(--ink-600);
  padding: 2px;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  cursor: pointer;
}

.switch.on {
  background: var(--moss-dim);
}

.switch:hover:not(:disabled) {
  background: var(--ink-500);
}

.switch.on:hover:not(:disabled) {
  background: var(--moss);
}

.knob {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--text-strong);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.18);
  transition: transform 0.15s ease;
}

.switch.on .knob {
  transform: translateX(14px);
}

.switch.disabled,
.switch:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
