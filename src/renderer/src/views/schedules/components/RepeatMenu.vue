<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { onClickOutside } from '@vueuse/core';
import { repeatOptions, type ScheduleRepeat, type ScheduleRule } from '@shared/schedule';
import { useAnchoredPop } from '@/components/ui/useAnchoredPop';

const props = defineProps<{
  repeat: ScheduleRepeat;
  weekdays: number[];
  monthDays: number[];
  month: number;
  day: number;
}>();

const emit = defineEmits<{
  patch: [value: Partial<ScheduleRule>];
}>();

const weekdayItems = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
  { value: 7, label: '周日' }
];

const childBranches = new Set<ScheduleRepeat>(['weekly', 'biweekly', 'monthly', 'yearly']);

const open = ref(false);
const panel = ref<Extract<ScheduleRepeat, 'weekly' | 'biweekly' | 'monthly' | 'yearly'> | null>(
  null
);
const triggerRef = ref<HTMLElement | null>(null);
const popRef = ref<HTMLElement | null>(null);
const monthListRef = ref<HTMLElement | null>(null);
const monthWheelRef = ref<HTMLElement | null>(null);
const dayWheelRef = ref<HTMLElement | null>(null);
const { style: popStyle, prepare } = useAnchoredPop(open, triggerRef, popRef, () => false, {
  maxHeight: 420,
  maxWidth: 480
});

onClickOutside(
  triggerRef,
  () => {
    open.value = false;
  },
  { ignore: [popRef] }
);

const summary = computed(() => repeatSummary());
const yearDays = computed(() =>
  Array.from({ length: daysInMonth(props.month) }, (_, index) => index + 1)
);
const monthNumbers = Array.from({ length: 12 }, (_, index) => index + 1);

watch(open, (value) => {
  if (!value) panel.value = null;
});

watch(panel, async (value) => {
  await nextTick();
  if (value === 'yearly') {
    const max = daysInMonth(props.month);
    if (props.day > max) emit('patch', { day: max });
    await nextTick();
    revealYear();
  }
  if (value === 'monthly') revealMonth();
});

watch(
  () => [props.month, props.day] as const,
  async () => {
    if (!open.value || panel.value !== 'yearly') return;
    await nextTick();
    revealYear();
  }
);

function toggle(): void {
  if (open.value) {
    open.value = false;
    return;
  }
  panel.value = null;
  prepare();
  open.value = true;
}

function hasChildren(value: ScheduleRepeat): boolean {
  return childBranches.has(value);
}

function choose(value: ScheduleRepeat): void {
  if (value !== 'weekly' && value !== 'biweekly' && value !== 'monthly' && value !== 'yearly') {
    if (props.repeat !== value) emit('patch', { repeat: value });
    open.value = false;
    return;
  }
  if (props.repeat !== value) {
    if (value === 'weekly' || value === 'biweekly') {
      emit('patch', { repeat: value, weekdays: [todayWeekday()] });
    } else if (value === 'monthly') {
      emit('patch', { repeat: value, monthDays: [todayMonthDay()] });
    } else {
      const now = new Date();
      emit('patch', { repeat: 'yearly', month: now.getMonth() + 1, day: now.getDate() });
    }
  }
  panel.value = value;
}

function toggleWeekday(day: number): void {
  const current = normalize(props.weekdays, 1, 7);
  const next = current.includes(day)
    ? current.filter((item) => item !== day)
    : [...current, day].sort((a, b) => a - b);
  if (next.length === 0) return;
  emit('patch', { weekdays: next });
}

function toggleMonthDay(day: number): void {
  const current = normalize(props.monthDays, 1, 31);
  const next = current.includes(day)
    ? current.filter((item) => item !== day)
    : [...current, day].sort((a, b) => a - b);
  if (next.length === 0) return;
  emit('patch', { monthDays: next });
}

function selectYearMonth(month: number): void {
  emit('patch', { month, day: Math.min(props.day, daysInMonth(month)) });
}

function selectYearDay(day: number): void {
  emit('patch', { day });
}

function weekdayOn(day: number): boolean {
  return props.weekdays.includes(day);
}

function monthDayOn(day: number): boolean {
  return props.monthDays.includes(day);
}

function revealMonth(): void {
  const list = monthListRef.value;
  const item = list?.querySelector<HTMLElement>('[data-selected="true"]');
  if (!list || !item) return;
  const next = item.offsetTop - list.clientHeight / 2 + item.offsetHeight / 2;
  list.scrollTop = Math.max(0, next);
}

