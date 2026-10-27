<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NButton, NModal, NSelect, useDialog, type SelectOption } from 'naive-ui'
import type { AgentItem, ModelItem } from '@/api/catalog'
import {
  prepareDraft,
  repeatOptions,
  toLocalInput,
  type ScheduleDraft,
  type SchedulePermission,
  type ScheduleRepeat
} from '@shared/schedule'

const props = defineProps<{
  show: boolean
  saving: boolean
  draft: ScheduleDraft | null
  agents: AgentItem[]
  models: ModelItem[]
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  save: [draft: ScheduleDraft]
}>()

const dialog = useDialog()
const form = ref<ScheduleDraft | null>(null)
const errorText = ref('')
const intervalAmount = ref(1)
const intervalUnit = ref<'minute' | 'hour'>('hour')

const weekdays = [
  { value: 1, label: '一' },
  { value: 2, label: '二' },
  { value: 3, label: '三' },
  { value: 4, label: '四' },
  { value: 5, label: '五' },
  { value: 6, label: '六' },
  { value: 7, label: '日' }
]

const agentOptions = computed<SelectOption[]>(() =>
  props.agents
    .filter((item) => item.agentId)
    .map((item) => ({ label: item.name, value: item.agentId }))
)

const modelOptions = computed<SelectOption[]>(() => [
  { label: '运行时使用当前模型', value: 0 },
  ...props.models.map((item) => ({
    label: item.providerName ? `${item.providerName} / ${item.name}` : item.name,
    value: item.id
  }))
])

const title = computed(() => (form.value?.id ? '编辑定时任务' : '新建定时任务'))

const canSave = computed(
  () =>
    Boolean(form.value?.name.trim() && form.value.prompt.trim() && form.value.agentId) &&
    !props.saving
)

watch(
  () => props.show,
  (open) => {
    if (!open || !props.draft) return
    form.value = structuredClone(props.draft)
    readInterval(form.value.rule.intervalMinutes)
    errorText.value = ''
  }
)

function close(): void {
  if (props.saving) return
  emit('update:show', false)
}

function readInterval(minutes: number): void {
  if (minutes >= 60 && minutes % 60 === 0) {
    intervalAmount.value = minutes / 60
    intervalUnit.value = 'hour'
    return
  }
  intervalAmount.value = Math.max(5, minutes)
  intervalUnit.value = 'minute'
}

function writeInterval(): void {
  if (!form.value) return
  const amount = Math.max(1, Math.round(Number(intervalAmount.value) || 1))
  if (intervalUnit.value === 'hour') {
    intervalAmount.value = Math.min(168, amount)
    form.value.rule.intervalMinutes = intervalAmount.value * 60
    return
  }
  intervalAmount.value = Math.min(24 * 60, Math.max(5, amount))
  form.value.rule.intervalMinutes = intervalAmount.value
}

function onRepeat(event: Event): void {
  if (!form.value) return
  const value = (event.target as HTMLSelectElement).value as ScheduleRepeat
  form.value.rule.repeat = value
  if (value === 'once' && !form.value.rule.onceAt) {
    const next = new Date()
    next.setDate(next.getDate() + 1)
    next.setHours(9, 0, 0, 0)
    form.value.rule.onceAt = next.toISOString()
  }
}

function onOnce(event: Event): void {
  if (!form.value) return
  const value = (event.target as HTMLInputElement).value
  form.value.rule.onceAt = value ? new Date(value).toISOString() : ''
}

function toggleWeekday(day: number): void {
  if (!form.value) return
  const days = new Set(form.value.rule.weekdays)
  if (days.has(day)) days.delete(day)
  else days.add(day)
  form.value.rule.weekdays = [...days].sort((a, b) => a - b)
}

function toggleMonthDay(day: number): void {
  if (!form.value) return
  const days = new Set(form.value.rule.monthDays)
  if (days.has(day)) days.delete(day)
  else days.add(day)
  form.value.rule.monthDays = [...days].sort((a, b) => a - b)
}

function onAgent(value: string | number | Array<string | number> | null): void {
  if (!form.value || typeof value !== 'string') return
  form.value.agentId = value
  form.value.agentName = props.agents.find((item) => item.agentId === value)?.name || ''
}

function onModel(value: string | number | Array<string | number> | null): void {
  if (!form.value) return
  const parsed = typeof value === 'number' ? value : Number(value)
  form.value.modelId = Number.isInteger(parsed) && parsed > 0 ? parsed : null
}

function pickPermission(value: SchedulePermission): void {
  if (!form.value || form.value.permission === value) return
  if (value !== 'full') {
    form.value.permission = value
    return
  }
  dialog.warning({
    title: '开启完全访问权限',
    content: '任务会在无人值守时执行，工具调用不再逐次确认。',
    positiveText: '开启',
    negativeText: '取消',
    onPositiveClick: () => {
      if (form.value) form.value.permission = 'full'
    }
  })
}

