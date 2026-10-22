<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { SkillRef } from '../../../lib/skillToken'
import { filterSkills } from '../../../lib/skillToken'

const props = defineProps<{
  skills: SkillRef[]
  open: boolean
  /** 勾选多项，点确认后一次加入。 */
  multiple?: boolean
}>()

const emit = defineEmits<{
  select: [skill: SkillRef]
  confirm: [skills: SkillRef[]]
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

const filtered = computed(() => filterSkills(props.skills, query.value))

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

// 键盘上下移动高亮时，只滚动技能列表自身，让当前项保持可见
watch(activeIndex, async () => {
  await nextTick()
  const list = listRef.value
  const item = list?.querySelectorAll('li')[activeIndex.value]
  if (!list || !item) return
  const listRect = list.getBoundingClientRect()
  const itemRect = item.getBoundingClientRect()
  if (itemRect.top < listRect.top) {
    list.scrollTop -= listRect.top - itemRect.top
  } else if (itemRect.bottom > listRect.bottom) {
    list.scrollTop += itemRect.bottom - listRect.bottom
  }
})

function pick(s: SkillRef) {
  if (props.multiple) {
    checked.value = checked.value.includes(s.id)
      ? checked.value.filter((id) => id !== s.id)
      : [...checked.value, s.id]
    return
  }
  emit('select', s)
}

function confirm() {
  const picked = props.skills.filter((s) => checked.value.includes(s.id))
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

function scopeLabel(s: SkillRef) {
  if (s.source === 'builtin') return '内置'
  if (s.source === 'claude') return 'Claude Code'
  if (s.source === 'project') return '项目'
  if (s.source === 'pkg') return '智能体包'
  return '本地'
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
  <span ref="anchorRef" class="skill-picker-anchor" aria-hidden="true" />
  <Teleport to="body">
    <div
      v-if="open"
      ref="panelRef"
      class="skill-picker"
      :style="panelStyle"
      @keydown="onKeydown"
    >
      <input
        ref="searchRef"
        v-model="query"
        type="search"
        class="search"
        placeholder="搜索技能…"
        autocomplete="off"
      />
      <ul ref="listRef">
        <li
          v-for="(s, i) in filtered"
          :key="s.id"
          :class="{ on: i === activeIndex, picked: multiple && checked.includes(s.id) }"
          @mousedown.prevent="pick(s)"
          @mouseenter="activeIndex = i"
        >
          <div class="row">
            <strong>/{{ s.name }}</strong>
            <em>{{ scopeLabel(s) }}</em>
          </div>
        </li>
        <li v-if="!filtered.length" class="empty">无匹配技能</li>
      </ul>
      <div v-if="multiple" class="foot">
        <button type="button" @click="emit('close')">取消</button>
        <button type="button" class="ok" :disabled="!checked.length" @click="confirm">确认</button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.skill-picker-anchor {
  display: none;
}

.skill-picker {
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

.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.row strong {
  font-size: 13px;
  font-weight: 560;
}

.row em {
  font-style: normal;
  font-size: 11px;
  color: var(--text-dim);
  padding: 1px 6px;
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  flex-shrink: 0;
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
