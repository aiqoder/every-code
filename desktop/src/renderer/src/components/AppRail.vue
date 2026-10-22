<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { adeApi, bridge } from '../api'
import { messageOf, useSessionStore } from '../stores/session'
import { toast } from '../composables/toast'
import { createPage } from '../surfaces/create'
import Dialog from './Dialog.vue'
import FormField from './FormField.vue'
import TextArea from './TextArea.vue'
import TextField from './TextField.vue'
import type { Project } from '../../../shared/types'

const session = useSessionStore()
const router = useRouter()
const route = useRoute()

const showProjects = ref(true)
const menu = ref<{ project: Project; x: number; y: number } | null>(null)

const editing = ref<Project | null>(null)
const editBusy = ref(false)
const editError = ref('')
const editForm = reactive({
  name: '',
  description: '',
  localDir: ''
})

const removing = ref<Project | null>(null)
const removeBusy = ref(false)

function createProject(): void {
  router.push('/project?new=1')
}

function pickProject(id: number): void {
  session.selectProject(id)
  void session.loadTasks(id)
  router.push('/project')
}

function openMenu(event: MouseEvent, project: Project): void {
  const fromMore = event.currentTarget instanceof HTMLElement && event.currentTarget.classList.contains('more')
  if (fromMore && menu.value?.project.id === project.id) {
    closeMenu()
    return
  }
  const width = 176
  const height = 148
  let x = event.clientX
  let y = event.clientY
  if (fromMore && event.currentTarget instanceof HTMLElement) {
    const rect = event.currentTarget.getBoundingClientRect()
    x = rect.right - width
    y = rect.bottom + 4
  }
  const maxX = Math.max(8, window.innerWidth - width - 8)
  const maxY = Math.max(8, window.innerHeight - height - 8)
  if (y > maxY) y = Math.max(8, y - height - 28)
  menu.value = {
    project,
    x: Math.min(Math.max(8, x), maxX),
    y: Math.min(Math.max(8, y), maxY)
  }
}

function closeMenu(): void {
  menu.value = null
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') closeMenu()
}

function onPointerDown(event: MouseEvent): void {
  const target = event.target
  if (target instanceof Element && target.closest('.more')) return
  closeMenu()
}

watch(menu, (value) => {
  if (value) window.addEventListener('mousedown', onPointerDown)
  else window.removeEventListener('mousedown', onPointerDown)
})

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('mousedown', onPointerDown)
})

function beginEdit(project: Project): void {
  closeMenu()
  editing.value = project
  editForm.name = project.name
  editForm.description = project.description || ''
  editForm.localDir = project.localDir || ''
  editError.value = ''
}

async function pickEditDir(): Promise<void> {
  const dir = await bridge.pickDirectory()
  if (dir) editForm.localDir = dir
}

async function saveEdit(): Promise<void> {
  const project = editing.value
  if (!project || editBusy.value) return
  const name = editForm.name.trim()
  const description = editForm.description.trim()
  const localDir = editForm.localDir.trim()
  if (!name) {
    editError.value = '项目名称不能为空'
    return
  }
  if (!localDir) {
    editError.value = '请选择本地目录'
    return
  }
  editBusy.value = true
  editError.value = ''
  try {
    if (localDir !== project.localDir) {
      const status = await bridge.gitStatus(localDir)
      if (!status.isRepo || !status.ready) {
        const init = await bridge.gitInit(localDir)
        if (!init.ok) {
          editError.value = `初始化 Git 仓库失败：${init.output}`
          return
        }
        toast.show(`已初始化仓库：${init.output.split('\n').filter(Boolean).pop() ?? '完成'}`)
      }
    }
    const updated = await adeApi.updateProject(project.id, { name, description, localDir })
    const idx = session.projects.findIndex((item) => item.id === updated.id)
    if (idx >= 0) session.projects[idx] = updated
    editing.value = null
    toast.success(`已更新项目「${updated.name}」`)
  } catch (err) {
    editError.value = messageOf(err)
  } finally {
    editBusy.value = false
  }
}

function checkEnvironment(project: Project): void {
  closeMenu()
  if (!project.localDir.trim()) {
    toast.error('这个项目没有本地目录')
    return
  }
  if (createPage.envBusy) {
    toast.show('环境检测还在进行')
    return
  }
  session.selectProject(project.id)
  createPage.envBusy = true
  createPage.envChatNonce += 1
  if (route.path !== '/project') void router.push('/project')
}

