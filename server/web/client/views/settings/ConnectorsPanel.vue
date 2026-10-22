<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import Dialog from '../../components/Dialog.vue'
import { useClientPort } from '../../ports/clientPort'
import type { ConnectorView } from '../../types'
import { useCapabilityBar } from './capabilityBar'
import { connectorToSpec, emptyConnectorSpec, formatSpec, parseConnectorDraft, readConnectorSpec } from './mcpDoc'

const port = useClientPort()
const bar = useCapabilityBar()
const list = ref<ConnectorView[]>([])

const shown = computed(() => {
  const q = bar.query.trim().toLowerCase()
  return list.value.filter((item) => {
    if (bar.installedOnly && !item.enabled) return false
    if (!q) return true
    return item.name.toLowerCase().includes(q) || (item.description || '').toLowerCase().includes(q) || item.url.toLowerCase().includes(q)
  })
})

watch(
  list,
  (items) => {
    bar.installedCount = items.length
  },
  { immediate: true },
)
const err = ref('')
const busy = ref(false)
const open = ref(false)
const editing = ref<ConnectorView | null>(null)
const name = ref('')
const description = ref('')
const spec = ref('')
const baseline = ref('')
const nameRef = ref<HTMLInputElement | null>(null)

const statusText = computed(() => (item: ConnectorView) => {
  if (!item.enabled) return '已停用'
  if (item.status === 'connected') return '已连接'
  if (item.status === 'connecting') return '连接中'
  if (item.status === 'error') return '连接失败'
  return '未连接'
})

const toolTarget = ref<ConnectorView | null>(null)

function endpointText(item: ConnectorView) {
  if (item.transport === 'stdio') return [item.command, ...(item.args ?? [])].filter(Boolean).join(' ')
  return item.url
}

function openTools(item: ConnectorView) {
  if (!item.tools?.length) return
  toolTarget.value = item
}

function closeTools() {
  toolTarget.value = null
}

function snapshot() {
  return JSON.stringify({
    name: name.value.trim(),
    description: description.value.trim(),
    spec: spec.value.trim(),
  })
}

const docError = computed(() => {
  if (!open.value || snapshot() === baseline.value) return ''
  try {
    draftInput()
    return ''
  } catch (e) {
    return e instanceof Error ? e.message : String(e)
  }
})
const canSave = computed(() => open.value && snapshot() !== baseline.value && !docError.value && !busy.value)

function draftInput() {
  const input = parseConnectorDraft(spec.value, name.value, description.value, editing.value)
  const clash = list.value.find((item) => item.name === input.name && item.id !== input.id)
  if (clash) throw new Error(`名称已存在：${input.name}`)
  return input
}

