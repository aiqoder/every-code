<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useClientPort } from '../../ports/clientPort'
import type { TokenUsageRange, TokenUsageStats, TokenUsageTrendPoint } from '../../types'

const emptyStats = (): TokenUsageStats => ({
  promptTokens: 0,
  completionTokens: 0,
  cacheCreationTokens: 0,
  cacheReadTokens: 0,
  realTokens: 0,
  requestCount: 0,
  cost: 0,
  cacheHitRate: 0,
  range: 'today',
  trend: [],
})

const port = useClientPort()
const loading = ref(false)
const err = ref('')
const range = ref<TokenUsageRange>('today')
const stats = ref<TokenUsageStats>(emptyStats())

const rangeOptions: { label: string; value: TokenUsageRange }[] = [
  { label: '当天', value: 'today' },
  { label: '近7天', value: '7d' },
  { label: '近30天', value: '30d' },
]

const cacheHitPercent = computed(() => Math.round((stats.value.cacheHitRate || 0) * 1000) / 10)

async function load() {
  loading.value = true
  err.value = ''
  try {
    const data = await port.getTokenUsage(range.value)
    stats.value = data ? { ...emptyStats(), ...data, trend: data.trend ?? [] } : emptyStats()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
    stats.value = emptyStats()
  } finally {
    loading.value = false
  }
}

function formatInt(n: number) {
  return Math.round(n || 0).toLocaleString('en-US')
}

function formatWan(n: number, digits = 1) {
  const v = n || 0
  if (Math.abs(v) < 10000) return `${v}`
  return `${(v / 10000).toFixed(digits)}万`
}

function formatCost(n: number) {
  return `$${(n || 0).toFixed(5)}`
}

function formatAxisLabel(at: string) {
  const m = at.match(/^(\d{4})-(\d{2})-(\d{2})(?: (\d{2}):(\d{2}))?/)
  if (!m) return at
  const label = `${Number(m[2])}/${Number(m[3])}`
  return m[4] ? `${label} ${m[4]}:${m[5] || '00'}` : label
}

type SeriesKey = 'promptTokens' | 'completionTokens' | 'cacheCreationTokens' | 'cacheReadTokens' | 'cost'

const series: { key: SeriesKey; label: string; color: string; area?: boolean; cost?: boolean }[] = [
  { key: 'cost', label: '成本', color: '#f87171', cost: true },
  { key: 'cacheCreationTokens', label: '缓存创建', color: '#f59e0b' },
  { key: 'cacheReadTokens', label: '缓存命中', color: '#60a5fa', area: true },
  { key: 'promptTokens', label: '输入', color: '#a78bfa', area: true },
  { key: 'completionTokens', label: '输出', color: '#34d399' },
]

const chart = computed(() => {
  const points = stats.value.trend || []
  const w = 720
  const h = 280
  const pad = { l: 48, r: 48, t: 16, b: 36 }
  const innerW = w - pad.l - pad.r
  const innerH = h - pad.t - pad.b
  const tokenMax = Math.max(
    1,
    ...points.flatMap((p) => [p.promptTokens, p.completionTokens, p.cacheCreationTokens, p.cacheReadTokens]),
  )
  const costMax = Math.max(1, ...points.map((p) => p.cost || 0), 0.01)
  const n = Math.max(points.length, 2)
  const xAt = (i: number) => pad.l + (innerW * i) / (n - 1)
  const yToken = (v: number) => pad.t + innerH - (innerH * v) / tokenMax
  const yCost = (v: number) => pad.t + innerH - (innerH * v) / costMax

  function line(key: SeriesKey, cost = false) {
    return points
      .map((p, i) => {
        const y = cost ? yCost(p.cost || 0) : yToken((p[key] as number) || 0)
        return `${i === 0 ? 'M' : 'L'}${xAt(i).toFixed(1)},${y.toFixed(1)}`
      })
      .join(' ')
  }

  function area(key: SeriesKey) {
    if (!points.length) return ''
    const body = line(key)
    const last = xAt(points.length - 1)
    const first = xAt(0)
    const base = pad.t + innerH
    return `${body} L${last.toFixed(1)},${base.toFixed(1)} L${first.toFixed(1)},${base.toFixed(1)} Z`
  }

  const ticks = pickTicks(points)
  return { w, h, pad, innerH, tokenMax, costMax, points, xAt, line, area, ticks }
})

function pickTicks(points: TokenUsageTrendPoint[]) {
  if (points.length <= 8) {
    return points.map((p, i) => ({ i, label: formatAxisLabel(p.at) }))
  }
  const step = Math.ceil(points.length / 6)
  const out: { i: number; label: string }[] = []
  for (let i = 0; i < points.length; i += step) {
    const point = points[i]
    if (!point) continue
    out.push({ i, label: formatAxisLabel(point.at) })
  }
  return out
}

onMounted(load)
watch(range, load)
</script>

