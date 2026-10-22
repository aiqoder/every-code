<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import { useShellStore } from '../stores/shell'
import { bridge } from '../api'

const shell = useShellStore()
const host = ref<HTMLDivElement | null>(null)
let term: Terminal | null = null
let fit: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null
let boundTabId = ''
let offData: (() => void) | null = null

const tabMenu = ref('')
const renaming = ref('')
const renameDraft = ref('')

const theme = {
  background: '#101216',
  foreground: '#c9c4b9',
  cursor: '#c8a05a',
  cursorAccent: '#101216',
  selectionBackground: 'rgba(200, 160, 90, 0.24)',
  black: '#1b1f25',
  red: '#b45b4a',
  green: '#7f9c6a',
  yellow: '#c99a4e',
  blue: '#6b8fa8',
  magenta: '#a07ba0',
  cyan: '#7ba39e',
  white: '#c9c4b9',
  brightBlack: '#615e58',
  brightRed: '#d0705d',
  brightGreen: '#94b47f',
  brightYellow: '#e0bd7c',
  brightBlue: '#84a9c2',
  brightMagenta: '#b894b8',
  brightCyan: '#94bcb6',
  brightWhite: '#f0ece3'
}

function ensureTerminal(): void {
  if (term || !host.value) return
  term = new Terminal({
    fontFamily: "'JetBrains Mono', 'SF Mono', Menlo, monospace",
    fontSize: 12.5,
    lineHeight: 1.32,
    letterSpacing: 0.2,
    theme,
    cursorBlink: true,
    scrollback: 8000,
    allowProposedApi: true,
    macOptionIsMeta: true
  })
  fit = new FitAddon()
  term.loadAddon(fit)
  term.loadAddon(new WebLinksAddon())
  term.open(host.value)
  fitIfVisible()

  term.onData((data) => {
    if (boundTabId) bridge.writePty(boundTabId, data)
  })
}

/** 折叠或动画过程中容器高度接近 0，这时 fit 会把行列压扁，shell 不再正常回显。 */
function fitIfVisible(): void {
  if (!fit || !term || !host.value) return
  const rect = host.value.getBoundingClientRect()
  if (rect.width < 40 || rect.height < 20) return
  fit.fit()
  if (boundTabId && term.cols >= 20 && term.rows >= 5) {
    bridge.resizePty(boundTabId, term.cols, term.rows)
  }
}

function attach(tabId: string): void {
  ensureTerminal()
  if (!term || !fit) return
  // 回放期间先解开绑定，避免和实时监听各写一遍。
  boundTabId = ''
  term.reset()
  const buffered = shell.buffers[tabId] ?? ''
  if (buffered) term.write(buffered)
  boundTabId = tabId
  nextTick(() => {
    if (boundTabId !== tabId) return
    fitIfVisible()
    term?.focus()
  })
}

function detach(): void {
  boundTabId = ''
}

async function newTab(): Promise<void> {
  const cwd = shell.activeTab?.cwd
  await shell.open({ title: `终端 ${shell.tabs.length + 1}`, cwd })
}

async function closeTab(id: string): Promise<void> {
  await shell.close(id)
  tabMenu.value = ''
}

function startRename(tab: { id: string; title: string }): void {
  renaming.value = tab.id
  renameDraft.value = tab.title
  tabMenu.value = ''
}

function commitRename(): void {
  if (renaming.value) shell.rename(renaming.value, renameDraft.value)
  renaming.value = ''
}

function toggleMenu(id: string): void {
  tabMenu.value = tabMenu.value === id ? '' : id
}

function clearTab(): void {
  if (!term || !boundTabId) return
  shell.buffers[boundTabId] = ''
  term.clear()
  tabMenu.value = ''
}

function copySelection(): void {
  if (!term) return
  const text = term.getSelection()
  if (text) navigator.clipboard.writeText(text)
  tabMenu.value = ''
}

function onDocClick(e: MouseEvent): void {
  const target = e.target as HTMLElement
  if (!target.closest('.tab-menu-wrap')) tabMenu.value = ''
}

