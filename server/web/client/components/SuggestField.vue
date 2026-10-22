<script setup lang="ts">
import { computed, ref } from 'vue'
import { onClickOutside } from '@vueuse/core'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    options?: string[]
    placeholder?: string
    maxLength?: number
    disabled?: boolean
    invalid?: boolean
    ariaLabel?: string
  }>(),
  {
    modelValue: '',
    options: () => [],
    placeholder: '',
    disabled: false,
    invalid: false,
    ariaLabel: '输入建议',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const open = ref(false)
const wrapRef = ref<HTMLElement | null>(null)
const activeIndex = ref(-1)
// 仅在用户敲击键盘输入时为 true；focus 打开或选中项后为 false，此时展示全量列表，
// 避免选中后再次点开只剩当前值匹配的唯一选项。
const typing = ref(false)

onClickOutside(wrapRef, () => {
  open.value = false
  activeIndex.value = -1
  typing.value = false
})

const filteredOptions = computed(() => {
  const items = props.options.filter(Boolean)
  const query = props.modelValue.trim().toLowerCase()
  if (!query || !typing.value) return items
  return items.filter((item) => item.toLowerCase().includes(query))
})

const showSuggestions = computed(
  () => open.value && !props.disabled && filteredOptions.value.length > 0,
)

function onFocus() {
  if (!props.disabled && props.options.length) {
    typing.value = false
    open.value = true
  }
}

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
  typing.value = true
  open.value = true
  activeIndex.value = -1
}

function pick(value: string) {
  emit('update:modelValue', value)
  typing.value = false
  open.value = false
  activeIndex.value = -1
}

function onKeydown(event: KeyboardEvent) {
  if (!showSuggestions.value) return
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    activeIndex.value = (activeIndex.value + 1) % filteredOptions.value.length
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    activeIndex.value =
      (activeIndex.value - 1 + filteredOptions.value.length) % filteredOptions.value.length
  } else if (event.key === 'Enter' && activeIndex.value >= 0) {
    event.preventDefault()
    const option = filteredOptions.value[activeIndex.value]
    if (option) pick(option)
  } else if (event.key === 'Escape') {
    open.value = false
    activeIndex.value = -1
  }
}
</script>

<template>
  <div ref="wrapRef" class="suggest-wrap" :class="{ open: showSuggestions, disabled }">
    <input
      class="field-input"
      :class="{ invalid }"
      type="text"
      :value="modelValue"
      :placeholder="placeholder"
      :maxlength="maxLength"
      :disabled="disabled"
      :aria-label="ariaLabel"
      :aria-expanded="showSuggestions"
      aria-autocomplete="list"
      role="combobox"
      @focus="onFocus"
      @input="onInput"
      @keydown="onKeydown"
    />
    <Transition name="suggest-pop">
      <ul v-if="showSuggestions" class="suggest-pop" role="listbox" :aria-label="ariaLabel">
        <li
          v-for="(opt, index) in filteredOptions"
          :key="opt"
          role="option"
          class="suggest-opt"
          :class="{ active: index === activeIndex }"
          :aria-selected="index === activeIndex"
          @mousedown.prevent="pick(opt)"
        >
          {{ opt }}
        </li>
      </ul>
    </Transition>
  </div>
</template>

<style scoped>
.suggest-wrap {
  position: relative;
  width: 100%;
}

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

.suggest-wrap.open .field-input {
  border-color: var(--border-strong, var(--border));
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

.suggest-pop {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 6px);
  z-index: 24;
  margin: 0;
  padding: 4px;
  list-style: none;
  max-height: 240px;
  overflow: auto;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
}

.suggest-opt {
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  color: var(--text);
  cursor: pointer;
  transition: background 0.12s;
}

.suggest-opt:hover,
.suggest-opt.active {
  background: var(--surface-hover);
}

.suggest-pop-enter-active,
.suggest-pop-leave-active {
  transition:
    opacity 0.14s ease,
    transform 0.14s ease;
}

.suggest-pop-enter-from,
.suggest-pop-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
