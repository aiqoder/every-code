<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useClientPort } from '../../ports/clientPort'
import { protocolLabel, type ProviderProtocol } from '../../lib/providerPresets'
import type { LlmProvider, Model } from '../../types'

const router = useRouter()
const port = useClientPort()
const providers = ref<LlmProvider[]>([])
const models = ref<Model[]>([])
const err = ref('')

interface TestState {
  testing: boolean
  ok?: boolean
  message?: string
}
const testState = reactive<Record<number, TestState>>({})

const ownProviders = computed(() => providers.value.filter((p) => p.editable))

async function load() {
  err.value = ''
  try {
    const [p, m] = await Promise.all([port.listProviders(), port.listModels()])
    providers.value = p
    models.value = m
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

function modelsOf(providerId: number | undefined) {
  return models.value.filter((m) => m.providerId === providerId)
}

function protocolOf(p: LlmProvider): ProviderProtocol {
  return p.anthropicBaseUrl && !p.openaiCompatibleBaseUrl ? 'anthropic' : 'openai'
}

function baseUrlOf(p: LlmProvider) {
  return p.openaiCompatibleBaseUrl || p.anthropicBaseUrl || ''
}

function createProvider() {
  router.push({ name: 'settings-provider-new', query: router.currentRoute.value.query })
}

function editProvider(item: LlmProvider) {
  if (!item.editable || !item.id) return
  router.push({ name: 'settings-provider-edit', params: { id: item.id }, query: router.currentRoute.value.query })
}

async function removeProvider(item: LlmProvider) {
  if (!item.editable || !item.id) return
  if (!confirm(`删除供应商「${item.name}」？其下的模型也会一并删除。`)) return
  err.value = ''
  try {
    await port.deleteProvider(item.id!)
    await load()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

async function testProvider(item: LlmProvider) {
  if (!item.id) return
  testState[item.id] = { testing: true }
  try {
    const list = await port.listRemoteModels(item.id!)
    testState[item.id] = {
      testing: false,
      ok: true,
      message: list.length ? `连接成功，拉取到 ${list.length} 个模型` : '连接成功，但远程未返回模型列表',
    }
  } catch (e) {
    testState[item.id] = {
      testing: false,
      ok: false,
      message: `连接失败：${e instanceof Error ? e.message : String(e)}`,
    }
  }
}

function createModel(item: LlmProvider) {
  if (!item.editable || !item.id) return
  router.push({
    name: 'settings-model-new',
    query: { ...router.currentRoute.value.query, providerId: String(item.id) },
  })
}

function editModel(item: Model) {
  if (!item.editable || !item.id) return
  router.push({ name: 'settings-model-edit', params: { id: item.id }, query: router.currentRoute.value.query })
}

async function removeModel(item: Model) {
  if (!item.editable || !item.id) return
  if (!confirm(`删除模型「${item.name}」？`)) return
  err.value = ''
  try {
    await port.deleteModel(item.id!)
    await load()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

function scopeLabel(row: { userId?: number | null }) {
  return row.userId ? '我的' : '平台'
}

onMounted(load)
</script>

<template>
  <div class="pane">
    <header class="pane-head">
      <div>
        <h2>模型配置</h2>
        <p>每张卡片是一家供应商，模型挂在对应供应商下；平台配置可查看、可用于对话，但不可编辑。</p>
      </div>
      <button type="button" class="add" @click="createProvider">+ 新建供应商</button>
    </header>
    <p v-if="err" class="err">{{ err }}</p>

    <section v-for="p in providers" :key="p.id" class="card">
      <header class="card-head">
        <div class="info">
          <strong>
            {{ p.name }}
            <em>{{ scopeLabel(p) }}</em>
            <em class="proto">{{ protocolLabel(protocolOf(p)) }}</em>
          </strong>
          <small>{{ baseUrlOf(p) || p.website || '未配置接口地址' }}</small>
        </div>
        <div class="ops">
          <template v-if="p.editable">
            <button type="button" :disabled="testState[p.id!]?.testing" @click="testProvider(p)">
              {{ testState[p.id!]?.testing ? '测试中…' : '测试连接' }}
            </button>
            <button type="button" @click="editProvider(p)">编辑</button>
            <button type="button" class="danger" @click="removeProvider(p)">删除</button>
          </template>
          <span v-else class="readonly">只读</span>
        </div>
      </header>
      <p
        v-if="testState[p.id!]?.message"
        class="test-msg"
        :class="testState[p.id!]?.ok ? 'ok' : 'fail'"
      >
        {{ testState[p.id!]?.message }}
      </p>

      <ul class="models">
        <li v-for="m in modelsOf(p.id)" :key="m.id">
          <span class="mname">
            {{ m.name }}
            <em v-if="!m.userId">平台</em>
          </span>
          <span class="mops">
            <template v-if="m.editable">
              <button type="button" @click="editModel(m)">编辑</button>
              <button type="button" class="danger" @click="removeModel(m)">删除</button>
            </template>
            <span v-else class="readonly">只读</span>
          </span>
        </li>
        <li v-if="!modelsOf(p.id).length" class="empty">还没有模型</li>
      </ul>

      <div v-if="p.editable" class="card-foot">
        <button type="button" class="add-model" @click="createModel(p)">+ 添加模型</button>
      </div>
    </section>

    <div v-if="!providers.length" class="empty-all">
      <p>还没有供应商，先新建一个（内置常见供应商模板，只需填 API Key）。</p>
      <p v-if="!ownProviders.length && providers.length" class="err">请先创建自己的供应商，再维护模型</p>
    </div>
  </div>
</template>

<style scoped>
.pane {
  max-width: 760px;
}

.pane-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 20px;
}

h2 {
  font-family: var(--display);
  font-weight: 600;
  font-size: 24px;
  margin: 0;
  letter-spacing: -0.02em;
}

.pane-head p {
  margin: 6px 0 0;
  color: var(--text-muted);
  font-size: 13px;
}

.add {
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  padding: 8px 16px;
  background: var(--accent);
  border: 0;
  color: #fff;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.add:hover {
  background: var(--accent-hover);
}

.card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px 18px;
  margin-bottom: 14px;
}

.card:hover {
  border-color: var(--border-strong);
}

.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.info {
  min-width: 0;
}

strong {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

em {
  font-style: normal;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-dim);
  padding: 2px 7px;
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
}

em.proto {
  color: var(--text-muted);
}

small {
  display: block;
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
  align-items: center;
}

button {
  font: inherit;
  font-size: 12px;
  padding: 6px 12px;
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text-muted);
  border-radius: var(--radius-sm);
}

button:hover:not(:disabled) {
  color: var(--text);
  background: var(--surface-hover);
}

button:disabled {
  opacity: 0.6;
}

button.danger:hover:not(:disabled) {
  color: var(--error);
  border-color: rgba(248, 113, 113, 0.3);
}

.test-msg {
  margin: 10px 0 0;
  font-size: 12px;
}

.test-msg.ok {
  color: var(--success);
}

.test-msg.fail {
  color: var(--error);
}

.models {
  list-style: none;
  padding: 0;
  margin: 12px 0 0;
  display: grid;
  gap: 6px;
}

.models li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.mname {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mops {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
  align-items: center;
}

.mops button {
  padding: 4px 10px;
}

.models li.empty {
  justify-content: center;
  color: var(--text-dim);
  font-size: 13px;
  background: transparent;
  border-style: dashed;
}

.card-foot {
  margin-top: 10px;
}

.add-model {
  color: var(--text-muted);
  border-style: dashed;
}

.add-model:hover:not(:disabled) {
  color: var(--accent);
  border-color: var(--accent);
  background: transparent;
}

.readonly {
  font-size: 12px;
  color: var(--text-dim);
}

.empty-all {
  color: var(--text-dim);
  font-size: 14px;
  text-align: center;
  padding: 32px 0;
}

.err {
  color: var(--error);
  margin: 0 0 12px;
  font-size: 13px;
}
</style>