onMounted(async () => {
  shell.subscribe()
  // 主进程的输出只进了 store 缓冲，必须再写进 xterm，否则敲命令没有任何回显。
  offData = bridge.onPtyData(({ id, data }) => {
    if (!term || id !== boundTabId || !data) return
    term.write(data)
  })
  await shell.refresh()
  ensureTerminal()
  if (shell.activeId) attach(shell.activeId)
  document.addEventListener('click', onDocClick)
  resizeObserver = new ResizeObserver(() => {
    fitIfVisible()
  })
  if (host.value) resizeObserver.observe(host.value)
})

onBeforeUnmount(() => {
  offData?.()
  offData = null
  resizeObserver?.disconnect()
  document.removeEventListener('click', onDocClick)
})

watch(
  () => shell.activeId,
  (id) => {
    if (!id) {
      detach()
      term?.reset()
      return
    }
    if (id !== boundTabId) attach(id)
  }
)

watch(
  () => shell.collapsed,
  async (v) => {
    if (!v) {
      await nextTick()
      ensureTerminal()
      if (shell.activeId) attach(shell.activeId)
      else fitIfVisible()
    }
  }
)
</script>

<template>
  <section class="shell-panel" :class="{ collapsed: shell.collapsed }">
    <div class="shell-head">
      <button class="collapse" :title="shell.collapsed ? '展开 Shell' : '收起 Shell'" @click="shell.toggle()">
        <span class="caret" :class="{ up: !shell.collapsed }">▴</span>
        <span class="label">Shell</span>
        <span class="count">{{ shell.tabs.length }}</span>
      </button>

      <div v-if="!shell.collapsed" class="tabs">
        <div v-for="tab in shell.tabs" :key="tab.id" class="tab-menu-wrap">
          <div
            class="tab"
            :class="{ active: tab.id === shell.activeId, dead: tab.exited }"
            @click="shell.setActive(tab.id)"
            @dblclick="startRename(tab)"
          >
            <span class="kind" :class="tab.kind">{{ tab.kind === 'ssh' ? 'SSH' : 'LOCAL' }}</span>
            <input
              v-if="renaming === tab.id"
              v-model="renameDraft"
              class="rename"
              @keydown.enter="commitRename"
              @keydown.esc="renaming = ''"
              @blur="commitRename"
              @click.stop
            />
            <span v-else class="title ellipsis">{{ tab.title }}</span>
            <button class="close" title="关闭" @click.stop="closeTab(tab.id)">✕</button>
          </div>
          <button class="tab-more" title="标签操作" @click.stop="toggleMenu(tab.id)">⋯</button>
          <div v-if="tabMenu === tab.id" class="menu">
            <button @click="startRename(tab)">重命名</button>
            <button @click="clearTab">清屏</button>
            <button @click="copySelection">复制选中</button>
            <button class="danger" @click="closeTab(tab.id)">关闭标签</button>
          </div>
        </div>
        <button class="new-tab" title="新建终端" @click="newTab">＋</button>
      </div>

      <div class="spacer" />
      <span v-if="!shell.collapsed && shell.activeTab" class="cwd ellipsis" :title="shell.activeTab.cwd">
        {{ shell.activeTab.cwd }}
      </span>
      <button v-if="!shell.collapsed" class="btn ghost sm" title="清屏" @click="clearTab">清屏</button>
    </div>

    <div v-show="!shell.collapsed" class="term-wrap">
      <div ref="host" class="term" />
      <div v-if="!shell.tabs.length" class="term-empty">
        <p class="serif">底栏是当前项目共用的终端</p>
        <p class="hint">Agent 的命令、验收输出和你手敲的命令都在这里。切换页面不会销毁会话。</p>
        <button class="btn sm" @click="newTab">新建终端</button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.shell-panel {
  flex: none;
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--line);
  background: var(--ink-900);
  height: v-bind('shell.height + "px"');
  transition: height 0.18s ease;
}
.shell-panel.collapsed {
  height: 34px;
}

