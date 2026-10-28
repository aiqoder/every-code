<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { useAnchoredPop } from './useAnchoredPop'

const props = withDefaults(
  defineProps<{
    start?: string
    end?: string
    disabled?: boolean
  }>(),
  {
    start: '',
    end: '',
    disabled: false
  }
)

const emit = defineEmits<{
  'update:start': [value: string]
  'update:end': [value: string]
}>()

const weekdayLabels = ['一', '二', '三', '四', '五', '六', '日']

const open = ref(false)
const anchor = ref('')
const hover = ref('')
const viewYear = ref(new Date().getFullYear())
const viewMonth = ref(new Date().getMonth())
const triggerRef = ref<HTMLElement | null>(null)
const popRef = ref<HTMLElement | null>(null)
const { style: popStyle, prepare } = useAnchoredPop(
  open,
  triggerRef,
  popRef,
  () => true,
  { maxHeight: 420 }
)

onClickOutside(
  triggerRef,
  () => {
    open.value = false
  },
  { ignore: [popRef] }
)

const filled = computed(() => Boolean(props.start || props.end))
const fieldStart = computed(() => {
  if (!anchor.value) return props.start
  if (hover.value && hover.value < anchor.value) return hover.value
  return anchor.value
})
const fieldEnd = computed(() => {
  if (!anchor.value) return props.end
  if (!hover.value || hover.value === anchor.value) return ''
  return hover.value < anchor.value ? anchor.value : hover.value
})
const hint = computed(() => {
  if (anchor.value) return '选择结束日期'
  if (props.start || props.end) return '点击日期重新选择'
  return '选择开始日期'
})
const rangeLabel = computed(() => {
  if (fieldStart.value && fieldEnd.value) return `有效期 ${fieldStart.value} 至 ${fieldEnd.value}`
  if (fieldStart.value) return `有效期 从 ${fieldStart.value} 起`
  if (fieldEnd.value) return `有效期 至 ${fieldEnd.value}`
  return '有效期'
})

const cells = computed(() => monthCells(viewYear.value, viewMonth.value))
const span = computed(() => {
  const from = anchor.value || props.start
  const to = anchor.value ? hover.value || anchor.value : props.end
  return rangeOf(from, to)
})
const today = formatYmd(new Date())

watch(open, (value) => {
  if (value) return
  anchor.value = ''
  hover.value = ''
})

function toggle(): void {
  if (props.disabled) return
  if (open.value) {
    open.value = false
    return
  }
  anchor.value = ''
  hover.value = ''
  const focus = parseYmd(props.start) || parseYmd(props.end) || new Date()
  viewYear.value = focus.getFullYear()
  viewMonth.value = focus.getMonth()
  prepare()
  open.value = true
}

function shiftMonth(delta: number): void {
  const next = new Date(viewYear.value, viewMonth.value + delta, 1)
  viewYear.value = next.getFullYear()
  viewMonth.value = next.getMonth()
}

function onHover(value: string): void {
  if (!anchor.value) return
  hover.value = value
}

function onPick(value: string): void {
  if (!anchor.value) {
    anchor.value = value
    hover.value = value
    return
  }
  const from = anchor.value
  if (value < from) {
    emit('update:start', value)
    emit('update:end', from)
  } else {
    emit('update:start', from)
    emit('update:end', value)
  }
  anchor.value = ''
  hover.value = ''
  open.value = false
}

function clear(): void {
  anchor.value = ''
  hover.value = ''
  emit('update:start', '')
  emit('update:end', '')
}

function dayLabel(value: string): string {
  const date = parseYmd(value)
  if (!date) return value
  return `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日`
}

function monthCells(year: number, month: number): { ymd: string; day: number; outside: boolean }[] {
  const first = new Date(year, month, 1)
  const offset = (first.getDay() + 6) % 7
  const cursor = new Date(year, month, 1 - offset)
  return Array.from({ length: 42 }, () => {
    const ymd = formatYmd(cursor)
    const cell = {
      ymd,
      day: cursor.getDate(),
      outside: cursor.getMonth() !== month
    }
    cursor.setDate(cursor.getDate() + 1)
    return cell
  })
}

