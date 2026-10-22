<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore, messageOf } from '../../stores/session'
import { adeApi } from '../../api'
import { toast } from '../../composables/toast'
import EmptyState from '../../components/EmptyState.vue'
import TextField from '../../components/TextField.vue'
import type { RunRecord } from '../../../../shared/types'

const session = useSessionStore()
const router = useRouter()

const busy = ref(false)
const kind = ref<string>('all')
const status = ref<string>('all')
const keyword = ref('')

const KINDS: Array<{ id: string; label: string }> = [
  { id: 'all', label: '全部' },
  { id: 'planner', label: '拆解' },
  { id: 'advisor', label: '顾问' },
  { id: 'coder', label: '开发' },
  { id: 'verifier', label: '验收' },
  { id: 'reviewer', label: '复核' },
  { id: 'installer', label: '安装' },
  { id: 'deploy', label: '发布' }
]

const STATUSES: Array<{ id: string; label: string }> = [
  { id: 'all', label: '全部状态' },
  { id: 'running', label: '进行中' },
  { id: 'done', label: '完成' },
  { id: 'failed', label: '失败' },
  { id: 'skipped', label: '跳过' }
]

const kindLabelOf: Record<string, string> = {
  planner: '拆解',
  advisor: '顾问',
  coder: '开发',
  verifier: '验收',
  reviewer: '复核',
  installer: '安装',
  deploy: '发布'
}

const statusLabelOf: Record<string, string> = {
  running: '进行中',
  done: '完成',
  failed: '失败',
  skipped: '跳过'
}

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return session.runs.filter((r) => {
    if (kind.value !== 'all' && r.kind !== kind.value) return false
    if (status.value !== 'all' && r.status !== status.value) return false
    if (!kw) return true
    return (
      String(r.id).includes(kw) ||
      String(r.taskId).includes(kw) ||
      (r.agentId ?? '').toLowerCase().includes(kw) ||
      (r.summary ?? '').toLowerCase().includes(kw) ||
      (r.error ?? '').toLowerCase().includes(kw)
    )
  })
})

const counts = computed(() => {
  const out: Record<string, number> = { running: 0, done: 0, failed: 0, skipped: 0 }
  for (const r of session.runs) out[r.status] = (out[r.status] ?? 0) + 1
  return out
})

function taskName(id: number): string {
  const task = session.allTasks.find((t) => t.id === id)
  return task?.name || (id ? `任务 #${id}` : '—')
}

function taskProject(id: number): string {
  const task = session.allTasks.find((t) => t.id === id)
  return task?.projectName || '—'
}

function tone(status: string): string {
  if (status === 'done') return 'moss'
  if (status === 'failed') return 'rust'
  if (status === 'running') return 'brass'
  return 'muted'
}

function elapsed(run: RunRecord): string {
  if (!run.startedAt) return '—'
  const start = new Date(run.startedAt.replace(' ', 'T')).getTime()
  const end = run.finishedAt ? new Date(run.finishedAt.replace(' ', 'T')).getTime() : Date.now()
  if (!Number.isFinite(start) || !Number.isFinite(end)) return '—'
  const sec = Math.max(0, Math.round((end - start) / 1000))
  if (sec < 60) return `${sec}s`
  const min = Math.floor(sec / 60)
  return `${min}m ${sec % 60}s`
}

async function refresh(): Promise<void> {
  busy.value = true
  try {
    await session.loadRuns()
    if (!session.allTasks.length) await session.loadAllTasks()
  } catch (err) {
    toast.error(messageOf(err))
  } finally {
    busy.value = false
  }
}

async function openTask(run: RunRecord): Promise<void> {
  if (!run.taskId) return
  if (!session.allTasks.some((t) => t.id === run.taskId)) {
    try {
      const task = await adeApi.getTask(run.taskId)
      session.upsertTask(task)
    } catch {
      toast.error('任务已不存在')
      return
    }
  }
  router.push(`/tasks/${run.taskId}`)
}

onMounted(async () => {
  if (!session.allTasks.length) await session.loadAllTasks()
  if (!session.runs.length) await session.loadRuns()
})
</script>

