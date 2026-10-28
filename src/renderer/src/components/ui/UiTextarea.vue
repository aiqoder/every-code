<script setup lang="ts">
withDefaults(
  defineProps<{
    modelValue?: string
    rows?: number
    placeholder?: string
    disabled?: boolean
    maxlength?: number
    bordered?: boolean
  }>(),
  {
    modelValue: '',
    rows: 3,
    placeholder: '',
    disabled: false,
    bordered: true
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  keydown: [event: KeyboardEvent]
}>()
</script>

<template>
  <textarea
    class="field-area"
    :class="{ plain: !bordered }"
    :value="modelValue"
    :placeholder="placeholder"
    :rows="rows"
    :maxlength="maxlength"
    :disabled="disabled"
    @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
    @keydown="emit('keydown', $event)"
  />
</template>

<style scoped>
.field-area {
  box-sizing: border-box;
  width: 100%;
  border: 1px solid var(--ec-field-border);
  border-radius: 10px;
  background: var(--ec-field);
  color: var(--text);
  padding: 10px 12px;
  font: inherit;
  font-size: var(--ec-text-body);
  line-height: 1.57;
  resize: vertical;
  caret-color: var(--accent);
}

.field-area:focus {
  outline: none;
  border-color: var(--accent);
}

.field-area.plain,
.field-area.plain:focus {
  border: 0;
  border-radius: 0;
  background: transparent;
  outline: none;
  resize: none;
}

.field-area:disabled {
  opacity: 0.5;
}

.field-area::placeholder {
  color: var(--text-dim);
}
</style>