async function load() {
  err.value = ''
  try {
    list.value = await port.listConnectors()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

async function openEditor(item?: ConnectorView) {
  editing.value = item ?? null
  name.value = item?.name ?? ''
  description.value = item?.description ?? ''
  spec.value = item ? connectorToSpec(item) : emptyConnectorSpec()
  baseline.value = snapshot()
  err.value = ''
  open.value = true
  await nextTick()
  nameRef.value?.focus()
}

function closeEditor() {
  if (busy.value) return
  open.value = false
  err.value = ''
}

function absorbPastedSpec() {
  let read
  try {
    read = readConnectorSpec(spec.value)
  } catch {
    return
  }
  if (!read.pastedName) return
  if (!name.value.trim()) name.value = read.pastedName
  spec.value = formatSpec(read.spec)
}

function onSpecPaste() {
  nextTick(() => absorbPastedSpec())
}

async function save() {
  if (!canSave.value || busy.value) return
  absorbPastedSpec()
  let input
  try {
    input = draftInput()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
    return
  }
  busy.value = true
  err.value = ''
  try {
    list.value = await port.saveConnector(input)
    open.value = false
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
    await load()
  } finally {
    busy.value = false
  }
}

async function toggle(item: ConnectorView) {
  err.value = ''
  try {
    list.value = await port.setConnectorEnabled(item.id, !item.enabled)
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

async function remove(item: ConnectorView) {
  if (!confirm(`删除连接器「${item.name}」？已同步的工具会从对话中移除。`)) return
  err.value = ''
  try {
    list.value = await port.deleteConnector(item.id)
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

onMounted(() => {
  bar.placeholder = '搜索连接器'
  bar.addLabel = '添加 MCP'
  bar.onAdd = () => openEditor()
  load()
})

onUnmounted(() => {
  bar.onAdd = null
})
</script>

<template>
  <div class="pane">
    <p v-if="err && !open" class="err">{{ err }}</p>

    <div v-if="shown.length" class="list">
      <article v-for="item in shown" :key="item.id" class="card">
        <header class="card-head">
          <div class="identity">
            <div class="title-row">
              <strong>{{ item.name }}</strong>
              <em>{{ item.transport === 'stdio' ? '本地命令' : '远程' }}</em>
            </div>
            <p v-if="item.description" class="desc">{{ item.description }}</p>
            <p v-if="endpointText(item)" class="endpoint">{{ endpointText(item) }}</p>
          </div>
          <div class="head-ops">
            <span class="status" :class="[item.status, { off: !item.enabled }]">{{ statusText(item) }}</span>
            <button type="button" class="switch" :class="{ on: item.enabled }" @click="toggle(item)">
              {{ item.enabled ? '启用' : '停用' }}
            </button>
            <button type="button" @click="openEditor(item)">编辑</button>
            <button type="button" class="danger" @click="remove(item)">删除</button>
          </div>
        </header>
        <p v-if="item.lastError" class="err">{{ item.lastError }}</p>
        <button v-if="item.tools?.length" type="button" class="tools-link" @click="openTools(item)">
          {{ item.toolCount || item.tools.length }} 个工具
        </button>
      </article>
    </div>
    <p v-else class="empty">{{ list.length ? '没有匹配的连接器' : '还没有连接器。可添加远程 MCP 地址，或本机启动命令。' }}</p>

    <Dialog :open="!!toolTarget" :title="toolTarget ? `${toolTarget.name} · 工具` : ''" width="440px" @close="closeTools">
      <ul class="tool-list">
        <li v-for="tool in toolTarget?.tools ?? []" :key="tool.name">
          <strong>{{ tool.name }}</strong>
          <p v-if="tool.description">{{ tool.description }}</p>
        </li>
      </ul>
    </Dialog>

    <div v-if="open" class="modal-mask">
      <section class="mcp-dialog" role="dialog" aria-labelledby="mcp-title">
        <header class="mcp-head">
          <div class="mcp-titles">
            <h3 id="mcp-title">{{ editing ? '编辑 MCP' : '添加 MCP' }}</h3>
            <p>填写名称，并粘贴这一条服务的 JSON 配置</p>
          </div>
          <button type="button" class="mcp-x" aria-label="关闭" @click="closeEditor">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 6l12 12M18 6 6 18" /></svg>
          </button>
        </header>
        <form class="mcp-form" @submit.prevent="save">
          <label>
            名称
            <input ref="nameRef" v-model="name" type="text" autocomplete="off" placeholder="例如 filesystem" />
          </label>
          <label>
            说明
            <input v-model="description" type="text" autocomplete="off" placeholder="可选，显示在卡片上" />
          </label>
          <label class="spec">
            JSON 配置
            <textarea
              v-model="spec"
              spellcheck="false"
              placeholder='{ "command": "npx", "args": ["-y", "@modelcontextprotocol/server-filesystem"] }'
              @paste="onSpecPaste"
              @blur="absorbPastedSpec"
            ></textarea>
          </label>
          <p class="hint">填写 command、args、env，或 url、headers。粘贴只含一个服务的 mcpServers 时，会自动取出这一条。</p>
          <p v-if="docError || (err && open)" class="mcp-err">{{ docError || err }}</p>
          <div class="mcp-actions">
            <button type="button" class="mcp-cancel" @click="closeEditor">取消</button>
            <button type="submit" class="mcp-save" :disabled="!canSave">{{ busy ? '保存中…' : '保存' }}</button>
          </div>
        </form>
      </section>
    </div>
  </div>
</template>

<style scoped>
.pane {
  width: 100%;
  max-width: 960px;
  margin: 0 auto;
  align-self: center;
  padding: 8px 24px 28px;
}
h3 { margin: 0; font-family: var(--display); font-weight: 600; }
.empty {
  margin: 48px 0 0;
  color: var(--text-dim);
  font-size: 14px;
  text-align: center;
}
.list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 320px));
  gap: 12px;
  justify-content: start;
}
.card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 16px 18px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}
.card:hover { border-color: var(--border-strong); }
.card-head {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 10px;
}
.identity { min-width: 0; flex: 1; }
.title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.title-row strong {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.01em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.title-row em {
  flex-shrink: 0;
  font-style: normal;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-dim);
  padding: 1px 7px;
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
}
.desc, .endpoint, .status, .err { margin: 0; }
.desc {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.45;
}
.endpoint {
  margin-top: 2px;
  color: var(--text-dim);
  font-family: var(--mono);
  font-size: 12px;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.head-ops {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.status {
  font-size: 12px;
  color: var(--text-dim);
  white-space: nowrap;
}
.status.connected:not(.off) { color: var(--success); }
.status.error, .err { color: var(--error, #c24141); font-size: 12px; }
.head-ops button {
  font: inherit;
  font-size: 12px;
  padding: 5px 12px;
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text-muted);
  border-radius: var(--radius-sm);
}
.head-ops button:hover { color: var(--text); background: var(--surface-hover); }
.head-ops button.danger:hover { color: var(--error); border-color: rgba(248, 113, 113, 0.35); }
.switch.on {
  color: var(--success);
  background: var(--success-soft);
  border-color: transparent;
}
.tools-link {
  align-self: flex-start;
  margin-top: -4px;
  padding: 0;
  border: 0;
  background: none;
  color: var(--text-muted);
  font: inherit;
  font-size: 12px;
  line-height: 1.4;
  text-decoration: underline;
  text-underline-offset: 3px;
}
.tools-link:hover { color: var(--accent); }
.tool-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.tool-list li {
  min-width: 0;
}
.tool-list strong {
  display: block;
  font-family: var(--mono);
  font-size: 12px;
  font-weight: 560;
}
.tool-list p {
  margin: 2px 0 0;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
}
.modal-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: grid;
  place-items: center;
  z-index: 20;
  padding: 28px 20px;
}
.mcp-dialog {
  width: min(520px, 100%);
  background: var(--bg-elevated);
  border-radius: 16px;
  box-shadow: var(--shadow);
  color: var(--text);
}
.mcp-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px 18px 0;
}
.mcp-titles { min-width: 0; flex: 1; }
.mcp-titles h3 { font-size: 16px; letter-spacing: -0.01em; }
.mcp-titles p { margin: 2px 0 0; color: var(--text-muted); font-size: 12px; }
.mcp-x {
  width: 32px;
  height: 32px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  display: grid;
  place-items: center;
  cursor: pointer;
}
.mcp-x svg {
  width: 18px;
  height: 18px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.mcp-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 18px 18px;
}
.mcp-form label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
}
.mcp-form input,
.mcp-form textarea {
  font: inherit;
  font-size: 13px;
  color: var(--text);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px 10px;
  outline: none;
}
.mcp-form textarea {
  min-height: 180px;
  resize: vertical;
  font-family: var(--mono);
  line-height: 1.5;
}
.hint {
  margin: -4px 0 0;
  color: var(--text-muted);
  font-size: 12px;
}
.mcp-err {
  margin: 0;
  color: var(--error, #c24141);
  font-size: 12px;
}
.mcp-actions { display: flex; justify-content: flex-end; gap: 8px; }
.mcp-cancel,
.mcp-save {
  font: inherit;
  cursor: pointer;
  height: 34px;
  padding: 0 16px;
  border-radius: 8px;
  font-size: 13px;
}
.mcp-cancel {
  border: 1px solid var(--border-strong);
  background: var(--bg-elevated);
  color: var(--text);
}
.mcp-save {
  border: 0;
  background: var(--text);
  color: var(--bg-elevated);
}
.mcp-save:disabled {
  background: var(--surface-hover);
  color: var(--text-dim);
  cursor: default;
}
</style>
