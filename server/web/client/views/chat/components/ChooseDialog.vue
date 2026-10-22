<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import Dialog from '../../../components/Dialog.vue'

export type ChooseItem = {
  id: string
  name: string
  description?: string
  badge?: string
}

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    items: ChooseItem[]
    searchPlaceholder?: string
    emptyText?: string
    noMatchText?: string
  }>(),
  {
    searchPlaceholder: '搜索',
    emptyText: '没有可选项',
    noMatchText: '无匹配项',
  },
)

const emit = defineEmits<{
  close: []
  confirm: [ids: string[]]
}>()

const query = ref('')
const checked = ref<string[]>([])
const searchRef = ref<HTMLInputElement | null>(null)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.items
  return props.items.filter((item) => {
    const hay = `${item.name} ${item.description || ''} ${item.badge || ''}`.toLowerCase()
    return hay.includes(q)
  })
})

const checkedIds = computed(() => checked.value.filter((id) => props.items.some((item) => item.id === id)))

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    query.value = ''
    checked.value = []
    await nextTick()
    searchRef.value?.focus()
  },
)

function toggle(id: string) {
  checked.value = checked.value.includes(id)
    ? checked.value.filter((item) => item !== id)
    : [...checked.value, id]
}

function close() {
  emit('close')
}

function confirm() {
  if (!checkedIds.value.length) return
  emit('confirm', [...checkedIds.value])
}
</script>

<template>
  <Dialog :open="open" :title="title" width="420px" @close="close">
    <input
      ref="searchRef"
      v-model="query"
      type="search"
      class="search"
      :placeholder="searchPlaceholder"
      autocomplete="off"
      :aria-label="searchPlaceholder"
      @keydown.enter.prevent="confirm"
    />
    <div class="list">
      <button
        v-for="item in filtered"
        :key="item.id"
        type="button"
        class="row"
        :class="{ on: checked.includes(item.id) }"
        :aria-pressed="checked.includes(item.id)"
        @click="toggle(item.id)"
      >
        <span class="meta">
          <strong class="name">{{ item.name }}</strong>
          <span v-if="item.description" class="desc">{{ item.description }}</span>
        </span>
        <span v-if="item.badge" class="badge">{{ item.badge }}</span>
        <span class="checkbox" aria-hidden="true">{{ checked.includes(item.id) ? '✓' : '' }}</span>
      </button>
      <p v-if="!filtered.length" class="empty">
        {{ query.trim() ? noMatchText : emptyText }}
      </p>
    </div>
    <template #footer>
      <span class="hint">已选 {{ checkedIds.length }} 个</span>
      <button type="button" class="ghost" @click="close">取消</button>
      <button type="button" class="primary" :disabled="!checkedIds.length" @click="confirm">添加</button>
    </template>
  </Dialog>
</template>

<style scoped>
.search {
  width: 100%;
  height: 32px;
  margin: 0 0 8px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  color: var(--text);
  font: inherit;
  font-size: 13px;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 320px;
  overflow: auto;
}

.row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.row:hover,
.row.on {
  background: var(--surface-hover);
}

.row.on {
  background: var(--accent-soft);
}

.meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
}

.row.on .name {
  color: var(--accent);
}

.desc {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  line-height: 16px;
  color: var(--text-dim);
}

.badge {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-dim);
  padding: 1px 6px;
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
}

.checkbox {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  display: grid;
  place-items: center;
  border: 1px solid var(--border-strong, var(--border));
  border-radius: 4px;
  font-size: 12px;
  color: var(--accent);
}

.row.on .checkbox {
  border-color: var(--accent);
}

.empty {
  margin: 0;
  padding: 24px 8px;
  color: var(--text-dim);
  font-size: 12px;
  text-align: center;
}

.hint {
  color: var(--text-dim);
  font-size: 12px;
}
</style>
