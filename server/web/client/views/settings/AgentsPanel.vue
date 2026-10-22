<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useClientPort } from '../../ports/clientPort'
import type { Agent, PackageEnvField, PackageInfo } from '../../types'
import { bytesToBase64 } from '../../lib/bytes'
import { emptyEnvListRow, parseEnvList, serializeEnvList } from '../../lib/envList'
import Dialog from '../../components/Dialog.vue'
import IconEye from '../../components/icons/IconEye.vue'
import IconEyeOff from '../../components/icons/IconEyeOff.vue'
import HoverTip from '../chat/components/HoverTip.vue'
import { useCapabilityBar } from './capabilityBar'

const port = useClientPort()
const bar = useCapabilityBar()
const fileInput = ref<HTMLInputElement | null>(null)
const packages = ref<PackageInfo[]>([])
const packagesRoot = ref('')
const agents = ref<Agent[]>([])
const err = ref('')
const installing = ref(false)

const panelOpen = ref(false)
const panelErr = ref('')
const dragging = ref(false)
const pickedFile = ref<File | null>(null)
const pickedPath = ref('')
const pickedName = ref('')
let dragDepth = 0
let offCpkDrop: (() => void) | undefined

// 环境变量配置弹窗
const envTarget = ref<PackageInfo | null>(null)
const envValues = ref<Record<string, string>>({})
/** list 类型字段的行编辑态：fieldName → 行 → 列值 */
const envListRows = ref<Record<string, string[][]>>({})
const envBusy = ref(false)
const envErr = ref('')
/** secret 字段明文可见：key 为 fieldName 或 fieldName:row:col */
const envReveal = ref<Record<string, boolean>>({})

function isListField(field: PackageEnvField) {
  return field.type === 'list' && (field.items?.length ?? 0) > 0
}

function listGridStyle(field: PackageEnvField) {
  const n = field.items?.length ?? 1
  return {
    gridTemplateColumns: `repeat(${n}, minmax(0, 1fr)) 52px`,
  }
}

function listColPlaceholder(col: { label?: string; required?: boolean }) {
  return col.required ? '必填' : '可选'
}

function isEnvRevealed(key: string) {
  return !!envReveal.value[key]
}

function toggleEnvReveal(key: string) {
  envReveal.value = { ...envReveal.value, [key]: !envReveal.value[key] }
}

function listSecretKey(fieldName: string, row: number, colName: string) {
  return `${fieldName}:${row}:${colName}`
}

const envDialogWidth = computed(() =>
  (envTarget.value?.env ?? []).some(isListField) ? '920px' : '420px',
)

// 已安装包的 ID。这些智能体由安装包提供，不单独再列一行。
const packageAgentIds = computed(() => new Set(packages.value.map((p) => p.id)))

const plainAgents = computed(() => agents.value.filter((a) => !packageAgentIds.value.has(a.id)))

function hit(text: string, q: string) {
  return text.toLowerCase().includes(q)
}

const shownPackages = computed(() => {
  const q = bar.query.trim().toLowerCase()
  return packages.value.filter((item) => !q || hit(item.name, q) || hit(item.description || '', q))
})

const shownAgents = computed(() => {
  if (bar.installedOnly) return []
  const q = bar.query.trim().toLowerCase()
  return plainAgents.value.filter((item) => !q || hit(item.name, q) || hit(item.description || '', q))
})

watch(
  packages,
  (items) => {
    bar.installedCount = items.length
  },
  { immediate: true },
)

watch(installing, (value) => {
  bar.addDisabled = value
})

function resetPick() {
  pickedFile.value = null
  pickedPath.value = ''
  pickedName.value = ''
  dragging.value = false
  dragDepth = 0
}

function openPanel() {
  if (installing.value) return
  panelErr.value = ''
  resetPick()
  panelOpen.value = true
}

function closePanel() {
  if (installing.value) return
  panelOpen.value = false
  panelErr.value = ''
  resetPick()
}

function acceptFile(file: File) {
  if (!file.name.toLowerCase().endsWith('.cpk')) {
    panelErr.value = '请选择 .cpk 文件'
    return
  }
  panelErr.value = ''
  pickedFile.value = file
  pickedPath.value = ''
  pickedName.value = file.name
}

