<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { onClickOutside } from '@vueuse/core'
import type { SelectOption } from './types'
import { useAnchoredPop } from './useAnchoredPop'

const props = withDefaults(
  defineProps<{
    modelValue: string | number | null
    options: SelectOption[]
    placeholder?: string
    disabled?: boolean
    size?: 'small' | 'medium'
    variant?: 'field' | 'ghost'
    clearable?: boolean
  }>(),
  {
    placeholder: '请选择',
    disabled: false,
    size: 'medium',
    variant: 'field',
    clearable: false
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
  clear: []
}>()

const open = ref(false)
const triggerRef = ref<HTMLElement | null>(null)
const popRef = ref<HTMLElement | null>(null)
const { style: popStyle, prepare } = useAnchoredPop(
  open,
  triggerRef,
  popRef,
  () => props.variant === 'field'
)

onClickOutside(
  triggerRef,
  () => {
    open.value = false
  },
  { ignore: [popRef] }
)

const unavailable = computed(() => props.disabled || props.options.length === 0)
const selected = computed(
  () => props.options.find((option) => option.value === props.modelValue) ?? null
)
const showClear = computed(() => props.clearable && selected.value !== null && !props.disabled)

watch(unavailable, (value) => {
  if (value) open.value = false
})

function toggle(): void {
  if (unavailable.value) return
  if (!open.value) prepare()
  open.value = !open.value
}

function pick(option: SelectOption): void {
  if (option.disabled) return
  emit('update:modelValue', option.value)
  open.value = false
}

function onClear(): void {
  open.value = false
  emit('clear')
}
</script>

<template>
  <div ref="triggerRef" class="ui-select" :class="[size, variant, { open, disabled: unavailable }]">
    <button
      type="button"
      class="trigger"
      :class="{ 'has-clear': showClear }"
      :disabled="unavailable"
      :title="selected?.title || selected?.label || undefined"
      :aria-expanded="open"
      aria-haspopup="listbox"
      @click="toggle"
    >
      <span class="label" :class="{ placeholder: !selected }">
        {{ selected?.label || placeholder }}
      </span>
      <svg class="chevron" viewBox="0 0 16 16" aria-hidden="true">
        <path
          d="M4 6.5 8 10.5 12 6.5"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    </button>
    <button v-if="showClear" type="button" class="clear" aria-label="清除" @click="onClear">
      <svg viewBox="0 0 16 16" aria-hidden="true">
        <path
          d="M4.5 4.5 11.5 11.5M11.5 4.5 4.5 11.5"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
          stroke-linecap="round"
        />
      </svg>
    </button>
    <Teleport to="body">
      <div v-if="open" ref="popRef" class="select-pop ec-stage" role="listbox" :style="popStyle">
        <button
          v-for="option in options"
          :key="`${typeof option.value}:${option.value}`"
          type="button"
          role="option"
          class="select-opt"
          :class="{ on: option.value === modelValue && !option.divided, divided: option.divided }"
          :title="option.title"
          :aria-selected="option.value === modelValue && !option.divided"
          :disabled="option.disabled"
          @click="pick(option)"
        >
          <span class="opt-label">{{ option.label }}</span>
          <svg
            v-if="option.value === modelValue && !option.divided"
            class="check"
            viewBox="0 0 16 16"
            aria-hidden="true"
          >
            <path
              d="M3.5 8.5 6.5 11.5 12.5 4.5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.ui-select {
  position: relative;
  display: block;
  width: 100%;
  min-width: 0;
  max-width: 100%;
}

.trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  box-sizing: border-box;
  width: 100%;
  max-width: 100%;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.ui-select.ghost.small .trigger {
  min-height: 28px;
  padding: 2px 8px;
  font-size: var(--ec-text-meta);
}

.ui-select.ghost.medium .trigger {
  min-height: 32px;
  padding: 4px 10px;
  font-size: var(--ec-text-body);
}

.ui-select.field .trigger {
  border: 1px solid var(--ec-field-border);
  border-radius: 10px;
  background: var(--ec-field);
  color: var(--text);
}

.ui-select.field.small .trigger {
  min-height: 32px;
  padding: 6px 10px;
  font-size: var(--ec-text-body);
}

.ui-select.field.medium .trigger {
  min-height: 40px;
  padding: 10px 12px;
  font-size: var(--ec-text-body);
}

.ui-select:not(.disabled) .trigger:hover,
.ui-select.open .trigger {
  color: var(--text);
  background: var(--surface-hover);
}

.ui-select.field:not(.disabled) .trigger:hover,
.ui-select.field.open .trigger {
  background: var(--ec-field);
  border-color: color-mix(in srgb, var(--ec-fg) 48%, transparent);
}

.trigger:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.label.placeholder {
  color: var(--text-dim);
}

.trigger.has-clear .label {
  margin-right: 22px;
}

.clear {
  position: absolute;
  top: 50%;
  right: 32px;
  z-index: 1;
  display: grid;
  place-items: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--text-muted);
  transform: translateY(-50%);
  cursor: pointer;
}

.ui-select.field.small .clear,
.ui-select.ghost.small .clear {
  right: 30px;
}

.clear:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.clear svg {
  width: 10px;
  height: 10px;
}

.chevron {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  opacity: 0.72;
  transition: transform 0.16s ease;
}

.ui-select.open .chevron {
  transform: rotate(180deg);
}

.ui-select.disabled .trigger {
  opacity: 0.45;
  cursor: not-allowed;
}

.select-pop {
  box-sizing: border-box;
  overflow: auto;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
}

.select-opt {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--ec-text-body);
  text-align: left;
  cursor: pointer;
}

.select-opt.divided {
  margin-top: 5px;
}

.select-opt.divided::before {
  content: '';
  position: absolute;
  top: -5px;
  right: 6px;
  left: 6px;
  height: 1px;
  background: var(--border);
}

.select-opt:hover:not(:disabled) {
  background: var(--surface-hover);
}

.select-opt.on {
  background: var(--accent-soft);
  color: var(--accent);
}

.select-opt:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.opt-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.check {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  color: var(--accent);
}
</style>