function rangeOf(from: string, to: string): { lo: string; hi: string } | null {
  if (!from && !to) return null
  if (from && !to) return { lo: from, hi: from }
  if (!from && to) return { lo: to, hi: to }
  return from <= to ? { lo: from, hi: to } : { lo: to, hi: from }
}

function parseYmd(value: string): Date | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  if (!match) return null
  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  const date = new Date(year, month - 1, day)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return null
  }
  return date
}

function formatYmd(date: Date): string {
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}
</script>

<template>
  <div ref="triggerRef" class="range" :class="{ open, disabled }">
    <button
      type="button"
      class="trigger"
      :disabled="disabled"
      aria-haspopup="dialog"
      :aria-expanded="open"
      :aria-label="rangeLabel"
      @click="toggle"
    >
      <span class="parts">
        <span class="part" :class="{ placeholder: !fieldStart }">{{ fieldStart || '开始日期' }}</span>
        <span class="sep">至</span>
        <span class="part" :class="{ placeholder: !fieldEnd }">{{ fieldEnd || '结束日期' }}</span>
      </span>
      <svg class="cal" viewBox="0 0 16 16" aria-hidden="true">
        <rect
          x="2.25"
          y="3.25"
          width="11.5"
          height="10.5"
          rx="1.5"
          fill="none"
          stroke="currentColor"
          stroke-width="1.4"
        />
        <path
          d="M2.25 6.5h11.5M5.5 2v2.5M10.5 2v2.5"
          fill="none"
          stroke="currentColor"
          stroke-width="1.4"
          stroke-linecap="round"
        />
      </svg>
    </button>
    <button
      v-if="filled && !disabled"
      type="button"
      class="clear"
      aria-label="清除有效期"
      @click="clear"
    >
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
      <div
        v-if="open"
        ref="popRef"
        class="pop ec-stage"
        role="dialog"
        aria-label="选择有效期"
        tabindex="-1"
        :style="popStyle"
      >
        <div class="head">
          <button type="button" class="nav" aria-label="上个月" @click="shiftMonth(-1)">
            <svg viewBox="0 0 16 16" aria-hidden="true">
              <path
                d="M10 3.5 5.5 8 10 12.5"
                fill="none"
                stroke="currentColor"
                stroke-width="1.6"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </button>
          <div class="title">{{ viewYear }}年{{ viewMonth + 1 }}月</div>
          <button type="button" class="nav" aria-label="下个月" @click="shiftMonth(1)">
            <svg viewBox="0 0 16 16" aria-hidden="true">
              <path
                d="M6 3.5 10.5 8 6 12.5"
                fill="none"
                stroke="currentColor"
                stroke-width="1.6"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </button>
        </div>
        <div class="weekdays" aria-hidden="true">
          <span v-for="label in weekdayLabels" :key="label">{{ label }}</span>
        </div>
        <div class="days" @mouseleave="hover = ''">
          <button
            v-for="cell in cells"
            :key="cell.ymd"
            type="button"
            class="day"
            :class="{
              outside: cell.outside,
              today: cell.ymd === today,
              start: span?.lo === cell.ymd,
              end: span?.hi === cell.ymd,
              in: span ? cell.ymd > span.lo && cell.ymd < span.hi : false
            }"
            :aria-label="dayLabel(cell.ymd)"
            :aria-pressed="span ? cell.ymd === span.lo || cell.ymd === span.hi : false"
            @mouseenter="onHover(cell.ymd)"
            @click="onPick(cell.ymd)"
          >
            <span class="num">{{ cell.day }}</span>
          </button>
        </div>
        <div class="foot">
          <span class="hint">{{ hint }}</span>
          <button type="button" class="text" :disabled="!filled" @click="clear">清除</button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.range {
  position: relative;
  width: 100%;
  min-width: 0;
}

.trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  box-sizing: border-box;
  width: 100%;
  min-height: 40px;
  padding: 10px 34px 10px 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  background-image: none;
  color: var(--text);
  font: inherit;
  font-size: var(--ec-text-body);
  line-height: 1.4;
  text-align: left;
  cursor: pointer;
}

.range.filled .trigger {
  padding-right: 56px;
}

.trigger:hover,
.range.open .trigger {
  background: var(--surface);
  background-image: none;
}

.trigger:hover {
  border-color: var(--border-strong);
}

.range.open .trigger,
.trigger:focus-visible {
  border-color: var(--accent);
  outline: none;
}

.range.disabled .trigger {
  opacity: 0.5;
  cursor: not-allowed;
}

.parts {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.part {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.part.placeholder,
.sep {
  color: var(--text-dim);
}

.sep {
  flex-shrink: 0;
  font-size: var(--ec-text-meta);
}

.cal {
  position: absolute;
  top: 50%;
  right: 12px;
  width: 15px;
  height: 15px;
  transform: translateY(-50%);
  color: var(--text-muted);
  pointer-events: none;
}

.clear {
  position: absolute;
  top: 50%;
  right: 32px;
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: transparent;
  background-image: none;
  color: var(--text-muted);
  transform: translateY(-50%);
  cursor: pointer;
}

.clear svg {
  width: 12px;
  height: 12px;
}

.clear:hover {
  background: var(--surface-hover);
  background-image: none;
  color: var(--text);
}

.pop {
  box-sizing: border-box;
  overflow: auto;
  padding: 10px 10px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
  outline: none;
}

.head {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 6px;
}

.title {
  flex: 1;
  min-width: 0;
  color: var(--text);
  font-size: var(--ec-text-body);
  font-weight: 600;
  text-align: center;
}

.nav,
.text,
.day {
  border: 0;
  background: transparent;
  background-image: none;
  color: var(--text);
  font: inherit;
  cursor: pointer;
}

.nav {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
}

.nav svg {
  width: 16px;
  height: 16px;
}

.nav:hover,
.text:hover:not(:disabled) {
  background: var(--surface-hover);
  background-image: none;
  color: var(--text);
}

.weekdays,
.days {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
}

.weekdays {
  margin-bottom: 2px;
  color: var(--text-dim);
  font-size: var(--ec-text-meta);
  text-align: center;
}

.weekdays span,
.day {
  height: 34px;
}

.weekdays span {
  display: grid;
  place-items: center;
}

.day {
  position: relative;
  display: grid;
  place-items: center;
  padding: 0;
  font-size: var(--ec-text-body);
}

.day::before {
  content: '';
  position: absolute;
  top: 3px;
  bottom: 3px;
  left: 0;
  right: 0;
  background: transparent;
}

.day.in::before,
.day.start::before,
.day.end::before {
  background: var(--accent-soft);
}

.day.start:not(.end)::before {
  left: 50%;
}

.day.end:not(.start)::before {
  right: 50%;
}

.day.start.end::before {
  background: transparent;
}

.day:hover {
  background: transparent;
  background-image: none;
}

.num {
  position: relative;
  z-index: 1;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
}

.day.outside {
  color: var(--text-dim);
}

.day.today:not(.start):not(.end) .num {
  box-shadow: inset 0 0 0 1px var(--accent);
}

.day.start .num,
.day.end .num {
  background: var(--accent);
  color: var(--ec-on-accent);
}

.day:hover .num {
  background: var(--surface-hover);
}

.day.start:hover .num,
.day.end:hover .num {
  background: var(--accent-hover);
  color: var(--ec-on-accent);
}

.foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 4px;
  padding-top: 6px;
  border-top: 1px solid var(--border);
}

.hint {
  color: var(--text-dim);
  font-size: var(--ec-text-meta);
}

.text {
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  font-size: var(--ec-text-meta);
}

.text:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
</style>
