<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useClientPort } from '../../ports/clientPort'

const port = useClientPort()
const link = computed(() => port.linkState())
const form = reactive({
  clientId: '',
  clientName: '',
})

watch(
  link,
  (s) => {
    form.clientId = s.clientId
    form.clientName = s.clientName
  },
  { immediate: true },
)

const wsLabel = computed(() => {
  if (!link.value.loggedIn) return '未登录'
  switch (link.value.wsStatus) {
    case 'OPEN':
      return '已连接'
    case 'CONNECTING':
      return '连接中…'
    default:
      return link.value.connected ? '已连接' : '重连中…'
  }
})

async function save() {
  await port.saveLink(form.clientId, form.clientName)
}
</script>

<template>
  <div class="pane">
    <header class="pane-head">
      <h2>接入</h2>
      <p class="ws">
        <span class="dot" :class="{ on: link.connected }" />
        {{ wsLabel }}
      </p>
    </header>

    <section class="card">
      <label>客户端 ID<input v-model="form.clientId" :disabled="link.clientIdReadonly" /></label>
      <label>显示名<input v-model="form.clientName" /></label>
      <div class="row">
        <button type="button" class="solid" :disabled="link.busy" @click="save">保存</button>
      </div>
      <p v-if="link.notice" class="err">{{ link.notice }}</p>
      <p v-if="link.lastError" class="err">{{ link.lastError }}</p>
    </section>
  </div>
</template>

<style scoped>
.pane {
  display: grid;
  gap: 20px;
  max-width: 720px;
}

.pane-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
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
  gap: 14px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 20px;
}

label {
  display: grid;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-muted);
}

input {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
  padding: 10px 12px;
  font-size: 14px;
}

.row {
  display: flex;
  gap: 10px;
  padding-top: 4px;
}

button {
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  padding: 9px 18px;
  border-radius: var(--radius-sm);
}

button:disabled {
  opacity: 0.45;
}

.solid {
  background: var(--accent);
  border: 0;
  color: #fff;
}

.solid:hover:not(:disabled) {
  background: var(--accent-hover);
}

.err {
  color: var(--error);
  margin: 0;
  font-size: 13px;
}

.ws {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}

.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-dim);
  flex-shrink: 0;
}

.dot.on {
  background: var(--success);
  box-shadow: 0 0 8px var(--success);
}
</style>