<template>
  <div class="runs">
    <header class="page-head">
      <div class="col">
        <span class="eyebrow">运行记录</span>
        <h1 class="serif">Agent 跑过的每一步</h1>
        <p class="lede">
          拆解、开发、验收、复核、发布都在这里留痕。点一条回到对应任务。
        </p>
      </div>
      <div class="spacer" />
      <div class="stats">
        <span class="stat live"><b>{{ counts.running }}</b> 进行中</span>
        <span class="stat ok"><b>{{ counts.done }}</b> 完成</span>
        <span class="stat bad"><b>{{ counts.failed }}</b> 失败</span>
      </div>
    </header>

    <div class="filters">
      <div class="group">
        <button
          v-for="k in KINDS"
          :key="k.id"
          class="chip-filter"
          :class="{ on: kind === k.id }"
          @click="kind = k.id"
        >
          {{ k.label }}
        </button>
      </div>
      <div class="spacer" />
      <TextField v-model="keyword" class="search" placeholder="搜索任务、Agent、摘要" spellcheck="false" />
      <select v-model="status" class="status-select">
        <option v-for="s in STATUSES" :key="s.id" :value="s.id">{{ s.label }}</option>
      </select>
      <button class="btn ghost sm" :disabled="busy" @click="refresh">{{ busy ? '刷新中…' : '刷新' }}</button>
    </div>

    <div class="list-wrap">
      <EmptyState
        v-if="!filtered.length"
        empty-title="没有匹配的运行记录"
        empty-hint="换个筛选条件，或先去任务页加一条任务让它跑起来。"
      />

      <div v-else class="list">
        <button v-for="run in filtered" :key="run.id" class="run-card" :class="tone(run.status)" @click="openTask(run)">
          <span class="bar" />
          <div class="rc-body">
            <div class="rc-top">
              <span class="rc-kind">{{ kindLabelOf[run.kind] ?? run.kind }}</span>
              <span class="rc-agent mono ellipsis">{{ run.agentId || '—' }}</span>
              <span class="rc-task ellipsis">{{ taskName(run.taskId) }}</span>
              <div class="spacer" />
              <span class="rc-status">{{ statusLabelOf[run.status] ?? run.status }}</span>
            </div>
            <p v-if="run.summary" class="rc-sum">{{ run.summary }}</p>
            <p v-if="run.error" class="rc-err">{{ run.error }}</p>
            <div class="rc-foot">
              <span class="mono">#{{ run.id }}</span>
              <span class="ellipsis">{{ taskProject(run.taskId) }}</span>
              <span class="spacer" />
              <span class="mono">{{ elapsed(run) }}</span>
              <span class="mono">{{ run.startedAt }}</span>
            </div>
          </div>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.runs {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.page-head {
  display: flex;
  align-items: flex-end;
  gap: 22px;
  padding: 22px 26px 14px;
  border-bottom: 1px solid var(--line-soft);
  -webkit-app-region: drag;
  flex-wrap: wrap;
}
.eyebrow {
  font-size: 10.5px;
  letter-spacing: 0.18em;
  color: var(--brass-dim);
  text-transform: uppercase;
  margin-bottom: 5px;
}
h1 {
  font-size: 21px;
}
.lede {
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-dim);
  max-width: 62em;
  line-height: 1.7;
}
.stats {
  display: flex;
  gap: 16px;
  padding-bottom: 3px;
}
.stat {
  font-size: 11.5px;
  color: var(--text-faint);
}
.stat b {
  font-family: var(--font-mono);
  color: var(--text-dim);
  font-weight: 500;
  margin-right: 3px;
}
.stat.live b {
  color: var(--brass);
}
.stat.ok b {
  color: var(--moss);
}
.stat.bad b {
  color: var(--rust);
}

.filters {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 26px;
  border-bottom: 1px solid var(--line-soft);
  background: var(--ink-850);
  flex-wrap: wrap;
}
.group {
  display: flex;
  gap: 2px;
  flex-wrap: wrap;
}
.chip-filter {
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  border: 1px solid transparent;
  font-size: 11.5px;
  color: var(--text-faint);
}
.chip-filter:hover {
  background: var(--ink-800);
  color: var(--text);
}
.chip-filter.on {
  border-color: var(--brass-dim);
  background: rgba(200, 160, 90, 0.08);
  color: var(--brass);
}
.filters .search {
  width: 220px;
  height: 26px;
  padding: 0 9px;
  font-size: 11.5px;
  border-radius: var(--r-sm);
}
.status-select {
  width: auto;
  height: 26px;
  padding: 0 8px;
  font-size: 11.5px;
}

.list-wrap {
  flex: 1;
  overflow: auto;
  padding: 14px 26px 34px;
}
.list {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.run-card {
  display: flex;
  gap: 0;
  text-align: left;
  background: var(--paper-900);
  border: 1px solid var(--line-soft);
  border-radius: var(--r-md);
  overflow: hidden;
  transition: border-color 0.13s ease, background 0.13s ease;
}
.run-card:hover {
  border-color: var(--line-strong);
  background: var(--paper-800);
}
.bar {
  width: 2px;
  flex: none;
  background: var(--line);
}
.run-card.moss .bar {
  background: var(--moss);
}
.run-card.rust .bar {
  background: var(--rust);
}
.run-card.brass .bar {
  background: var(--brass);
}
.rc-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 9px 13px;
}
.rc-top {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
}
.rc-kind {
  font-size: 10px;
  letter-spacing: 0.09em;
  text-transform: uppercase;
  color: var(--text-dim);
  border: 1px solid var(--line);
  border-radius: 3px;
  padding: 0 4px;
  flex: none;
}
.rc-agent {
  font-size: 10px;
  color: var(--text-faint);
  max-width: 120px;
}
.rc-task {
  font-size: 12px;
  color: var(--text-strong);
  max-width: 40%;
}
.rc-status {
  font-size: 11px;
  color: var(--text-faint);
  flex: none;
}
.run-card.moss .rc-status {
  color: var(--moss);
}
.run-card.rust .rc-status {
  color: #d98d7d;
}
.run-card.brass .rc-status {
  color: var(--brass);
}
.rc-sum {
  font-size: 11.5px;
  line-height: 1.6;
  color: var(--text-dim);
  white-space: pre-wrap;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.rc-err {
  font-size: 11.5px;
  line-height: 1.6;
  color: #d98d7d;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.rc-foot {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 9.5px;
  color: var(--text-faint);
  min-width: 0;
}
</style>
