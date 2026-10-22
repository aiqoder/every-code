<script setup lang="ts">
withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    rows?: number
    maxLength?: number
    disabled?: boolean
  }>(),
  {
    modelValue: '',
    placeholder: '',
    rows: 3,
    disabled: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <textarea
    class="field-area"
    :value="modelValue"
    :placeholder="placeholder"
    :rows="rows"
    :maxlength="maxLength"
    :disabled="disabled"
    @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
  />
</template>

<style scoped>
.field-area {
  width: 100%;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
  border-radius: 10px;
  padding: 10px 12px;
  font: inherit;
  font-size: 13px;
  resize: vertical;
  line-height: 1.55;
}

.field-area:disabled {
  opacity: 0.5;
}

.field-area::placeholder {
  color: var(--text-dim);
}
</style>