<template>
  <div class="pane" :class="{ loading }">
    <header class="pane-head">
      <div>
        <h2>用量统计</h2>
        <p>统计当前账号的模型调用消耗，成本按模型单价（美元 / 百万 tokens）估算。</p>
      </div>
    </header>
    <p v-if="err" class="err">{{ err }}</p>

    <section class="card hero-card">
      <div class="hero">
        <div class="hero-main">
          <div class="icon token">⊕</div>
          <div>
            <div class="label">真实消耗 Tokens</div>
            <div class="hero-value">
              {{ formatInt(stats.realTokens) }}
              <small>≈{{ formatWan(stats.realTokens, 2) }}</small>
            </div>
          </div>
        </div>
        <div class="hero-side">
          <div>
            <span class="label">总请求数</span>
            <strong>{{ formatInt(stats.requestCount) }}</strong>
          </div>
          <div>
            <span class="label">总成本</span>
            <strong>{{ formatCost(stats.cost) }}</strong>
          </div>
        </div>
      </div>

      <div class="metrics">
        <div class="metric">
          <div class="icon">↑</div>
          <div>
            <div class="label">新增输入</div>
            <b>{{ formatWan(stats.promptTokens) }}</b>
          </div>
        </div>
        <div class="metric">
          <div class="icon">↓</div>
          <div>
            <div class="label">Output</div>
            <b>{{ formatWan(stats.completionTokens) }}</b>
          </div>
        </div>
        <div class="metric">
          <div class="icon">▣</div>
          <div>
            <div class="label">创建</div>
            <b>{{ formatWan(stats.cacheCreationTokens) }}</b>
          </div>
        </div>
        <div class="metric">
          <div class="icon">⚡</div>
          <div>
            <div class="label">命中</div>
            <b>{{ formatWan(stats.cacheReadTokens) }}</b>
          </div>
        </div>
        <div class="metric rate">
          <div class="rate-head">
            <span class="label">缓存命中率</span>
            <strong>{{ cacheHitPercent }}%</strong>
          </div>
          <div class="bar"><i :style="{ width: `${Math.min(cacheHitPercent, 100)}%` }" /></div>
        </div>
      </div>
    </section>

    <section class="card chart-card">
      <div class="chart-head">
        <h3>使用趋势</h3>
        <div class="range">
          <button
            v-for="opt in rangeOptions"
            :key="opt.value"
            type="button"
            :class="{ on: range === opt.value }"
            @click="range = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <svg
        class="chart"
        :viewBox="`0 0 ${chart.w} ${chart.h}`"
        preserveAspectRatio="none"
        role="img"
        aria-label="用量趋势"
      >
        <line
          :x1="chart.pad.l"
          :x2="chart.w - chart.pad.r"
          :y1="chart.pad.t + chart.innerH"
          :y2="chart.pad.t + chart.innerH"
          class="axis"
        />
        <text
          v-for="tick in chart.ticks"
          :key="tick.i"
          :x="chart.xAt(tick.i)"
          :y="chart.h - 10"
          class="tick"
          text-anchor="middle"
        >
          {{ tick.label }}
        </text>
        <text :x="12" :y="20" class="axis-label">{{ formatWan(chart.tokenMax, 0) }}</text>
        <text :x="chart.w - 8" :y="20" class="axis-label cost" text-anchor="end">${{ chart.costMax.toFixed(2) }}</text>
        <template v-for="s in series" :key="s.key">
          <path v-if="s.area" :d="chart.area(s.key)" :fill="s.color" class="area" />
          <path :d="chart.line(s.key, s.cost)" fill="none" :stroke="s.color" class="line" />
        </template>
      </svg>

      <div class="legend">
        <span v-for="s in series" :key="s.key">
          <i :style="{ background: s.color }" />
          {{ s.label }}
        </span>
      </div>
    </section>
  </div>
</template>

<style scoped>
.pane {
  max-width: 880px;
}

.pane.loading {
  opacity: 0.72;
}

.pane-head {
  margin-bottom: 20px;
}

h2 {
  font-family: var(--display);
  font-weight: 600;
  font-size: 24px;
  margin: 0;
  letter-spacing: -0.02em;
}

.pane-head p,
.label {
  color: var(--text-muted);
  font-size: 13px;
}

.pane-head p {
  margin: 6px 0 0;
}

.err {
  color: var(--error);
  margin: 0 0 12px;
  font-size: 13px;
}

.card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 20px 22px;
}

.hero-card {
  margin-bottom: 16px;
}

.hero {
  display: flex;
  justify-content: space-between;
  gap: 24px;
}

.hero-main {
  display: flex;
  gap: 12px;
}

.hero-value {
  margin-top: 6px;
  font-size: 30px;
  font-weight: 650;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}

.hero-value small {
  margin-left: 8px;
  font-size: 13px;
  font-weight: 400;
  color: var(--text-dim);
}

.hero-side {
  display: flex;
  gap: 24px;
}

.hero-side > div {
  display: grid;
  gap: 6px;
  justify-items: end;
}

.hero-side strong {
  font-size: 18px;
  font-variant-numeric: tabular-nums;
}

.metrics {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 14px;
  margin-top: 22px;
}

.metric {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.metric b {
  display: block;
  margin-top: 4px;
  font-size: 17px;
  font-variant-numeric: tabular-nums;
}

.icon {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--surface);
  color: var(--text-muted);
  display: grid;
  place-items: center;
  flex-shrink: 0;
  font-size: 14px;
}

.icon.token {
  background: var(--accent-soft);
  color: var(--accent);
}

.rate {
  display: block;
}

.rate-head {
  display: flex;
  justify-content: space-between;
  margin-bottom: 10px;
}

.bar {
  height: 8px;
  border-radius: var(--radius-full);
  background: var(--surface);
  overflow: hidden;
}

.bar i {
  display: block;
  height: 100%;
  background: var(--success);
}

.chart-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}

.range {
  display: flex;
  padding: 3px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
}

.range button {
  font: inherit;
  font-size: 12px;
  padding: 5px 12px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  border-radius: var(--radius-full);
}

.range button.on {
  background: var(--accent);
  color: #fff;
}

.chart {
  width: 100%;
  height: 280px;
}

.axis {
  stroke: var(--border);
}

.tick,
.axis-label {
  fill: var(--text-dim);
  font-size: 10px;
}

.axis-label.cost {
  fill: #f87171;
}

.line {
  stroke-width: 1.8;
  stroke-linejoin: round;
  stroke-linecap: round;
}

.area {
  opacity: 0.16;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  justify-content: center;
  color: var(--text-dim);
  font-size: 12px;
}

.legend i {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
}
</style>
