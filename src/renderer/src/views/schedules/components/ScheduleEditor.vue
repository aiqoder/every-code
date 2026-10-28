<script setup lang="ts">
import { computed, ref, toRaw, watch } from 'vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiDateRange from '@/components/ui/UiDateRange.vue';
import UiDialog from '@/components/ui/UiDialog.vue';
import UiField from '@/components/ui/UiField.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';
import type { SelectOption } from '@/components/ui/types';
import type { AgentItem, ModelItem } from '@/api/catalog';
import { listConversations } from '@/api/conversations';
import { useSessionStore } from '@/stores/sessionStore';
import {
  prepareDraft,
  toLocalInput,
  type ScheduleDraft,
  type ScheduleRule
} from '@shared/schedule';
import RepeatMenu from './RepeatMenu.vue';

const addDirectory = '__add_directory__';

const props = defineProps<{
  show: boolean;
  saving: boolean;
  draft: ScheduleDraft | null;
  agents: AgentItem[];
  models: ModelItem[];
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  save: [draft: ScheduleDraft];
}>();

const session = useSessionStore();
const form = ref<ScheduleDraft | null>(null);
const errorText = ref('');
const knownDirs = ref<string[]>([]);
const intervalAmount = ref(1);
const intervalText = ref('1');
const intervalUnit = ref<'minute' | 'hour'>('hour');

const agentOptions = computed<SelectOption[]>(() =>
  props.agents
    .filter((item) => item.agentId)
    .map((item) => ({ label: item.name, value: item.agentId }))
);

const modelOptions = computed<SelectOption[]>(() => [
  { label: '运行时使用当前模型', value: 0 },
  ...props.models.map((item) => ({
    label: item.providerName ? `${item.providerName} / ${item.name}` : item.name,
    value: item.id
  }))
]);

const intervalUnitOptions: SelectOption[] = [
  { label: '小时', value: 'hour' },
  { label: '分钟', value: 'minute' }
];

const workspaceOptions = computed<SelectOption[]>(() => {
  const paths = [...knownDirs.value];
  const current = form.value?.workspace.trim() || '';
  if (current && !paths.includes(current)) paths.unshift(current);
  const bases = paths.map((path) => directoryBase(path));
  return [
    ...paths.map((path, index) => ({
      label: bases.filter((base) => base === bases[index]).length > 1 ? path : bases[index],
      title: path,
      value: path
    })),
    { label: '新增目录', value: addDirectory, divided: paths.length > 0 }
  ];
});

const title = computed(() => (form.value?.id ? '编辑定时任务' : '新建定时任务'));

function cloneDraft(draft: ScheduleDraft): ScheduleDraft {
  return structuredClone(toRaw(draft));
}

const canSave = computed(
  () =>
    Boolean(form.value?.name.trim() && form.value.prompt.trim() && form.value.agentId) &&
    !props.saving
);

watch(
  () => props.show,
  (open) => {
    if (!open || !props.draft) return;
    form.value = cloneDraft(props.draft);
    readInterval(form.value.rule.intervalMinutes);
    errorText.value = '';
    void loadDirectories();
  }
);

function close(): void {
  if (props.saving) return;
  emit('update:show', false);
}

function syncIntervalText(): void {
  intervalText.value = String(intervalAmount.value);
}

function readInterval(minutes: number): void {
  if (minutes >= 60 && minutes % 60 === 0) {
    intervalAmount.value = minutes / 60;
    intervalUnit.value = 'hour';
    syncIntervalText();
    return;
  }
  intervalAmount.value = Math.max(5, minutes);
  intervalUnit.value = 'minute';
  syncIntervalText();
}

function writeInterval(): void {
  if (!form.value) return;
  const amount = Math.max(1, Math.round(Number(intervalAmount.value) || 1));
  if (intervalUnit.value === 'hour') {
    intervalAmount.value = Math.min(168, amount);
    form.value.rule.intervalMinutes = intervalAmount.value * 60;
    syncIntervalText();
    return;
  }
  intervalAmount.value = Math.min(24 * 60, Math.max(5, amount));
  form.value.rule.intervalMinutes = intervalAmount.value;
  syncIntervalText();
}