async function pickWorkspace(): Promise<void> {
  if (!form.value || !window.api?.schedule) return
  const picked = await window.api.schedule.pickDirectory()
  if (picked) form.value.workspace = picked
}

function submit(): void {
  if (!form.value || !canSave.value) return
  writeInterval()
  try {
    emit('save', prepareDraft(structuredClone(form.value)))
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '无法保存'
  }
}
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="title"
    :bordered="false"
    :mask-closable="!saving"
    :close-on-esc="!saving"
    :style="{ width: '36rem', maxWidth: 'calc(100% - 2rem)' }"
    @update:show="emit('update:show', $event)"
  >
    <form
      v-if="form"
      class="flex max-h-[70vh] flex-col gap-4 overflow-auto pr-1"
      @submit.prevent="submit"
    >
      <label class="block">
        <span class="text-xs text-[var(--ec-fg-muted)]">名称</span>
        <input
          v-model="form.name"
          maxlength="80"
          class="field mt-1"
          placeholder="用来区分不同的定时任务"
        />
      </label>

      <div>
        <span class="text-xs text-[var(--ec-fg-muted)]">提示词</span>
        <div class="mt-1 overflow-hidden rounded-md border border-[var(--ec-border)]">
          <textarea
            v-model="form.prompt"
            rows="6"
            maxlength="8000"
            class="block w-full resize-y bg-transparent px-3 py-2 text-sm outline-none"
            placeholder="描述任务目标和输出要求"
          />
          <div class="flex flex-wrap gap-2 border-t border-[var(--ec-border)] px-2 py-2">
            <div class="min-w-40 flex-1">
              <n-select
                size="small"
                :value="form.agentId || null"
                :options="agentOptions"
                placeholder="选择智能体"
                @update:value="onAgent"
              />
            </div>
            <div class="min-w-44 flex-1">
              <n-select
                size="small"
                :value="form.modelId || 0"
                :options="modelOptions"
                placeholder="模型"
                @update:value="onModel"
              />
            </div>
          </div>
        </div>
        <p v-if="agentOptions.length === 0" class="mt-1 text-xs text-[var(--ec-attention)]">
          还没有可选的智能体。先在智能体页安装并同步。
        </p>
      </div>

      <div>
        <span class="text-xs text-[var(--ec-fg-muted)]">工作空间</span>
        <div class="mt-1 flex gap-2">
          <input
            :value="form.workspace"
            readonly
            class="field min-w-0 flex-1"
            placeholder="未选择时使用当前工作目录"
          />
          <button type="button" class="ghost" @click="pickWorkspace">选择</button>
          <button v-if="form.workspace" type="button" class="ghost" @click="form.workspace = ''">
            清除
          </button>
        </div>
      </div>

      <div>
        <span class="text-xs text-[var(--ec-fg-muted)]">权限</span>
        <div class="mt-1 grid grid-cols-2 gap-2">
          <button
            type="button"
            class="choice"
            :class="form.permission === 'default' ? 'choice-on' : ''"
            @click="pickPermission('default')"
          >
            <span class="block text-sm">默认权限</span>
            <span class="mt-1 block text-xs text-[var(--ec-fg-muted)]"
              >到点后打开对话，等你确认再发送</span
            >
          </button>
          <button
            type="button"
            class="choice"
            :class="form.permission === 'full' ? 'choice-on' : ''"
            @click="pickPermission('full')"
          >
            <span class="block text-sm">完全访问权限（推荐）</span>
            <span class="mt-1 block text-xs text-[var(--ec-fg-muted)]"
              >无人值守执行，工具调用不再逐次确认</span
            >
          </button>
        </div>
      </div>

      <label class="block">
        <span class="text-xs text-[var(--ec-fg-muted)]">重复</span>
        <select :value="form.rule.repeat" class="field mt-1" @change="onRepeat">
          <option v-for="item in repeatOptions" :key="item.value" :value="item.value">
            {{ item.label }}
          </option>
        </select>
      </label>

      <div v-if="form.rule.repeat === 'interval'">
        <span class="text-xs text-[var(--ec-fg-muted)]">间隔</span>
        <div class="mt-1 flex items-center gap-2">
          <span class="text-sm">每</span>
          <input
            v-model.number="intervalAmount"
            type="number"
            min="1"
            class="field w-24"
            @change="writeInterval"
          />
          <select v-model="intervalUnit" class="field w-24" @change="writeInterval">
            <option value="hour">小时</option>
            <option value="minute">分钟</option>
          </select>
        </div>
        <p class="mt-1 text-xs text-[var(--ec-fg-muted)]">保存后从现在起算，最短 5 分钟。</p>
      </div>

      <label v-else-if="form.rule.repeat === 'once'" class="block">
        <span class="text-xs text-[var(--ec-fg-muted)]">时间</span>
        <input
          :value="toLocalInput(form.rule.onceAt)"
          type="datetime-local"
          class="field mt-1"
          @input="onOnce"
        />
      </label>

      <template v-else>
        <div v-if="form.rule.repeat === 'weekly' || form.rule.repeat === 'biweekly'">
          <span class="text-xs text-[var(--ec-fg-muted)]">星期</span>
          <div class="mt-1 flex flex-wrap gap-1">
            <button
              v-for="item in weekdays"
              :key="item.value"
              type="button"
              class="chip"
              :class="form.rule.weekdays.includes(item.value) ? 'chip-on' : ''"
              @click="toggleWeekday(item.value)"
            >
              {{ item.label }}
            </button>
          </div>
          <p v-if="form.rule.repeat === 'biweekly'" class="mt-1 text-xs text-[var(--ec-fg-muted)]">
            以开始日期所在周为第一周，之后每隔一周执行。没填开始日期时从本周算起。
          </p>
        </div>

        <div v-if="form.rule.repeat === 'monthly'">
          <span class="text-xs text-[var(--ec-fg-muted)]">日期</span>
          <div class="mt-1 grid grid-cols-7 gap-1">
            <button
              v-for="day in 31"
              :key="day"
              type="button"
              class="chip"
              :class="form.rule.monthDays.includes(day) ? 'chip-on' : ''"
              @click="toggleMonthDay(day)"
            >
              {{ day }}
            </button>
          </div>
          <p class="mt-1 text-xs text-[var(--ec-fg-muted)]">没有这一天的月份会跳过。</p>
        </div>

        <div v-if="form.rule.repeat === 'yearly'" class="flex gap-2">
          <label class="block flex-1">
            <span class="text-xs text-[var(--ec-fg-muted)]">月份</span>
            <select v-model.number="form.rule.month" class="field mt-1">
              <option v-for="month in 12" :key="month" :value="month">{{ month }} 月</option>
            </select>
          </label>
          <label class="block flex-1">
            <span class="text-xs text-[var(--ec-fg-muted)]">日期</span>
            <input
              v-model.number="form.rule.day"
              type="number"
              min="1"
              max="31"
              class="field mt-1"
            />
          </label>
        </div>

        <label class="block">
          <span class="text-xs text-[var(--ec-fg-muted)]">时间</span>
          <input v-model="form.rule.time" type="time" class="field mt-1" />
        </label>

        <div class="grid grid-cols-2 gap-2">
          <label class="block">
            <span class="text-xs text-[var(--ec-fg-muted)]">开始日期</span>
            <input v-model="form.rule.startDate" type="date" class="field mt-1" />
          </label>
          <label class="block">
            <span class="text-xs text-[var(--ec-fg-muted)]">结束日期</span>
            <input v-model="form.rule.endDate" type="date" class="field mt-1" />
          </label>
        </div>
      </template>

      <p v-if="errorText" class="text-sm text-[var(--ec-danger)]">{{ errorText }}</p>
    </form>
    <template #footer>
      <div class="flex justify-end gap-2">
        <n-button :disabled="saving" @click="close">取消</n-button>
        <n-button type="primary" :loading="saving" :disabled="!canSave" @click="submit"
          >确定</n-button
        >
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.field {
  height: 2rem;
  width: 100%;
  border-radius: 0.375rem;
  border: 1px solid var(--ec-border);
  background: var(--ec-canvas);
  padding: 0 0.5rem;
  font-size: 0.875rem;
  outline: none;
}

textarea.field,
textarea {
  height: auto;
}

.ghost {
  height: 2rem;
  flex-shrink: 0;
  border-radius: 0.375rem;
  border: 1px solid var(--ec-border);
  padding: 0 0.75rem;
  font-size: 0.875rem;
}

.ghost:hover {
  background: color-mix(in srgb, var(--ec-fg) 8%, transparent);
}

.choice {
  border-radius: 0.375rem;
  border: 1px solid var(--ec-border);
  padding: 0.625rem 0.75rem;
  text-align: left;
}

.choice-on {
  border-color: var(--ec-accent);
  background: color-mix(in srgb, var(--ec-accent) 12%, transparent);
}

.chip {
  min-width: 2rem;
  border-radius: 0.375rem;
  border: 1px solid var(--ec-border);
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  color: var(--ec-fg-muted);
}

.chip-on {
  border-color: var(--ec-accent);
  background: color-mix(in srgb, var(--ec-accent) 16%, transparent);
  color: var(--ec-accent);
}
</style>
