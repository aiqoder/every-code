<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { Agent } from '../../../types'
import { skillAvatarColor, skillInitial } from '../../../lib/skillAvatar'
import IconFolder from './icons/IconFolder.vue'
import IconPlus from './icons/IconPlus.vue'
import IconChevronDown from './icons/IconChevronDown.vue'

const props = defineProps<{
  /** 已加入的工作目录路径列表 */
  workspaces: string[]
  /** 当前工作目录 */
  currentWorkspace: string
  /** 已安装的智能体 */
  agents: Agent[]
  /** 当前选中的智能体安装包 ID */
  activeId: string | null
}>()

const emit = defineEmits<{
  /** 切换工作目录 */
  selectWorkspace: [path: string]
  /** 新建工作目录（弹出系统选择框） */
  pickWorkspace: []
  /** 从列表移除工作目录 */
  removeWorkspace: [path: string]
  /** 在文件管理器中打开工作目录 */
  openWorkspaceFolder: [path: string]
  /** 选中智能体 */
  selectAgent: [id: string]
}>()

const dropdownOpen = ref(false)
const dropdownWrap = ref<HTMLElement | null>(null)

/** 点击下拉外部时收起目录选择。 */
function onDocPointerDown(e: PointerEvent) {
  if (!dropdownOpen.value) return
  if (dropdownWrap.value && e.target instanceof Node && dropdownWrap.value.contains(e.target)) return
  dropdownOpen.value = false
}

onMounted(() => document.addEventListener('pointerdown', onDocPointerDown))
onUnmounted(() => document.removeEventListener('pointerdown', onDocPointerDown))

/** 下拉展示的目录名（取最后一段路径） */
const currentLabel = computed(() => labelOf(props.currentWorkspace))

function labelOf(path: string) {
  const p = (path || '').replace(/\\/g, '/').replace(/\/+$/, '')
  const i = p.lastIndexOf('/')
  return i >= 0 ? p.slice(i + 1) || p : p
}

function samePath(a?: string, b?: string) {
  return (a || '').replace(/\\/g, '/').replace(/\/+$/, '') === (b || '').replace(/\\/g, '/').replace(/\/+$/, '')
}

/** 选中下拉中的目录 */
function pickWorkspace(path: string) {
  dropdownOpen.value = false
  if (!samePath(path, props.currentWorkspace)) emit('selectWorkspace', path)
}

function fallbackColor(a: Agent) {
  return skillAvatarColor(String(a.id))
}

function descText(a: Agent) {
  return a.description?.trim() || '暂无简介'
}
</script>

<template>
  <aside class="panel" aria-label="工作目录与智能体">
    <!-- 顶部：工作目录下拉选择 -->
    <div class="ws-head">
      <div ref="dropdownWrap" class="ws-dropdown">
        <div class="ws-trigger">
          <button
            type="button"
            class="ws-folder"
            :disabled="!currentWorkspace"
            :title="currentWorkspace ? '打开工作目录' : '未选择工作目录'"
            aria-label="打开工作目录"
            @click="dropdownOpen = false; emit('openWorkspaceFolder', currentWorkspace)"
          >
            <IconFolder />
          </button>
          <button
            type="button"
            class="ws-pick"
            :title="currentWorkspace || '未选择工作目录'"
            :aria-expanded="dropdownOpen"
            @click="dropdownOpen = !dropdownOpen"
          >
            <span class="ws-label">{{ currentLabel || '选择工作目录' }}</span>
            <IconChevronDown class="chevron" :class="{ open: dropdownOpen }" />
          </button>
        </div>
        <div v-if="dropdownOpen" class="ws-pop" role="listbox" aria-label="工作目录">
          <button
            v-for="p in workspaces"
            :key="p"
            type="button"
            role="option"
            class="ws-opt"
            :class="{ on: samePath(p, currentWorkspace) }"
            :title="p"
            @click="pickWorkspace(p)"
          >
            <IconFolder />
            <span class="ws-opt-label">{{ labelOf(p) }}</span>
          </button>
          <p v-if="!workspaces.length" class="ws-empty">尚未添加工作目录</p>
          <button type="button" class="ws-add" @click="dropdownOpen = false; emit('pickWorkspace')">
            <IconPlus />
            新建工作目录
          </button>
        </div>
      </div>
    </div>

    <!-- 下方：已安装智能体 -->
    <div class="agent-head">
      <span class="agent-title">智能体</span>
    </div>
    <div class="agent-list">
      <div v-for="a in agents" :key="a.id" class="agent-item" :class="{ on: a.id === activeId }">
        <button type="button" class="agent-main" :title="a.name" @click="emit('selectAgent', a.id)">
          <span class="avatar-wrap">
            <img v-if="a.avatar" :src="a.avatar" class="avatar" alt="" />
            <span v-else class="avatar fallback" :style="{ background: fallbackColor(a) }">
              {{ skillInitial(a.name) }}
            </span>
          </span>
          <span class="meta">
            <strong class="name">{{ a.name }}</strong>
            <span class="sign">{{ descText(a) }}</span>
          </span>
        </button>
      </div>
      <p v-if="!agents.length" class="agent-empty">尚未安装智能体</p>
    </div>
  </aside>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  gap: 8px;
  padding: 12px 12px 16px;
  border-right: 1px solid var(--border);
  background: var(--bg-chrome-soft);
}

