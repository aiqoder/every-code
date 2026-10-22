<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

export type ConnectorPick = {
  id: string
  name: string
  description?: string
}

const props = defineProps<{
  connectors: ConnectorPick[]
  open: boolean
  multiple?: boolean
}>()

const emit = defineEmits<{
  select: [item: ConnectorPick]
  confirm: [items: ConnectorPick[]]
  close: []
}>()

const checked = ref<string[]>([])

const query = ref('')
const searchRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLUListElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const anchorRef = ref<HTMLElement | null>(null)
const panelStyle = ref<Record<string, string>>({})
const activeIndex = ref(0)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.connectors
  return props.connectors.filter((item) => item.name.toLowerCase().includes(q) || (item.description || '').toLowerCase().includes(q))
})

function place() {
  const wrap = anchorRef.value?.parentElement || anchorRef.value
  if (!wrap) return
  const rect = wrap.getBoundingClientRect()
  const width = Math.min(320, Math.round(window.innerWidth * 0.72))
  const gap = 8
  const pad = 12
  let left = rect.width >= width ? rect.left : rect.right - width
  left = Math.max(pad, Math.min(left, window.innerWidth - width - pad))
  const bottom = Math.max(pad, window.innerHeight - rect.top + gap)
  panelStyle.value = {
    left: `${Math.round(left)}px`,
    bottom: `${Math.round(bottom)}px`,
    width: `${width}px`,
  }
}

function onDocPointerDown(e: PointerEvent) {
  if (!props.open) return
  const t = e.target
  if (!(t instanceof Node)) return
  if (panelRef.value?.contains(t)) return
  const wrap = anchorRef.value?.parentElement
  if (wrap?.contains(t)) return
  emit('close')
}

function bindPlaceListeners() {
  window.addEventListener('resize', place)
  window.addEventListener('scroll', place, true)
  document.addEventListener('pointerdown', onDocPointerDown, true)
}

function unbindPlaceListeners() {
  window.removeEventListener('resize', place)
  window.removeEventListener('scroll', place, true)
  document.removeEventListener('pointerdown', onDocPointerDown, true)
}

watch(
  () => props.open,
  async (v) => {
    unbindPlaceListeners()
    if (!v) return
    query.value = ''
    checked.value = []
    activeIndex.value = 0
    place()
    bindPlaceListeners()
    await nextTick()
    place()
    searchRef.value?.focus()
  },
)

watch(filtered, () => {
  activeIndex.value = 0
})

function pick(item: ConnectorPick) {
  if (props.multiple) {
    checked.value = checked.value.includes(item.id)
      ? checked.value.filter((id) => id !== item.id)
      : [...checked.value, item.id]
    return
  }
  emit('select', item)
}

function confirm() {
  const picked = props.connectors.filter((item) => checked.value.includes(item.id))
  if (!picked.length) return
  emit('confirm', picked)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    emit('close')
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (!filtered.value.length) return
    activeIndex.value = (activeIndex.value + 1) % filtered.value.length
    return
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    if (!filtered.value.length) return
    activeIndex.value = (activeIndex.value - 1 + filtered.value.length) % filtered.value.length
    return
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    if (props.multiple) {
      confirm()
      return
    }
    const hit = filtered.value[activeIndex.value]
    if (hit) pick(hit)
  }
}

onMounted(() => {
  if (props.open) {
    place()
    bindPlaceListeners()
    searchRef.value?.focus()
  }
})

onBeforeUnmount(() => {
  unbindPlaceListeners()
})
</script>

<template>
  <span ref="anchorRef" class="connector-picker-anchor" aria-hidden="true" />
  <Teleport to="body">
    <div v-if="open" ref="panelRef" class="connector-picker" :style="panelStyle" @keydown="onKeydown">
      <input ref="searchRef" v-model="query" type="search" class="search" placeholder="搜索连接器…" autocomplete="off" />
      <ul ref="listRef">
        <li
          v-for="(item, i) in filtered"
          :key="item.id"
          :class="{ on: i === activeIndex, picked: multiple && checked.includes(item.id) }"
          @mousedown.prevent="pick(item)"
          @mouseenter="activeIndex = i"
        >
          <strong>{{ item.name }}</strong>
          <span v-if="item.description">{{ item.description }}</span>
        </li>
        <li v-if="!filtered.length" class="empty">没有可添加的连接器</li>
      </ul>
      <div v-if="multiple" class="foot">
        <button type="button" @click="emit('close')">取消</button>
        <button type="button" class="ok" :disabled="!checked.length" @click="confirm">确认</button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.connector-picker-anchor {
  display: none;
}

.connector-picker {
  position: fixed;
  z-index: 200;
  max-height: 360px;
  display: flex;
  flex-direction: column;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.18);
  overflow: hidden;
}

.search {
  margin: 0;
  border: 0;
  border-bottom: 1px solid var(--border);
  background: transparent;
  color: var(--text);
  padding: 10px 12px;
  font: inherit;
  font-size: 13px;
  outline: none;
}

ul {
  list-style: none;
  margin: 0;
  padding: 4px;
  overflow: auto;
  max-height: 300px;
}

li {
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

li.on,
li:hover:not(.empty) {
  background: var(--surface-hover);
}

li.empty {
  color: var(--text-dim);
  font-size: 13px;
  cursor: default;
  text-align: center;
}

li strong {
  display: block;
  font-size: 13px;
  font-weight: 560;
}

li span {
  display: block;
  margin-top: 2px;
  color: var(--text-muted);
  font-size: 12px;
}

li.picked {
  background: var(--surface);
}

.foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 8px;
  border-top: 1px solid var(--border);
}

.foot button {
  font: inherit;
  font-size: 13px;
  height: 30px;
  padding: 0 12px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--bg-elevated);
  color: var(--text);
  cursor: pointer;
}

.foot .ok {
  border: 0;
  background: var(--text);
  color: var(--bg-elevated);
}

.foot .ok:disabled {
  background: var(--surface-hover);
  color: var(--text-dim);
  cursor: default;
}
</style>
