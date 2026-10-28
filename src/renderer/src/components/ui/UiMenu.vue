<script setup lang="ts">
import { ref } from 'vue'
import { onClickOutside } from '@vueuse/core'
import type { MenuOption } from './types'
import { useAnchoredPop } from './useAnchoredPop'

const props = defineProps<{
  options: MenuOption[]
}>()

const emit = defineEmits<{
  select: [key: string | number]
}>()

const open = ref(false)
const triggerRef = ref<HTMLElement | null>(null)
const popRef = ref<HTMLElement | null>(null)
const { style: popStyle, prepare } = useAnchoredPop(open, triggerRef, popRef, () => false)

onClickOutside(
  triggerRef,
  () => {
    open.value = false
  },
  { ignore: [popRef] }
)

function toggle(): void {
  if (!open.value) prepare()
  open.value = !open.value
}

function choose(option: MenuOption): void {
  if (option.type === 'divider') return
  emit('select', option.key)
  open.value = false
}
</script>

<template>
  <div ref="triggerRef" class="menu-trigger" @click="toggle">
    <slot />
    <Teleport to="body">
      <div v-if="open" ref="popRef" class="menu-pop ec-stage" role="menu" :style="popStyle" @click.stop>
        <template v-for="option in props.options" :key="option.key">
          <div v-if="option.type === 'divider'" class="divider" role="separator" />
          <button
            v-else
            type="button"
            class="item"
            :class="{ danger: option.danger }"
            role="menuitem"
            @click="choose(option)"
          >
            {{ option.label }}
          </button>
        </template>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.menu-trigger {
  display: inline-flex;
  max-width: 100%;
}

.menu-pop {
  box-sizing: border-box;
  min-width: 160px;
  overflow: auto;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
}

.item {
  display: block;
  width: 100%;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--ec-text-body);
  line-height: 1.35;
  text-align: left;
  white-space: nowrap;
  cursor: pointer;
}

.item:hover {
  background: var(--surface-hover);
}

.item.danger {
  color: var(--error);
}

.item.danger:hover {
  background: color-mix(in srgb, var(--error) 12%, transparent);
}

.divider {
  height: 1px;
  margin: 4px 6px;
  background: var(--border);
}
</style>