function onRulePatch(patch: Partial<ScheduleRule>): void {
  if (!form.value) return;
  Object.assign(form.value.rule, patch);
  if (patch.repeat === 'once' && !form.value.rule.onceAt) {
    const next = new Date();
    next.setDate(next.getDate() + 1);
    next.setHours(9, 0, 0, 0);
    form.value.rule.onceAt = next.toISOString();
  }
}

function onOnce(value: string): void {
  if (!form.value) return;
  form.value.rule.onceAt = value ? new Date(value).toISOString() : '';
}

function onIntervalAmount(value: string): void {
  intervalText.value = value;
  const amount = Number(value);
  if (Number.isFinite(amount)) intervalAmount.value = amount;
}

function onIntervalUnit(value: string | number | null): void {
  if (value !== 'hour' && value !== 'minute') return;
  intervalUnit.value = value;
  writeInterval();
}

function onAgent(value: string | number | null): void {
  if (!form.value || typeof value !== 'string') return;
  form.value.agentId = value;
  form.value.agentName = props.agents.find((item) => item.agentId === value)?.name || '';
}

function onModel(value: string | number | null): void {
  if (!form.value) return;
  const parsed = typeof value === 'number' ? value : Number(value);
  form.value.modelId = Number.isInteger(parsed) && parsed > 0 ? parsed : null;
}

function directoryBase(path: string): string {
  const parts = path.split(/[/\\]/).filter((part) => part.length > 0);
  return parts[parts.length - 1] || path;
}

function rememberDirectory(path: string): void {
  const trimmed = path.trim();
  if (!trimmed || knownDirs.value.includes(trimmed)) return;
  knownDirs.value = [...knownDirs.value, trimmed];
}

async function loadDirectories(): Promise<void> {
  const found: string[] = [];
  const push = (path: string): void => {
    const trimmed = path.trim();
    if (trimmed && !found.includes(trimmed)) found.push(trimmed);
  };
  if (window.api?.loop) push((await window.api.loop.getWorkspace().catch(() => '')) || '');
  if (window.api?.schedule) {
    const state = await window.api.schedule.state().catch(() => null);
    for (const task of state?.tasks || []) push(task.workspace || '');
  }
  if (session.loggedIn) {
    const list = await listConversations(session.clientId).catch(() => []);
    for (const item of list) push(item.workspace || '');
  }
  if (form.value?.workspace) push(form.value.workspace);
  knownDirs.value = found;
}

async function pickWorkspace(): Promise<void> {
  if (!form.value || !window.api?.schedule) return;
  const picked = await window.api.schedule.pickDirectory();
  if (!picked || !form.value) return;
  rememberDirectory(picked);
  form.value.workspace = picked;
}

function onWorkspace(value: string | number | null): void {
  if (!form.value || typeof value !== 'string') return;
  if (value === addDirectory) {
    void pickWorkspace();
    return;
  }
  form.value.workspace = value;
}

function clearWorkspace(): void {
  if (!form.value) return;
  form.value.workspace = '';
}

function submit(): void {
  if (!form.value || !canSave.value) return;
  writeInterval();
  try {
    emit('save', prepareDraft(cloneDraft(form.value)));
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '无法保存';
  }
}
</script>