.shell-head {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 34px;
  flex: none;
  padding: 0 10px;
  border-bottom: 1px solid var(--line-soft);
  background: var(--ink-850);
}
.collapse {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 4px;
  color: var(--text-dim);
  font-size: 11.5px;
  letter-spacing: 0.04em;
}
.collapse:hover {
  color: var(--text-strong);
}
.caret {
  font-size: 9px;
  transition: transform 0.16s ease;
  color: var(--brass-dim);
}
.caret.up {
  transform: rotate(180deg);
}
.label {
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.count {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-faint);
  padding: 0 5px;
  border-radius: 999px;
  background: var(--ink-800);
}

.tabs {
  display: flex;
  align-items: center;
  gap: 2px;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
  padding: 0 2px;
}
.tabs::-webkit-scrollbar {
  display: none;
}
.tab-menu-wrap {
  position: relative;
  display: flex;
  align-items: center;
}
.tab {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 24px;
  padding: 0 4px 0 8px;
  border-radius: var(--r-sm);
  border: 1px solid transparent;
  color: var(--text-dim);
  font-size: 11.5px;
  max-width: 220px;
  cursor: pointer;
}
.tab:hover {
  background: var(--ink-800);
}
.tab.active {
  background: var(--paper-800);
  border-color: var(--line);
  color: var(--text-strong);
}
.tab.dead {
  opacity: 0.55;
}
.kind {
  font-family: var(--font-mono);
  font-size: 8.5px;
  letter-spacing: 0.06em;
  color: var(--text-faint);
  border: 1px solid var(--line);
  border-radius: 3px;
  padding: 0 3px;
  flex: none;
}
.kind.ssh {
  color: var(--brass-dim);
  border-color: var(--brass-dim);
}
.title {
  max-width: 140px;
}
.rename {
  width: 110px;
  height: 19px;
  padding: 0 5px;
  font-size: 11.5px;
}
.close {
  width: 16px;
  height: 16px;
  border-radius: 3px;
  color: var(--text-faint);
  font-size: 9px;
  line-height: 1;
  flex: none;
  opacity: 0;
  transition: opacity 0.12s ease;
}
.tab:hover .close,
.tab.active .close {
  opacity: 1;
}
.close:hover {
  background: var(--rust-dim);
  color: #f0ece3;
}
.tab-more {
  width: 15px;
  height: 22px;
  color: var(--text-faint);
  font-size: 12px;
  border-radius: 3px;
  opacity: 0;
}
.tab-menu-wrap:hover .tab-more {
  opacity: 1;
}
.menu {
  position: absolute;
  top: 26px;
  left: 0;
  min-width: 132px;
  z-index: 30;
  display: flex;
  flex-direction: column;
  padding: 4px;
  background: var(--paper-700);
  border: 1px solid var(--line-strong);
  border-radius: var(--r-md);
  box-shadow: var(--shadow-soft);
}
.menu button {
  text-align: left;
  padding: 6px 9px;
  border-radius: var(--r-sm);
  font-size: 12px;
  color: var(--text);
}
.menu button:hover {
  background: var(--ink-750);
  color: var(--text-strong);
}
.menu button.danger {
  color: #d98d7d;
}
.new-tab {
  width: 22px;
  height: 22px;
  border-radius: var(--r-sm);
  color: var(--text-faint);
  font-size: 13px;
  flex: none;
}
.new-tab:hover {
  background: var(--ink-800);
  color: var(--brass);
}
.cwd {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-faint);
  max-width: 38%;
  direction: rtl;
  text-align: left;
}

.term-wrap {
  flex: 1;
  position: relative;
  min-height: 0;
  padding: 6px 4px 4px 10px;
}
.term {
  height: 100%;
  width: 100%;
}
.term-empty {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 7px;
  background: var(--ink-900);
  color: var(--text-faint);
  text-align: center;
  padding: 20px;
}
.term-empty .serif {
  font-size: 13px;
  color: var(--text-dim);
}
.term-empty .hint {
  font-size: 11.5px;
  max-width: 42em;
  line-height: 1.7;
}
</style>