async function openFolder(project: Project): Promise<void> {
  closeMenu()
  const dir = project.localDir.trim()
  if (!dir) {
    toast.error('这个项目没有本地目录')
    return
  }
  try {
    await bridge.openExternal(dir)
  } catch (err) {
    toast.error(messageOf(err))
  }
}

function askRemove(project: Project): void {
  closeMenu()
  removing.value = project
}

async function confirmRemove(): Promise<void> {
  const project = removing.value
  if (!project || removeBusy.value) return
  removeBusy.value = true
  try {
    const wasCurrent = session.currentProjectId === project.id
    await adeApi.deleteProject(project.id)
    const nextTasks = { ...session.tasks }
    delete nextTasks[project.id]
    session.tasks = nextTasks
    await session.loadProjects()
    if (wasCurrent) {
      const next = session.projects[0]
      session.selectProject(next?.id ?? 0)
      if (next) await session.loadTasks(next.id)
      if (route.path.startsWith('/tasks/')) router.push('/project')
    }
    removing.value = null
    toast.success(`已移除项目「${project.name}」`)
  } catch (err) {
    toast.error(messageOf(err))
  } finally {
    removeBusy.value = false
  }
}

function openSettings(): void {
  router.push('/settings')
}

async function logout(): Promise<void> {
  await session.logout()
  toast.show('已退出登录')
  router.replace('/login')
}

const statusDot = computed(() => {
  if (!session.connection.loggedIn) return 'off'
  return session.connection.wsOnline ? 'on' : 'warn'
})
</script>

<template>
  <aside class="rail">
    <div class="brand">
      <span class="mark serif">A</span>
      <div class="col">
        <span class="name serif">ADE</span>
        <span class="tagline">让 Agent 把活干完</span>
      </div>
    </div>

    <div class="section">
      <button class="section-head" @click="showProjects = !showProjects">
        <span class="label">项目</span>
        <span class="chev" :class="{ open: showProjects }">›</span>
      </button>
      <div v-if="showProjects" class="projects">
        <div
          v-for="p in session.projects"
          :key="p.id"
          class="project-row"
          :class="{ active: p.id === session.currentProjectId, open: menu?.project.id === p.id }"
          @contextmenu.prevent="openMenu($event, p)"
        >
          <button class="project" type="button" @click="pickProject(p.id)">
            <span class="dot" :class="{ busy: p.runningTask > 0 }" />
            <span class="pname ellipsis">{{ p.name }}</span>
            <span class="pcount">{{ p.taskCount }}</span>
          </button>
          <button
            class="more"
            type="button"
            title="项目操作"
            aria-haspopup="menu"
            :aria-expanded="menu?.project.id === p.id"
            @click.stop="openMenu($event, p)"
          >
            ⋯
          </button>
        </div>
        <p v-if="!session.projects.length" class="rail-empty">还没有项目</p>
        <button class="btn sm new-project" @click="createProject">
          ＋ 新建项目
        </button>
      </div>
    </div>

    <div class="spacer" />

    <div class="footer">
      <button class="foot-item" :class="{ on: route.path.startsWith('/models') }" @click="router.push('/models')">
        <span class="fi-label">模型</span>
      </button>
      <button class="foot-item" :class="{ on: route.path.startsWith('/hosts') }" @click="router.push('/hosts')">
        <span class="fi-label">主机</span>
      </button>
      <button class="foot-item" @click="openSettings">
        <span class="fi-label">服务连接</span>
        <span class="fi-meta">
          <span class="conn" :class="statusDot" />
          {{ session.connection.wsOnline ? '在线' : session.connection.loggedIn ? '待连' : '未登录' }}
        </span>
      </button>
      <div class="foot-actions">
        <button class="btn ghost sm" @click="openSettings">设置</button>
        <button v-if="session.connection.loggedIn" class="btn ghost sm" @click="logout">退出</button>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="menu"
        class="menu"
        role="menu"
        :style="{ top: `${menu.y}px`, left: `${menu.x}px` }"
        @mousedown.stop
      >
        <button type="button" role="menuitem" @click="beginEdit(menu.project)">编辑</button>
        <button type="button" role="menuitem" @click="openFolder(menu.project)">打开所在文件夹</button>
        <button type="button" role="menuitem" :disabled="createPage.envBusy" @click="checkEnvironment(menu.project)">
          {{ createPage.envBusy ? '检测中…' : '检测开发环境' }}
        </button>
        <button type="button" role="menuitem" class="danger" @click="askRemove(menu.project)">移除</button>
      </div>
    </Teleport>

    <Dialog
      :open="!!editing"
      title="编辑项目"
      subtitle="修改名称、介绍和本地目录。"
      width="520px"
      :persistent="editBusy"
      @close="editing = null"
    >
      <div class="form">
        <FormField label="名称">
          <TextField v-model="editForm.name" />
        </FormField>
        <FormField label="详细介绍">
          <TextArea v-model="editForm.description" :rows="5" />
        </FormField>
        <FormField label="本地目录">
          <div class="dir-row">
            <span class="dir ellipsis" :title="editForm.localDir">{{ editForm.localDir || '未设置' }}</span>
            <button class="btn sm" type="button" :disabled="editBusy" @click="pickEditDir">更换</button>
          </div>
        </FormField>
      </div>
      <p v-if="editError" class="err">{{ editError }}</p>
      <template #footer>
        <button class="btn ghost" type="button" :disabled="editBusy" @click="editing = null">取消</button>
        <div class="spacer" />
        <button class="btn primary" type="button" :disabled="editBusy" @click="saveEdit">
          {{ editBusy ? '保存中…' : '保存' }}
        </button>
      </template>
    </Dialog>

    <Dialog
      :open="!!removing"
      title="移除项目"
      subtitle="任务记录会删除，本机目录保留。"
      width="420px"
      :persistent="removeBusy"
      @close="removing = null"
    >
      <p v-if="removing" class="confirm-copy">
        移除「{{ removing.name }}」后，这个项目下的任务记录会删除。文件夹仍留在原处。
      </p>
      <p v-if="removing?.localDir" class="dir confirm-dir ellipsis" :title="removing.localDir">{{ removing.localDir }}</p>
      <template #footer>
        <button class="btn ghost" type="button" :disabled="removeBusy" @click="removing = null">取消</button>
        <div class="spacer" />
        <button class="btn danger" type="button" :disabled="removeBusy" @click="confirmRemove">
          {{ removeBusy ? '移除中…' : '移除' }}
        </button>
      </template>
    </Dialog>
  </aside>
