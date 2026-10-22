<script setup lang="ts">
import { useThemeStore, type ThemePreference } from '../../stores/theme'

const theme = useThemeStore()

const options: { value: ThemePreference; label: string; hint: string }[] = [
  { value: 'light', label: '亮色', hint: '浅色界面' },
  { value: 'dark', label: '暗色', hint: '深色界面' },
  { value: 'system', label: '系统', hint: '跟随系统外观' },
]

function pick(value: ThemePreference) {
  theme.setPreference(value)
}
</script>

<template>
  <div class="pane">
    <header class="pane-head">
      <h2>外观</h2>
    </header>
    <section class="card">
      <button
        v-for="opt in options"
        :key="opt.value"
        type="button"
        class="opt"
        :class="{ on: theme.preference === opt.value }"
        @click="pick(opt.value)"
      >
        <span>
          <strong>{{ opt.label }}</strong>
          <small>{{ opt.hint }}</small>
        </span>
        <span v-if="theme.preference === opt.value" class="check">✓</span>
      </button>
    </section>
  </div>
</template>

<style scoped>
.pane {
  display: grid;
  gap: 20px;
  max-width: 720px;
}

h2 {
  font-family: var(--display);
  font-weight: 600;
  font-size: 24px;
  margin: 0;
  letter-spacing: -0.02em;
}

.card {
  display: grid;
  gap: 4px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 8px;
}

.opt {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 10px 12px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  text-align: left;
  cursor: pointer;
}

.opt:hover {
  background: var(--surface-hover);
}

.opt.on {
  background: var(--accent-soft);
}

.opt span:first-child {
  flex: 1;
  display: grid;
  gap: 2px;
}

.opt strong {
  font-size: 13px;
  font-weight: 600;
}

.opt.on strong {
  color: var(--accent);
}

.opt small {
  font-size: 11px;
  color: var(--text-dim);
}

.check {
  color: var(--accent);
  font-size: 13px;
  font-weight: 700;
}
</style>
