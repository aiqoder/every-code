<script setup lang="ts">
withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    maxLength?: number
    disabled?: boolean
    type?: 'text' | 'password' | 'url' | 'email'
    invalid?: boolean
  }>(),
  {
    modelValue: '',
    placeholder: '',
    disabled: false,
    type: 'text',
    invalid: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <input
    class="field-input"
    :class="{ invalid }"
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :maxlength="maxLength"
    :disabled="disabled"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
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