function acceptPath(path: string) {
  const name = path.split(/[/\\]/).pop() || path
  if (!name.toLowerCase().endsWith('.cpk')) {
    panelErr.value = '请拖入 .cpk 文件'
    return
  }
  panelErr.value = ''
  pickedFile.value = null
  pickedPath.value = path
  pickedName.value = name
}

function pickFile() {
  if (installing.value) return
  fileInput.value?.click()
}

function onFile(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (file) acceptFile(file)
}

function onDragEnter(ev: DragEvent) {
  ev.preventDefault()
  dragDepth += 1
  dragging.value = true
}

function onDragOver(ev: DragEvent) {
  ev.preventDefault()
  if (ev.dataTransfer) ev.dataTransfer.dropEffect = 'copy'
}

function onDragLeave(ev: DragEvent) {
  ev.preventDefault()
  dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) dragging.value = false
}

function onHtmlDrop(ev: DragEvent) {
  ev.preventDefault()
  dragging.value = false
  dragDepth = 0
  const file = ev.dataTransfer?.files?.[0]
  if (file) acceptFile(file)
}

function onKeydown(ev: KeyboardEvent) {
  if (ev.key === 'Escape' && panelOpen.value) closePanel()
}

async function load() {
  err.value = ''
  try {
    const [pkgs, root, agts] = await Promise.all([
      port.listPackages(),
      port.packageRoot(),
      port.listAgents(),
    ])
    packages.value = pkgs
    packagesRoot.value = root
    agents.value = agts
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

async function openPackagesRoot() {
  err.value = ''
  try {
    await port.openPackageRoot()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

async function confirmInstall() {
  if (installing.value || !pickedName.value) return
  installing.value = true
  err.value = ''
  panelErr.value = ''
  try {
    if (pickedFile.value) {
      const buf = new Uint8Array(await pickedFile.value.arrayBuffer())
      await port.installPackage(pickedFile.value.name, bytesToBase64(buf))
    } else if (pickedPath.value) {
      await port.installPackageFromPath(pickedPath.value)
    } else {
      panelErr.value = '请先选择 .cpk 文件'
      return
    }
    panelOpen.value = false
    resetPick()
    await load()
  } catch (e) {
    panelErr.value = e instanceof Error ? e.message : String(e)
  } finally {
    installing.value = false
  }
}

async function uninstall(pkg: PackageInfo) {
  if (!confirm(`卸载智能体包「${pkg.name}」？将同时删除其注册的智能体与本地文件。`)) return
  err.value = ''
  try {
    await port.uninstallPackage(pkg.id)
    await load()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

function openEnv(pkg: PackageInfo) {
  envTarget.value = pkg
  const values: Record<string, string> = {}
  const lists: Record<string, string[][]> = {}
  for (const field of pkg.env ?? []) {
    if (isListField(field)) {
      const items = field.items ?? []
      const rows = parseEnvList(items, field.value ?? '')
      lists[field.name] = rows.length > 0 ? rows : [emptyEnvListRow(items.length)]
    } else {
      values[field.name] = field.value ?? ''
    }
  }
  envValues.value = values
  envListRows.value = lists
  envReveal.value = {}
  envErr.value = ''
}

function closeEnv() {
  envTarget.value = null
  envListRows.value = {}
  envReveal.value = {}
}

function addListRow(field: PackageEnvField) {
  const items = field.items ?? []
  const cur = envListRows.value[field.name] ?? []
  envListRows.value = {
    ...envListRows.value,
    [field.name]: [...cur, emptyEnvListRow(items.length)],
  }
}

function removeListRow(fieldName: string, index: number) {
  const cur = [...(envListRows.value[fieldName] ?? [])]
  if (cur.length <= 1) {
    const field = envTarget.value?.env?.find((f) => f.name === fieldName)
    cur[0] = emptyEnvListRow(field?.items?.length ?? 0)
  } else {
    cur.splice(index, 1)
  }
  envListRows.value = { ...envListRows.value, [fieldName]: cur }
}

async function submitEnv() {
  const pkg = envTarget.value
  if (!pkg || envBusy.value) return
  envBusy.value = true
  envErr.value = ''
  try {
    const payload: Record<string, string> = { ...envValues.value }
    for (const field of pkg.env ?? []) {
      if (!isListField(field)) continue
      payload[field.name] = serializeEnvList(field.items ?? [], envListRows.value[field.name] ?? [])
    }
    await port.savePackageEnv(pkg.id, payload)
    closeEnv()
    await load()
  } catch (e) {
    envErr.value = e instanceof Error ? e.message : String(e)
  } finally {
    envBusy.value = false
  }
}

async function clearEnv() {
  const pkg = envTarget.value
  if (!pkg || envBusy.value) return
  envBusy.value = true
  envErr.value = ''
  try {
    await port.clearPackageEnv(pkg.id)
    closeEnv()
    await load()
  } catch (e) {
    envErr.value = e instanceof Error ? e.message : String(e)
  } finally {
    envBusy.value = false
  }
}

function descText(row: Agent) {
  return row.description?.trim() || ''
}

onMounted(() => {
  bar.placeholder = '搜索智能体'
  bar.addLabel = '添加智能体'
  bar.onAdd = openPanel
  window.addEventListener('keydown', onKeydown)
  offCpkDrop = port.subscribeCpkDrop?.(
    (data) => {
      if (!panelOpen.value) return
      if (data.error) {
        panelErr.value = data.error
        return
      }
      if (data.path) acceptPath(data.path)
    },
    'agent-cpk-dropped',
  )
  load()
})

onUnmounted(() => {
  bar.onAdd = null
  bar.addDisabled = false
  window.removeEventListener('keydown', onKeydown)
  offCpkDrop?.()
})
</script>

<template>
  <div class="pane">
    <p v-if="err" class="err">{{ err }}</p>

    <section v-if="shownPackages.length" class="section">
      <div class="section-head">
        <h3 class="section-title">已安装的智能体包</h3>
        <button
          v-if="packagesRoot"
          type="button"
          class="path-link"
          :title="packagesRoot"
          @click="openPackagesRoot"
        >
          {{ packagesRoot }}
        </button>
      </div>
      <div class="grid">
        <article v-for="p in shownPackages" :key="p.id" class="card">
          <div class="card-top">
            <img v-if="p.avatar" :src="p.avatar" class="avatar" alt="" />
            <span v-else class="avatar fallback">{{ p.name.slice(0, 1) }}</span>
            <div class="title-wrap">
              <div class="name-row">
                <strong class="name" :title="p.name">{{ p.name }}</strong>
                <span class="ver">v{{ p.version }}</span>
              </div>
              <div class="meta-row">
                <span v-if="p.author" class="author" :title="p.author">{{ p.author }}</span>
                <div class="actions">
                  <button
                    v-if="p.env?.length"
                    type="button"
                    class="text-btn"
                    :class="{ warn: !p.envReady }"
                    @click="openEnv(p)"
                  >
                    配置
                  </button>
                  <button type="button" class="text-btn danger" @click="uninstall(p)">卸载</button>
                </div>
              </div>
            </div>
          </div>

          <HoverTip :content="p.description" :disabled="!p.description">
            <p class="desc">{{ p.description || '暂无简介' }}</p>
          </HoverTip>
        </article>
      </div>
    </section>

    <section v-if="shownAgents.length" class="section">
      <h3 class="section-title">其他可用智能体</h3>
      <div class="grid">
        <article v-for="a in shownAgents" :key="a.id" class="card">
          <div class="card-top">
            <img v-if="a.avatar" :src="a.avatar" class="avatar" alt="" />
            <span v-else class="avatar fallback">{{ a.name.slice(0, 1) }}</span>
            <div class="title-wrap">
              <div class="name-row">
                <strong class="name" :title="a.name">{{ a.name }}</strong>
                <span v-if="a.version" class="ver">v{{ a.version }}</span>
              </div>
              <div class="meta-row">
                <span class="author">我的</span>
                <div class="actions">
                  <span class="readonly">只读</span>
                </div>
              </div>
            </div>
          </div>
          <HoverTip :content="descText(a)" :disabled="!descText(a)">
            <p class="desc">{{ descText(a) || '暂无简介' }}</p>
          </HoverTip>
        </article>
      </div>
    </section>

    <p v-if="!shownPackages.length && !shownAgents.length" class="empty">
      {{ packages.length || plainAgents.length ? '没有匹配的智能体' : '还没有已安装的智能体。可到商店安装，或点击右上角添加 .cpk' }}
    </p>

    <div v-if="panelOpen" class="overlay">
      <button type="button" class="backdrop" aria-label="关闭" :disabled="installing" @click="closePanel" />
      <div
        id="agent-cpk-drop"
        class="panel"
        data-file-drop-target
        role="dialog"
        aria-modal="true"
        aria-label="添加智能体"
        @dragenter="onDragEnter"
        @dragover="onDragOver"
        @dragleave="onDragLeave"
        @drop="onHtmlDrop"
      >
        <h3>添加智能体</h3>
        <p class="tip">将 .cpk 拖到下方区域，或点击选择文件。确认后安装到本机。</p>
        <div
          class="drop"
          role="button"
          tabindex="0"
          :class="{ over: dragging, 'has-file': pickedName, disabled: installing }"
          @click="pickFile"
          @keydown.enter.prevent="pickFile"
          @keydown.space.prevent="pickFile"
        >
          <input
            ref="fileInput"
            class="file-input"
            type="file"
            accept=".cpk,application/octet-stream"
            :disabled="installing"
            @click.stop
            @change="onFile"
          />
          <template v-if="pickedName">
            <strong>{{ pickedName }}</strong>
            <span>点击或拖入可更换文件</span>
          </template>
          <template v-else>
            <strong>拖入 .cpk</strong>
            <span>或点击选择文件</span>
          </template>
        </div>
        <p v-if="panelErr" class="err">{{ panelErr }}</p>
        <footer>
          <button type="button" class="btn ghost" :disabled="installing" @click="closePanel">取消</button>
          <button type="button" class="btn primary" :disabled="installing || !pickedName" @click="confirmInstall">
            {{ installing ? '安装中…' : '确认安装' }}
          </button>
        </footer>
      </div>
    </div>

    <Dialog
      :open="!!envTarget"
      :title="envTarget ? `配置「${envTarget.name}」` : ''"
      :width="envDialogWidth"
      :close-on-backdrop="false"
      @close="closeEnv"
    >
      <p class="modal-tip">按该智能体开发者的要求填写。值只保存在本机，运行脚本时注入为对应环境变量。</p>
      <template v-for="field in envTarget?.env ?? []" :key="field.name">
        <div v-if="isListField(field)" class="field list-field">
          <span>
            {{ field.label || field.name }}
            <em v-if="field.required">*</em>
            <code>{{ field.name }}</code>
          </span>
          <div class="list-editor">
            <div class="list-grid list-head" :style="listGridStyle(field)">
              <span
                v-for="col in field.items"
                :key="col.name"
                class="list-th"
                :title="col.label || col.name"
              >
                {{ col.label || col.name }}
                <em v-if="col.required">*</em>
              </span>
              <span class="list-th list-th-action" aria-hidden="true"></span>
            </div>
            <div
              v-for="(row, ri) in envListRows[field.name] || []"
              :key="ri"
              class="list-grid list-row"
              :style="listGridStyle(field)"
            >
              <template v-for="(col, ci) in field.items" :key="col.name">
                <div v-if="col.secret" class="secret-wrap list-secret">
                  <input
                    v-model="row[ci]"
                    class="list-cell"
                    :type="isEnvRevealed(listSecretKey(field.name, ri, col.name)) ? 'text' : 'password'"
                    autocomplete="off"
                    :placeholder="listColPlaceholder(col)"
                    :title="col.label || col.name"
                  />
                  <button
                    type="button"
                    class="secret-toggle"
                    :aria-label="isEnvRevealed(listSecretKey(field.name, ri, col.name)) ? '隐藏' : '显示'"
                    :title="isEnvRevealed(listSecretKey(field.name, ri, col.name)) ? '隐藏' : '显示'"
                    @click="toggleEnvReveal(listSecretKey(field.name, ri, col.name))"
                  >
                    <IconEyeOff v-if="isEnvRevealed(listSecretKey(field.name, ri, col.name))" />
                    <IconEye v-else />
                  </button>
                </div>
                <input
                  v-else
                  v-model="row[ci]"
                  class="list-cell"
                  type="text"
                  autocomplete="off"
                  :placeholder="listColPlaceholder(col)"
                  :title="col.label || col.name"
                />
              </template>
              <button
                type="button"
                class="list-remove"
                :disabled="envBusy"
                title="删除此行"
                @click="removeListRow(field.name, ri)"
              >
                删除
              </button>
            </div>
            <button type="button" class="list-add" :disabled="envBusy" @click="addListRow(field)">
              + 添加一行
            </button>
          </div>
        </div>
        <div v-else class="field">
          <span>
            {{ field.label || field.name }}
            <em v-if="field.required">*</em>
            <code>{{ field.name }}</code>
          </span>
          <div v-if="field.secret" class="secret-wrap">
            <input
              v-model="envValues[field.name]"
              :type="isEnvRevealed(field.name) ? 'text' : 'password'"
              autocomplete="off"
              :placeholder="field.required ? '必填' : '可选'"
              @keyup.enter="submitEnv"
            />
            <button
              type="button"
              class="secret-toggle"
              :aria-label="isEnvRevealed(field.name) ? '隐藏' : '显示'"
              :title="isEnvRevealed(field.name) ? '隐藏' : '显示'"
              @click="toggleEnvReveal(field.name)"
            >
              <IconEyeOff v-if="isEnvRevealed(field.name)" />
              <IconEye v-else />
            </button>
          </div>
          <input
            v-else
            v-model="envValues[field.name]"
            type="text"
            autocomplete="off"
            :placeholder="field.required ? '必填' : '可选'"
            @keyup.enter="submitEnv"
          />
        </div>
      </template>
      <p v-if="envErr" class="err">{{ envErr }}</p>
      <template #footer>
        <button type="button" class="env-clear" :disabled="envBusy" @click="clearEnv">清除</button>
        <button type="button" @click="closeEnv">取消</button>
        <button type="button" class="primary" :disabled="envBusy" @click="submitEnv">
          {{ envBusy ? '保存中…' : '保存' }}
        </button>
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
.pane {
  width: 100%;
  max-width: 960px;
  margin: 0 auto;
  align-self: center;
  padding: 20px 24px 28px;
}

.overlay {
  position: fixed;
  inset: 0;
  z-index: 130;
  display: grid;
  place-items: center;
  padding: 24px;
}

.backdrop {
  position: absolute;
  inset: 0;
  border: 0;
  background: rgba(0, 0, 0, 0.5);
}

.panel {
  position: relative;
  z-index: 1;
  width: min(440px, 100%);
  padding: 18px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
}

.panel h3 {
  margin: 0 0 8px;
  font-size: 16px;
}

.tip {
  margin: 0 0 14px;
  font-size: 13px;
  color: var(--text-muted);
  line-height: 1.5;
}

.drop {
  min-height: 148px;
  border: 1.5px dashed var(--border-strong);
  border-radius: var(--radius);
  background: var(--surface);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 20px;
  text-align: center;
  cursor: pointer;
  color: var(--text);
}

.drop strong {
  font-size: 15px;
  font-weight: 600;
}

.drop span {
  font-size: 12px;
  color: var(--text-muted);
}

.drop.over,
.drop.has-file,
.panel.file-drop-target-active .drop {
  border-color: var(--accent);
  background: var(--accent-soft);
}

.drop.disabled {
  opacity: 0.6;
  pointer-events: none;
}

.file-input {
  display: none;
}

.panel footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 14px;
}

.btn {
  font: inherit;
  font-size: 13px;
  padding: 7px 14px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  cursor: pointer;
}

.btn.ghost {
  background: transparent;
  color: var(--text);
}

.btn.primary {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}

.btn:disabled {
  opacity: 0.55;
  cursor: default;
}

.pane-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

h2 {
  font-family: var(--display);
  font-weight: 600;
  font-size: 24px;
  margin: 0;
  letter-spacing: -0.02em;
}

.add-label {
  display: inline-flex;
  align-items: center;
  font-size: 13px;
  font-weight: 500;
  padding: 8px 16px;
  background: var(--accent);
  color: #fff;
  border-radius: var(--radius-full);
  border: 0;
  cursor: pointer;
}

.add-label:hover {
  background: var(--accent-hover);
}

.section {
  margin-bottom: 20px;
}

.section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin: 0 0 10px;
  min-width: 0;
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-muted);
  margin: 0 0 10px;
  flex-shrink: 0;
}

.section-head .section-title {
  margin: 0;
}

.path-link {
  margin: 0;
  padding: 0;
  border: 0;
  background: none;
  font-family: var(--mono);
  font-size: 11px;
  line-height: 16px;
  color: var(--text-dim);
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
  min-width: 0;
  max-width: 60%;
}

.path-link:hover {
  color: var(--accent);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
}

.card {
  display: flex;
  flex-direction: column;
  gap: 0;
  min-width: 0;
  padding: 14px 16px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  transition: border-color 0.12s;
}

.card:hover {
  border-color: var(--border-strong);
}

.card-top {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.avatar {
  width: 36px;
  height: 36px;
  border-radius: 9px;
  object-fit: cover;
  flex-shrink: 0;
  background: var(--surface);
}

.avatar.fallback {
  display: grid;
  place-items: center;
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 13px;
  font-weight: 600;
}

.title-wrap {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.name-row {
  display: flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
}

.name {
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ver {
  flex-shrink: 0;
  font-family: var(--mono);
  font-size: 11px;
  font-weight: 500;
  line-height: 16px;
  color: var(--text-dim);
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  min-height: 18px;
}

.author {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  line-height: 18px;
  color: var(--text-dim);
}

.actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  margin-left: auto;
}

.text-btn,
.readonly {
  margin: 0;
  padding: 0;
  border: 0;
  background: none;
  font: inherit;
  font-size: 12px;
  line-height: 18px;
  color: var(--text-muted);
}

.text-btn {
  cursor: pointer;
}

.text-btn + .text-btn {
  position: relative;
  margin-left: 16px;
}

.text-btn + .text-btn::before {
  content: '';
  position: absolute;
  left: -8px;
  top: 4px;
  width: 1px;
  height: 10px;
  background: var(--border-strong);
}

.text-btn:hover {
  color: var(--text);
}

.text-btn.warn {
  color: var(--error);
}

.text-btn.danger:hover {
  color: var(--error);
}

.desc {
  margin: 8px 0 0;
  padding-top: 8px;
  border-top: 1px solid var(--border);
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  word-break: break-word;
  min-height: calc(8px + 1.5em * 2);
}

.empty {
  margin: 24px 0 0;
  color: var(--text-dim);
  font-size: 14px;
  text-align: center;
}

.err {
  color: var(--error);
  margin: 0 0 12px;
  font-size: 13px;
}

.panel .err {
  margin: 10px 0 0;
}

.modal-tip {
  margin: 0 0 14px;
  font-size: 12px;
  color: var(--text-muted);
  word-break: break-all;
}

.env-clear {
  margin-right: auto;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
  font-size: 12px;
  color: var(--text-muted);
}

.field em {
  color: var(--error);
  font-style: normal;
  margin-left: 2px;
}

.field code {
  margin-left: 6px;
  font-size: 11px;
  color: var(--text-dim);
}

.field input {
  font: inherit;
  font-size: 13px;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  color: var(--text);
  outline: none;
}

.field input:focus {
  border-color: var(--accent);
}

.secret-wrap {
  position: relative;
  display: flex;
  align-items: center;
  min-width: 0;
  width: 100%;
}

.secret-wrap input {
  width: 100%;
  padding-right: 34px;
  box-sizing: border-box;
}

.secret-toggle {
  position: absolute;
  right: 4px;
  top: 50%;
  transform: translateY(-50%);
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  margin: 0;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
}

.secret-toggle:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.secret-toggle svg {
  width: 16px;
  height: 16px;
  display: block;
}

.list-secret .secret-toggle {
  right: 2px;
  width: 26px;
  height: 26px;
}

.list-secret .list-cell {
  padding-right: 30px;
}

.list-editor {
  display: flex;
  flex-direction: column;
  gap: 6px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  padding: 8px;
  overflow-x: auto;
}

.list-grid {
  display: grid;
  gap: 8px;
  align-items: center;
  min-width: 640px;
}

.list-head {
  padding: 0 0 4px;
  border-bottom: 1px solid var(--border);
}

.list-th {
  font-size: 11px;
  line-height: 1.35;
  color: var(--text-muted);
  min-height: 2.7em;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  word-break: break-word;
}

.list-th-action {
  min-height: 0;
}

.list-row {
  padding: 2px 0;
}

.list-cell {
  font: inherit;
  font-size: 13px;
  height: 34px;
  box-sizing: border-box;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text);
  outline: none;
  min-width: 0;
  width: 100%;
}

.list-cell:focus {
  border-color: var(--accent);
}

.list-remove,
.list-add {
  font: inherit;
  font-size: 12px;
  height: 34px;
  padding: 0 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  white-space: nowrap;
}

.list-add {
  align-self: flex-start;
  margin-top: 4px;
  border-style: dashed;
  height: 32px;
}

.list-remove:disabled,
.list-add:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