function revealYear(): void {
  revealCenter(monthWheelRef.value, props.month);
  revealCenter(dayWheelRef.value, Math.min(props.day, daysInMonth(props.month)));
}

function revealCenter(container: HTMLElement | null, value: number): void {
  if (!container) return;
  const item = container.querySelector<HTMLElement>(`[data-value="${value}"]`);
  if (!item) return;
  const next = item.offsetTop - container.clientHeight / 2 + item.offsetHeight / 2;
  container.scrollTop = Math.max(0, next);
}

function repeatSummary(): string {
  if (props.repeat === 'interval') return '间隔';
  if (props.repeat === 'once') return '单次';
  if (props.repeat === 'daily') return '每天';
  if (props.repeat === 'weekly' || props.repeat === 'biweekly') {
    const days = normalize(props.weekdays, 1, 7);
    const prefix = props.repeat === 'weekly' ? '每周' : '每双周';
    if (days.length === 0) return prefix;
    if (sameDays(days, [1, 2, 3, 4, 5, 6, 7])) return props.repeat === 'weekly' ? '每天' : prefix;
    if (sameDays(days, [1, 2, 3, 4, 5]))
      return props.repeat === 'weekly' ? '工作日' : `${prefix} 工作日`;
    if (sameDays(days, [6, 7])) return props.repeat === 'weekly' ? '周末' : `${prefix} 周末`;
    const named = days.map((day) => weekdayItems[day - 1]?.label).filter(Boolean);
    return `${prefix} ${named.join('、')}`;
  }
  if (props.repeat === 'monthly') {
    const days = normalize(props.monthDays, 1, 31);
    if (days.length === 0) return '每月';
    if (days.length === 31) return '每天';
    return `每月 ${days.map((day) => `${day}日`).join('、')}`;
  }
  return `每年 ${props.month}月${props.day}日`;
}

function sameDays(days: number[], expected: number[]): boolean {
  return days.length === expected.length && expected.every((day) => days.includes(day));
}

function normalize(days: number[], min: number, max: number): number[] {
  return [...new Set(days.filter((day) => day >= min && day <= max))].sort((a, b) => a - b);
}

function todayWeekday(): number {
  const day = new Date().getDay();
  return day === 0 ? 7 : day;
}

function todayMonthDay(): number {
  return new Date().getDate();
}

function daysInMonth(month: number): number {
  const safe = Math.min(12, Math.max(1, month));
  return new Date(new Date().getFullYear(), safe, 0).getDate();
}
</script>

