<script setup lang="ts">
import { computed, ref } from 'vue'
import { onClickOutside } from '@vueuse/core'
import IconCheck from './icons/IconCheck.vue'
import IconChevronDown from './icons/IconChevronDown.vue'

export interface SelectOption {
  value: number
  label: string
  hint?: string
  icon?: string
}

const props = withDefaults(
  defineProps<{
    modelValue: number | null
    options: SelectOption[]
    placeholder?: string
    ariaLabel?: string
    disabled?: boolean
    invalid?: boolean
    variant?: 'toolbar' | 'field'
    placement?: 'top' | 'bottom'
    showIcon?: boolean
  }>(),
  {
    placeholder: '请选择',
    ariaLabel: '下拉选择',
    disabled: false,
    invalid: false,
    variant: 'toolbar',
    placement: 'top',
    showIcon: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const open = ref(false)
const wrapRef = ref<HTMLElement | null>(null)

onClickOutside(wrapRef, () => {
  open.value = false
})

const selected = computed(() => props.options.find((o) => o.value === props.modelValue))

const triggerLabel = computed(() => selected.value?.label || props.placeholder)

function toggle() {
  if (props.disabled || !props.options.length) return
  open.value = !open.value
}

function pick(value: number) {
  emit('update:modelValue', value)
  open.value = false
}

function iconFallback(opt: SelectOption) {
  const t = opt.label.trim()
  return t ? t.slice(0, 1) : '?'
}
</script>

<template>
  <div
    ref="wrapRef"
    class="select-wrap"
    :class="[variant, placement, { open, invalid, 'with-icon': showIcon, disabled: disabled || !options.length }]"
  >
    <button
      type="button"
      class="select-trigger"
      :aria-label="ariaLabel"
      :aria-expanded="open"
      :disabled="disabled || !options.length"
      @click="toggle"
    >
      <span
        v-if="showIcon && selected"
        class="select-icon"
        aria-hidden="true"
      >
        <img v-if="selected.icon" :src="selected.icon" alt="" />
        <span v-else>{{ iconFallback(selected) }}</span>
      </span>
      <span class="select-label" :class="{ placeholder: !selected }">{{ triggerLabel }}</span>
      <IconChevronDown class="chevron" />
    </button>
    <Transition name="select-pop">
      <div v-if="open" class="select-pop" role="listbox" :aria-label="ariaLabel">
        <button
          v-for="opt in options"
          :key="opt.value"
          type="button"
          role="option"
          class="select-opt"
          :class="{ on: opt.value === modelValue }"
          :aria-selected="opt.value === modelValue"
          @click="pick(opt.value)"
        >
          <span v-if="showIcon" class="select-icon opt-icon" aria-hidden="true">
            <img v-if="opt.icon" :src="opt.icon" alt="" />
            <span v-else>{{ iconFallback(opt) }}</span>
          </span>
          <span class="opt-main">
            <strong>{{ opt.label }}</strong>
            <small v-if="opt.hint">{{ opt.hint }}</small>
          </span>
          <IconCheck v-if="opt.value === modelValue" class="check" />
        </button>
        <p v-if="!options.length" class="select-empty">暂无选项</p>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.select-wrap {
  position: relative;
  min-width: 0;
  max-width: 220px;
  flex-shrink: 1;
}

.select-wrap.with-icon:not(.field) {
  max-width: 236px;
}

.select-wrap.with-icon:not(.field) .select-trigger {
  gap: 6px;
  padding-left: 4px;
}

.select-wrap.field {
  max-width: none;
  flex: 1;
  width: 100%;
}

.select-trigger {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  min-height: 28px;
  padding: 2px 6px 2px 8px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font: inherit;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition:
    color 0.12s,
    background 0.12s,
    border-color 0.12s;
}

.select-wrap.field .select-trigger {
  width: 100%;
  min-height: 40px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  color: var(--text);
  font-size: 14px;
  font-weight: 400;
}

.select-wrap:not(.disabled) .select-trigger:hover,
.select-wrap.open .select-trigger {
  color: var(--text);
  background: var(--surface-hover);
  border-color: var(--border);
}

.select-wrap.field:not(.disabled) .select-trigger:hover,
.select-wrap.field.open .select-trigger {
  background: var(--surface);
  border-color: var(--border-strong, var(--border));
}

.select-wrap.field.invalid .select-trigger {
  border-color: var(--error);
}

.select-icon {
  display: grid;
  place-items: center;
  width: 16px;
  height: 16px;
  border-radius: 4px;
  overflow: hidden;
  flex-shrink: 0;
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 10px;
  font-weight: 600;
  line-height: 1;
}

.select-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.select-wrap.field .select-icon {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  font-size: 12px;
}

.opt-icon {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  font-size: 11px;
}

.select-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  text-align: left;
}

.select-label.placeholder {
  color: var(--text-dim);
}

.chevron {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  opacity: 0.72;
  transition: transform 0.16s ease;
}

.select-wrap.open .chevron {
  transform: rotate(180deg);
}

.select-wrap.disabled .select-trigger {
  opacity: 0.45;
  cursor: not-allowed;
}

.select-pop {
  position: absolute;
  left: 0;
  z-index: 24;
  min-width: min(240px, 72vw);
  max-width: min(320px, 86vw);
  max-height: 280px;
  overflow: auto;
  padding: 4px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
}

.select-wrap.toolbar.top .select-pop,
.select-wrap.top .select-pop {
  bottom: calc(100% + 6px);
}

.select-wrap.bottom .select-pop {
  top: calc(100% + 6px);
  bottom: auto;
}

.select-wrap.field .select-pop {
  min-width: 100%;
  max-width: none;
}

.select-opt {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  text-align: left;
  cursor: pointer;
  transition: background 0.12s;
}

.select-opt:hover {
  background: var(--surface-hover);
}

.select-opt.on {
  background: var(--accent-soft);
}

.opt-main {
  min-width: 0;
  flex: 1;
  display: grid;
  gap: 1px;
}

.opt-main strong {
  font-size: 13px;
  font-weight: 500;
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.select-opt.on .opt-main strong {
  color: var(--accent);
}

.opt-main small {
  font-size: 11px;
  color: var(--text-dim);
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

.select-empty {
  margin: 0;
  padding: 10px 12px;
  color: var(--text-dim);
  font-size: 12px;
}

.select-pop-enter-active,
.select-pop-leave-active {
  transition:
    opacity 0.14s ease,
    transform 0.14s ease;
}

.select-pop-enter-from,
.select-pop-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

.select-wrap.bottom .select-pop-enter-from,
.select-wrap.bottom .select-pop-leave-to {
  transform: translateY(-4px);
}
</style>
