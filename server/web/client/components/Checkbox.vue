<script setup lang="ts">
import IconCheck from './icons/IconCheck.vue'

const props = withDefaults(
  defineProps<{
    modelValue?: boolean
    disabled?: boolean
    size?: 'default' | 'compact'
    inline?: boolean
  }>(),
  {
    modelValue: false,
    disabled: false,
    size: 'default',
    inline: false,
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
  <label
    class="checkbox"
    :class="[size, { checked: modelValue, disabled, inline }]"
    @click.prevent="toggle"
  >
    <span class="checkbox-box" role="checkbox" :aria-checked="modelValue">
      <IconCheck v-if="modelValue" class="check-icon" />
    </span>
    <span v-if="$slots.default" class="checkbox-label"><slot /></span>
  </label>
</template>

<style scoped>
.checkbox {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
  font-size: 13px;
  color: var(--text);
}

.checkbox.inline {
  margin-right: 12px;
}

.checkbox.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.checkbox-box {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 16px;
  height: 16px;
  border: 1px solid var(--border-strong, var(--border));
  border-radius: 4px;
  background: var(--surface);
  transition:
    border-color 0.12s,
    background 0.12s;
}

.checkbox.compact .checkbox-box {
  width: 14px;
  height: 14px;
  border-radius: 3px;
}

.checkbox.checked .checkbox-box {
  border-color: var(--accent);
  background: var(--accent);
}

.check-icon {
  width: 10px;
  height: 10px;
  color: #fff;
}

.checkbox.compact .check-icon {
  width: 9px;
  height: 9px;
}

.checkbox-label {
  line-height: 1.4;
}
</style>
