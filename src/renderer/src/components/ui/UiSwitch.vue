<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    value: boolean
    disabled?: boolean
    size?: 'small' | 'medium'
    ariaLabel?: string
  }>(),
  {
    disabled: false,
    size: 'medium'
  }
)

const emit = defineEmits<{
  'update:value': [value: boolean]
}>()

function toggle(): void {
  if (props.disabled) return
  emit('update:value', !props.value)
}
</script>

<template>
  <button
    type="button"
    class="ui-switch"
    :class="[size, { on: value }]"
    role="switch"
    :aria-checked="value"
    :aria-label="ariaLabel"
    :disabled="disabled"
    @click="toggle"
  >
    <span class="knob" />
  </button>
</template>

<style scoped>
.ui-switch {
  position: relative;
  flex-shrink: 0;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: var(--border-strong);
  cursor: pointer;
  transition: background 0.15s;
}

.ui-switch.on {
  background: var(--accent);
}

.ui-switch.small {
  width: 28px;
  height: 16px;
}

.ui-switch.medium {
  width: 36px;
  height: 20px;
}

.knob {
  position: absolute;
  top: 2px;
  left: 2px;
  border-radius: 50%;
  background: #fff;
  transition: transform 0.15s;
}

.ui-switch.small .knob {
  width: 12px;
  height: 12px;
}

.ui-switch.medium .knob {
  width: 16px;
  height: 16px;
}

.ui-switch.small.on .knob {
  transform: translateX(12px);
}

.ui-switch.medium.on .knob {
  transform: translateX(16px);
}

.ui-switch:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.ui-switch:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
</style>
