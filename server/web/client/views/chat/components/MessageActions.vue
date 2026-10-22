<script setup lang="ts">
import { ref } from 'vue'
import IconCopy from './icons/IconCopy.vue'
import IconEdit from './icons/IconEdit.vue'
import IconBookmark from './icons/IconBookmark.vue'

const props = withDefaults(
  defineProps<{
    role: 'user' | 'assistant'
    canEdit?: boolean
    canFavorite?: boolean
    disabled?: boolean
  }>(),
  { canFavorite: true },
)

const emit = defineEmits<{
  copy: []
  edit: []
  favorite: []
}>()

const copied = ref(false)

async function onCopy() {
  emit('copy')
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 1500)
}
</script>

<template>
  <div class="actions" :class="role">
    <button type="button" class="act" :disabled="disabled" title="复制" @click="onCopy">
      <IconCopy />
      <span v-if="copied" class="tip">已复制</span>
    </button>
    <template v-if="role === 'user'">
      <button
        v-if="canEdit"
        type="button"
        class="act"
        :disabled="disabled"
        title="编辑"
        @click="emit('edit')"
      >
        <IconEdit />
      </button>
      <button
        v-if="canFavorite"
        type="button"
        class="act"
        :disabled="disabled"
        title="收藏"
        @click="emit('favorite')"
      >
        <IconBookmark />
      </button>
    </template>
  </div>
</template>

<style scoped>
.actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  min-height: 22px;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease-in-out;
}

.actions.assistant {
  justify-content: flex-start;
}

.act {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
}

.act svg {
  width: 14px;
  height: 14px;
}

.act:hover:not(:disabled) {
  color: var(--text);
  background: var(--surface-hover);
}

.act:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.tip {
  position: absolute;
  bottom: calc(100% + 4px);
  left: 50%;
  transform: translateX(-50%);
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--bg-elevated);
  color: var(--text);
  font-size: 11px;
  white-space: nowrap;
  box-shadow: var(--shadow);
}
</style>