</template>

<style scoped>
.rail {
  width: var(--rail-w);
  flex: none;
  display: flex;
  flex-direction: column;
  background: var(--ink-800);
  border-right: 1px solid var(--line-soft);
  padding: 0 0 10px;
  overflow: hidden;
}

.brand {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 22px 18px 18px;
  -webkit-app-region: drag;
}
.mark {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: var(--r-sm);
  border: 1px solid var(--line);
  background: var(--btn-default-bg);
  color: var(--brass);
  font-size: 15px;
  font-weight: 600;
  line-height: 1;
}
.name {
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--text-strong);
}
.tagline {
  font-size: 12px;
  color: var(--text-faint);
  font-weight: 400;
}

.section {
  padding: 10px 12px 4px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.section-head {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 2px 6px 6px;
}
.label {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: normal;
  color: var(--text-dim);
  text-transform: none;
  padding: 0 6px;
}
.chev {
  color: var(--text-faint);
  font-size: 12px;
  transition: transform 0.16s ease;
  margin-left: auto;
}
.chev.open {
  transform: rotate(90deg);
}

.projects {
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.project-row {
  position: relative;
}
.project-row:hover .project,
.project-row.open .project {
  background: var(--ink-750);
  color: var(--text-strong);
}
.project-row.active .project {
  background: var(--ink-700);
  color: var(--text-strong);
}
.project-row.active .pname {
  font-weight: 500;
}
.project {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 30px;
  padding: 0 8px;
  border-radius: var(--r-sm);
  color: var(--text-dim);
  font-size: 14px;
  font-weight: 400;
  text-align: left;
  transition: background 0.12s ease, color 0.12s ease;
}
.more {
  position: absolute;
  z-index: 1;
  top: 6px;
  right: 4px;
  width: 18px;
  height: 18px;
  border-radius: var(--r-sm);
  color: var(--text-faint);
  font-size: 14px;
  line-height: 1;
  opacity: 0;
  pointer-events: none;
}
.project-row:hover .more,
.project-row.open .more,
.more:focus-visible {
  opacity: 1;
  pointer-events: auto;
  color: var(--text);
  background: var(--ink-650);
}
.project-row:hover .pcount,
.project-row.open .pcount {
  opacity: 0;
}
.dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--ink-500);
  flex: none;
}
.dot.busy {
  background: var(--brass);
  box-shadow: 0 0 6px rgba(200, 160, 90, 0.6);
}
.pname {
  flex: 1;
}
.pcount {
  font-size: 10.5px;
  color: var(--text-faint);
  font-family: var(--font-mono);
}
.new-project {
  margin: 6px 4px 0;
  justify-content: flex-start;
  color: var(--text-dim);
  border-color: var(--line-soft);
  background: transparent;
}
.new-project:hover {
  color: var(--text-strong);
  border-color: var(--line-strong);
  background: var(--btn-default-hover);
}
.rail-empty {
  font-size: 12px;
  color: var(--text-faint);
  padding: 4px 10px 6px;
}