<template>
  <div ref="triggerRef" class="repeat" :class="{ open }">
    <button
      type="button"
      class="trigger"
      :title="summary"
      aria-haspopup="menu"
      :aria-expanded="open"
      @click="toggle"
    >
      <span class="label">{{ summary }}</span>
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
    <Teleport to="body">
      <div v-if="open" ref="popRef" class="pop" :style="popStyle">
        <div class="card ec-stage">
          <ul class="col" role="menu" aria-label="重复">
            <li v-for="option in repeatOptions" :key="option.value">
              <button
                type="button"
                class="opt"
                role="menuitem"
                :class="{
                  on: option.value === repeat,
                  active: panel === option.value
                }"
                :aria-haspopup="hasChildren(option.value) ? 'menu' : undefined"
                :aria-expanded="hasChildren(option.value) ? panel === option.value : undefined"
                @click="choose(option.value)"
              >
                <span class="opt-label">{{ option.label }}</span>
                <svg
                  v-if="hasChildren(option.value)"
                  class="arrow"
                  viewBox="0 0 16 16"
                  aria-hidden="true"
                >
                  <path
                    d="M6 4 10 8 6 12"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.6"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
                <svg
                  v-else-if="option.value === repeat"
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
            </li>
          </ul>
        </div>

        <div v-if="panel" class="side-anchor" :class="{ year: panel === 'yearly' }">
          <div
            v-if="panel === 'weekly' || panel === 'biweekly'"
            class="card side ec-stage"
            role="menu"
            :aria-label="panel === 'weekly' ? '每周' : '每双周'"
          >
            <ul class="col">
              <li v-for="item in weekdayItems" :key="item.value">
                <button
                  type="button"
                  class="opt"
                  role="menuitemcheckbox"
                  :aria-checked="weekdayOn(item.value)"
                  @click="toggleWeekday(item.value)"
                >
                  <span class="opt-label">{{ item.label }}</span>
                  <svg
                    v-if="weekdayOn(item.value)"
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
              </li>
            </ul>
          </div>

          <div
            v-else-if="panel === 'monthly'"
            class="card side ec-stage"
            role="menu"
            aria-label="每月"
          >
            <ul ref="monthListRef" class="col scroll">
              <li v-for="dayNumber in 31" :key="dayNumber">
                <button
                  type="button"
                  class="opt"
                  role="menuitemcheckbox"
                  :data-selected="monthDayOn(dayNumber) ? 'true' : undefined"
                  :aria-checked="monthDayOn(dayNumber)"
                  @click="toggleMonthDay(dayNumber)"
                >
                  <span class="opt-label">{{ dayNumber }}日</span>
                  <svg
                    v-if="monthDayOn(dayNumber)"
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
              </li>
            </ul>
          </div>

          <div v-else-if="panel === 'yearly'" class="card side year ec-stage" aria-label="每年">
            <div ref="monthWheelRef" class="wheel" role="listbox" aria-label="月份">
              <button
                v-for="monthNumber in monthNumbers"
                :key="monthNumber"
                type="button"
                class="cell"
                role="option"
                :data-value="monthNumber"
                :class="{ on: monthNumber === month }"
                :aria-selected="monthNumber === month"
                @click="selectYearMonth(monthNumber)"
              >
                {{ monthNumber }}月
              </button>
            </div>
            <div ref="dayWheelRef" class="wheel day" role="listbox" aria-label="日期">
              <button
                v-for="dayNumber in yearDays"
                :key="dayNumber"
                type="button"
                class="cell"
                role="option"
                :data-value="dayNumber"
                :class="{ on: dayNumber === day }"
                :aria-selected="dayNumber === day"
                @click="selectYearDay(dayNumber)"
              >
                {{ dayNumber }}日
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.repeat {
  position: relative;
  width: 100%;
  min-width: 0;
}

.trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  box-sizing: border-box;
  width: 100%;
  min-height: 40px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  background-image: none;
  color: var(--text);
  font: inherit;
  font-size: var(--ec-text-body);
  text-align: left;
  cursor: pointer;
}

.trigger:hover,
.repeat.open .trigger {
  border-color: var(--border-strong);
  background: var(--surface);
  background-image: none;
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

.chevron {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  color: var(--text-muted);
  transition: transform 0.16s ease;
}

.repeat.open .chevron {
  transform: rotate(180deg);
}

.pop {
  display: flex;
  align-items: stretch;
  gap: 4px;
  overflow: visible;
}

.side-anchor {
  position: relative;
  flex: none;
  width: 140px;
}

.side-anchor.year {
  width: 226px;
}

.card {
  box-sizing: border-box;
  flex: none;
  width: 140px;
  padding: 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
}

.card.side {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  width: auto;
  overflow: hidden;
}

.card.side.year {
  flex-direction: row;
  gap: 4px;
}

.col {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.card.side .col,
.wheel {
  flex: 1 1 auto;
  min-height: 0;
  scrollbar-width: thin;
  overscroll-behavior: contain;
}

.col.scroll {
  position: relative;
  overflow-y: auto;
}

.opt,
.cell {
  display: flex;
  align-items: center;
  gap: 8px;
  box-sizing: border-box;
  width: 100%;
  min-height: 32px;
  padding: 5px 8px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  background-image: none;
  color: var(--text);
  font: inherit;
  font-size: var(--ec-text-body);
  line-height: 22px;
  text-align: left;
  cursor: pointer;
  user-select: none;
}

.opt:hover,
.cell:hover {
  background: var(--surface-hover);
  background-image: none;
}

.opt.active {
  background: var(--surface);
  background-image: none;
}

.opt-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.arrow,
.check {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.arrow {
  color: var(--text-muted);
  opacity: 0.55;
  transition: opacity 0.12s ease;
}

.opt:hover .arrow,
.opt.on .arrow,
.opt.active .arrow {
  opacity: 1;
}

.check {
  color: var(--accent);
}

.wheel {
  position: relative;
  width: auto;
  overflow-y: auto;
}

.wheel.day {
  flex: 0.82 1 auto;
}

.cell.on {
  background: var(--surface);
  background-image: none;
  font-weight: 500;
}

.cell.on:hover {
  background: var(--surface-hover);
  background-image: none;
}
</style>