.ws-head {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.ws-dropdown {
  position: relative;
  flex: 1;
  width: 100%;
  min-width: 0;
}

.ws-trigger {
  display: flex;
  align-items: stretch;
  width: 100%;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  overflow: hidden;
  transition: background 0.12s, border-color 0.12s;
}

.ws-trigger:hover {
  background: var(--surface-hover);
  border-color: var(--border-strong, var(--border));
}

.ws-folder,
.ws-pick {
  display: flex;
  align-items: center;
  border: 0;
  background: transparent;
  color: var(--text);
  font: inherit;
  cursor: pointer;
}

.ws-folder {
  flex-shrink: 0;
  padding: 8px 8px 8px 10px;
  color: var(--text-muted);
}

.ws-folder:hover:not(:disabled) {
  color: var(--accent);
}

.ws-folder:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.ws-folder svg {
  width: 14px;
  height: 14px;
}

.ws-pick {
  flex: 1;
  min-width: 0;
  gap: 6px;
  padding: 8px 10px 8px 2px;
}

.ws-pick svg {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  color: var(--text-muted);
}

.ws-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
  text-align: left;
}

.chevron {
  transition: transform 0.16s ease;
}

.chevron.open {
  transform: rotate(180deg);
}

.ws-pop {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 6px);
  z-index: 30;
  padding: 4px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
  max-height: 280px;
  overflow: auto;
}

.ws-opt {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.ws-opt svg {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  color: var(--text-muted);
}

.ws-opt:hover {
  background: var(--surface-hover);
}

.ws-opt.on {
  background: var(--accent-soft);
}

.ws-opt.on .ws-opt-label {
  color: var(--accent);
}

.ws-opt-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.ws-empty {
  margin: 0;
  padding: 10px 12px;
  color: var(--text-dim);
  font-size: 12px;
}

.ws-add {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  margin-top: 2px;
  padding: 8px 10px;
  border: 0;
  border-top: 1px solid var(--border);
  border-radius: 0 0 var(--radius-sm) var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.ws-add:hover {
  color: var(--accent);
  background: var(--surface-hover);
}

.ws-add svg {
  width: 13px;
  height: 13px;
}

.agent-head {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  padding: 0 4px;
}

.agent-title {
  flex: 1;
  min-width: 0;
  color: var(--text-dim);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.head-act {
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  flex-shrink: 0;
  cursor: pointer;
}

.head-act svg {
  width: 14px;
  height: 14px;
}

.head-act:hover {
  color: var(--accent);
  background: var(--surface-hover);
}

.agent-list {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.agent-item {
  display: flex;
  align-items: center;
  gap: 2px;
  border-radius: var(--radius-sm);
  transition: background 0.12s;
}

.agent-item:hover {
  background: var(--surface-hover);
}

.agent-item.on {
  background: var(--accent-soft);
}

.agent-main {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
  padding: 8px;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.avatar-wrap {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
}

.avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
  display: block;
  background: var(--surface);
}

.avatar.fallback {
  display: grid;
  place-items: center;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
}

.meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
  color: var(--text);
}

.agent-item.on .name {
  color: var(--accent);
}

.sign {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  line-height: 16px;
  color: var(--text-dim);
}

.agent-item .head-act {
  opacity: 0;
  transition: opacity 0.12s;
}

.agent-item:hover .head-act {
  opacity: 1;
}

.agent-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  margin: 0;
  padding: 24px 12px;
  color: var(--text-dim);
  font-size: 12px;
  line-height: 18px;
  text-align: center;
}

.empty-add {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: 1px dashed var(--border-strong, var(--border));
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--text-muted);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.empty-add:hover {
  color: var(--accent);
  border-color: var(--accent);
}

.empty-add svg {
  width: 13px;
  height: 13px;
}
</style>
