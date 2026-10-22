<script setup lang="ts">
import { watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { provideCapabilityBar } from './capabilityBar'

const route = useRoute()
const bar = provideCapabilityBar()

watch(
  () => route.name,
  () => {
    bar.query = ''
    bar.installedOnly = false
  },
)

function add() {
  bar.onAdd?.()
}
</script>

<template>
  <div class="hub">
    <header class="bar">
      <div class="tools">
        <label class="search">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="11" cy="11" r="6" />
            <path d="m20 20-3.5-3.5" />
          </svg>
          <input v-model="bar.query" type="search" :placeholder="bar.placeholder" />
        </label>
        <button
          type="button"
          class="chip"
          :class="{ on: bar.installedOnly }"
          @click="bar.installedOnly = !bar.installedOnly"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <rect x="4" y="4" width="16" height="16" rx="3" />
            <path d="m8 12 2.5 2.5L16 9" />
          </svg>
          我安装的 {{ bar.installedCount }}
        </button>
        <button type="button" class="add" :disabled="bar.addDisabled || !bar.onAdd" @click="add">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M12 5v14M5 12h14" />
          </svg>
          {{ bar.addLabel }}
        </button>
      </div>
    </header>
    <RouterView />
  </div>
</template>

<style scoped>
.hub {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: auto;
}

.bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16px;
  padding: 16px 24px 8px;
  flex-shrink: 0;
}

.tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.search {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  color: var(--text-muted);
}

.search svg,
.chip svg,
.add svg {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.search input {
  width: 168px;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: 13px;
}

.chip,
.add {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  color: var(--text);
  font: inherit;
  font-size: 13px;
  white-space: nowrap;
}

.chip.on {
  border-color: var(--text);
}

.add:disabled {
  opacity: 0.5;
}
</style>