.stations {
  margin-top: 10px;
  border-top: 1px solid var(--line-soft);
  padding-top: 12px;
}
.station {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 7px 8px 7px 0;
  border-radius: var(--r-sm);
  text-align: left;
  color: var(--text-dim);
  transition: background 0.12s ease;
}
.station:hover {
  background: var(--ink-750);
}
.station.active {
  background: var(--ink-700);
}
.rail-line {
  width: 2px;
  height: 24px;
  border-radius: 2px;
  background: var(--ink-500);
  flex: none;
  transition: background 0.15s ease;
}
.station.active .rail-line.brass {
  background: var(--brass);
}
.rail-line.moss {
  background: var(--moss-dim);
}
.rail-line.amber {
  background: var(--amber);
  opacity: 0.6;
}
.rail-line.clay {
  background: var(--clay);
  opacity: 0.6;
}
.station.active .rail-line {
  opacity: 1;
}
.st-text {
  gap: 1px;
  min-width: 0;
  flex: 1;
}
.st-name {
  font-size: 13px;
  color: var(--text);
}
.station.active .st-name {
  color: var(--text-strong);
  font-weight: 500;
}
.st-hint {
  font-size: 10.5px;
  color: var(--text-faint);
}
.st-count {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-faint);
  padding: 0 5px;
  border-radius: 999px;
  background: var(--ink-850);
}
.st-count.brass {
  color: var(--brass);
}
.st-count.moss {
  color: var(--moss);
}
.st-count.amber {
  color: var(--amber);
}

.footer {
  border-top: 1px solid var(--line-soft);
  padding: 8px 12px 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.foot-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  border-radius: var(--r-sm);
  color: var(--text-dim);
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
}
.foot-item:hover,
.foot-item.on {
  background: var(--ink-750);
  color: var(--text-strong);
}
.fi-meta {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  color: var(--text-faint);
  font-family: var(--font-mono);
}
.conn {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--ink-500);
}
.conn.on {
  background: var(--moss);
  box-shadow: 0 0 6px rgba(127, 156, 106, 0.7);
}
.conn.warn {
  background: var(--amber);
}
.foot-actions {
  display: flex;
  gap: 4px;
  padding: 2px 4px 0;
}

.menu {
  position: fixed;
  z-index: 40;
  min-width: 168px;
  display: flex;
  flex-direction: column;
  padding: 4px;
  background: var(--paper-700);
  border: 1px solid var(--line-strong);
  border-radius: var(--r-md);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.28);
}
.menu button {
  text-align: left;
  padding: 6px 9px;
  border-radius: var(--r-sm);
  font-size: 13px;
  color: var(--text);
}
.menu button:hover {
  background: var(--ink-750);
  color: var(--text-strong);
}
.menu button:disabled {
  opacity: 0.45;
}
.menu button.danger {
  color: #d98d7d;
}
.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.dir-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.dir {
  flex: 1;
  min-width: 0;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-dim);
  padding: 6px 8px;
  border: 1px solid var(--line-soft);
  border-radius: var(--r-sm);
  background: var(--ink-850);
}
.confirm-copy {
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
}
.confirm-dir {
  margin-top: 10px;
}
.spacer {
  flex: 1;
}
.err {
  margin-top: 12px;
  font-size: 12px;
  color: #d98d7d;
  padding: 7px 10px;
  border-radius: var(--r-sm);
  background: rgba(180, 91, 74, 0.12);
  border: 1px solid var(--rust-dim);
}
</style>
