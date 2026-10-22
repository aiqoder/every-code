<script setup lang="ts">
withDefaults(
  defineProps<{
    label?: string
    disabled?: boolean
    accept?: string
  }>(),
  {
    label: '选择文件',
    disabled: false,
  },
)

const emit = defineEmits<{
  pick: [file: File]
}>()

function onChange(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (file) emit('pick', file)
}
</script>

<template>
  <label class="file-picker" :class="{ disabled }">
    <input type="file" hidden :accept="accept" :disabled="disabled" @change="onChange" />
    <slot>{{ label }}</slot>
  </label>
</template>

<style scoped>
.file-picker {
  display: inline-flex;
  align-items: center;
  border: 1px dashed var(--border);
  background: transparent;
  color: var(--text-muted);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-size: 12px;
  cursor: pointer;
}

.file-picker.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
