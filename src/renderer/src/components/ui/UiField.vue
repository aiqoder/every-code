<script setup lang="ts">
withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    type?: 'text' | 'password' | 'url' | 'email' | 'number' | 'date' | 'time' | 'datetime-local'
    disabled?: boolean
    readonly?: boolean
    maxlength?: number
    min?: string | number
    max?: string | number
    step?: string | number
    autocomplete?: string
    required?: boolean
    invalid?: boolean
  }>(),
  {
    modelValue: '',
    placeholder: '',
    type: 'text',
    disabled: false,
    readonly: false,
    invalid: false,
    required: false
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [event: Event]
}>()
</script>

<template>
  <input
    class="field-input"
    :class="{ invalid }"
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :maxlength="maxlength"
    :min="min"
    :max="max"
    :step="step"
    :disabled="disabled"
    :readonly="readonly"
    :autocomplete="autocomplete"
    :required="required"
    :aria-invalid="invalid || undefined"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    @change="emit('change', $event)"
  />
</template>

<style scoped>
.field-input {
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  border: 1px solid var(--ec-field-border);
  border-radius: 10px;
  background: var(--ec-field);
  color: var(--text);
  padding: 10px 12px;
  font: inherit;
  font-size: var(--ec-text-body);
  line-height: 1.4;
  caret-color: var(--accent);
  color-scheme: inherit;
}

.field-input:focus {
  outline: none;
  border-color: var(--accent);
}

.field-input.invalid,
.field-input.invalid:focus {
  border-color: var(--error);
}

.field-input:disabled {
  opacity: 0.5;
}

.field-input:read-only {
  cursor: default;
}

.field-input::placeholder {
  color: var(--text-dim);
}
</style>