<template>
  <UiDialog
    :show="show"
    :title="title"
    width="44rem"
    :mask-closable="!saving"
    :close-on-esc="!saving"
    @update:show="emit('update:show', $event)"
  >
    <form v-if="form" class="flex flex-col gap-4" @submit.prevent="submit">
      <label class="block">
        <span class="text-xs text-[var(--ec-fg-muted)]">名称</span>
        <UiField
          v-model="form.name"
          class="mt-1"
          :maxlength="80"
          placeholder="用来区分不同的定时任务"
        />
      </label>

      <div>
        <span class="text-xs text-[var(--ec-fg-muted)]">提示词</span>
        <div class="mt-1 overflow-hidden rounded-md border border-[var(--ec-border)]">
          <UiTextarea
            v-model="form.prompt"
            :rows="6"
            :maxlength="8000"
            :bordered="false"
            placeholder="描述任务目标和输出要求"
          />
          <div class="flex items-center gap-2 border-t border-[var(--ec-border)] px-2 py-2">
            <div class="min-w-0 flex-1">
              <UiSelect
                size="small"
                :model-value="form.agentId || null"
                :options="agentOptions"
                placeholder="选择智能体"
                @update:model-value="onAgent"
              />
            </div>
            <div class="min-w-0 flex-1">
              <UiSelect
                size="small"
                :model-value="form.modelId || 0"
                :options="modelOptions"
                placeholder="模型"
                @update:model-value="onModel"
              />
            </div>
            <div
              class="min-w-0 flex-1"
              :title="form.workspace || '未选择时使用当前工作目录'"
            >
              <UiSelect
                size="small"
                clearable
                :model-value="form.workspace || null"
                :options="workspaceOptions"
                placeholder="工作目录"
                @update:model-value="onWorkspace"
                @clear="clearWorkspace"
              />
            </div>
          </div>
        </div>
        <p v-if="agentOptions.length === 0" class="mt-1 text-xs text-[var(--ec-attention)]">
          还没有可选的智能体。先在智能体页安装并同步。
        </p>
      </div>

      <div class="flex flex-col gap-1">
        <div class="flex items-center gap-3">
          <div class="flex min-w-0 flex-1 items-center gap-2">
            <span class="shrink-0 text-xs text-[var(--ec-fg-muted)]">重复</span>
            <RepeatMenu
              class="min-w-0 flex-1"
              :repeat="form.rule.repeat"
              :weekdays="form.rule.weekdays"
              :month-days="form.rule.monthDays"
              :month="form.rule.month"
              :day="form.rule.day"
              @patch="onRulePatch"
            />
          </div>
          <div
            v-if="form.rule.repeat !== 'once'"
            class="flex min-w-0 flex-[1.4] items-center gap-2"
          >
            <span class="shrink-0 text-xs text-[var(--ec-fg-muted)]">有效期</span>
            <UiDateRange
              v-model:start="form.rule.startDate"
              v-model:end="form.rule.endDate"
              class="min-w-0 flex-1"
            />
          </div>
          <div v-if="form.rule.repeat === 'interval'" class="flex shrink-0 items-center gap-2">
            <span class="shrink-0 text-xs text-[var(--ec-fg-muted)]">间隔</span>
            <span class="text-sm">每</span>
            <div class="w-16">
              <UiField
                type="number"
                :min="1"
                :model-value="intervalText"
                @update:model-value="onIntervalAmount"
                @change="writeInterval"
              />
            </div>
            <div class="w-24">
              <UiSelect
                :model-value="intervalUnit"
                :options="intervalUnitOptions"
                @update:model-value="onIntervalUnit"
              />
            </div>
          </div>
          <label v-else class="flex shrink-0 items-center gap-2">
            <span class="shrink-0 text-xs text-[var(--ec-fg-muted)]">时间</span>
            <UiField
              v-if="form.rule.repeat === 'once'"
              class="w-48"
              type="datetime-local"
              :model-value="toLocalInput(form.rule.onceAt)"
              @update:model-value="onOnce"
            />
            <UiField v-else v-model="form.rule.time" class="w-32" type="time" />
          </label>
        </div>
        <p v-if="form.rule.repeat === 'interval'" class="text-xs text-[var(--ec-fg-muted)]">
          保存后从现在起算，最短 5 分钟。
        </p>
        <p v-if="form.rule.repeat === 'biweekly'" class="text-xs text-[var(--ec-fg-muted)]">
          以开始日期所在周为第一周，之后每隔一周执行。没填开始日期时从本周算起。
        </p>
        <p v-if="form.rule.repeat === 'monthly'" class="text-xs text-[var(--ec-fg-muted)]">
          没有这一天的月份会跳过。
        </p>
      </div>

      <p v-if="errorText" class="text-sm text-[var(--ec-danger)]">{{ errorText }}</p>
    </form>
    <template #footer>
      <UiButton :disabled="saving" @click="close">取消</UiButton>
      <UiButton type="primary" :loading="saving" :disabled="!canSave" @click="submit"
        >确定</UiButton
      >
    </template>
  </UiDialog>
</template>
