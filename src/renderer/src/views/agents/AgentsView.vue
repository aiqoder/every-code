<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { NButton, useMessage } from 'naive-ui'
import { listAgents, resolveAgentKey, type AgentItem } from '@/api/catalog'
import { useSessionStore } from '@/stores/sessionStore'
import type { LoopPackage } from '@shared/loop'
import { syncPackageAgent } from '@/views/chat/syncPackage'
import InstallPackagePanel from './components/InstallPackagePanel.vue'
import { syncInstalledPackages } from './syncInstalled'

const session = useSessionStore()
const message = useMessage()
const agents = ref<AgentItem[]>([])
const packages = ref<LoopPackage[]>([])
const toolsDir = ref('')
const toolNotes = ref<string[]>([])
const agentId = ref(localStorage.getItem('everycode.agentId') || '')
const packageBusy = ref(false)
const actionId = ref('')
const errorText = ref('')
const installOpen = ref(false)
const installError = ref('')

const palette = ['#1f6feb', '#3fb950', '#d29922', '#8957e5', '#f85149', '#39c5cf']
const query = ref('')
const focusId = ref('')
const envDraft = ref<Record<string, string>>({})

const linkedIds = computed(
  () =>
    new Set(packages.value.map((item) => item.remoteId).filter((id): id is number => id != null))
)

const shownPackages = computed(() =>
  packages.value.filter((item) => hit(item.name, item.description))
)

const shownAgents = computed(() =>
  agents.value.filter((item) => !linkedIds.value.has(item.id) && hit(item.name, item.description))
)

const focus = computed(() => packages.value.find((item) => item.id === focusId.value) || null)

watch(agentId, (value) => localStorage.setItem('everycode.agentId', value))

watch(
  [agentId, packages, agents],
  () => {
    if (focusId.value && packages.value.some((item) => item.id === focusId.value)) return
    const match = packages.value.find((item) => packageActive(item))
    if (match) focusId.value = match.id
  },
  { immediate: true }
)

watch(
  focus,
  (pkg) => {
    const next: Record<string, string> = {}
    for (const field of pkg?.env || []) next[field.name] = field.value
    envDraft.value = next
  },
  { immediate: true }
)

watch(
  () => session.loggedIn,
  (loggedIn, wasLoggedIn) => {
    if (loggedIn && !wasLoggedIn) void boot()
  }
)

onMounted(() => {
  void boot()
})

function hit(name: string, description: string): boolean {
  const text = query.value.trim().toLowerCase()
  if (!text) return true
  return name.toLowerCase().includes(text) || description.toLowerCase().includes(text)
}

function initial(name: string): string {
  const text = name.trim()
  return text ? text.slice(0, 1).toUpperCase() : '?'
}

function colorOf(name: string): string {
  let index = 0
  for (const char of name) index = (index + char.charCodeAt(0)) % palette.length
  return palette[index]
}

function slugForPackage(item: LoopPackage): string {
  if (item.remoteId == null) return ''
  return agents.value.find((agent) => agent.id === item.remoteId)?.agentId || ''
}

function packageActive(item: LoopPackage): boolean {
  const slug = slugForPackage(item)
  return slug !== '' && slug === agentId.value
}

function columnHint(field: LoopPackage['env'][number]): string {
  return field.items.map((item) => item.label || item.name).join(' | ')
}

async function boot(): Promise<void> {
  errorText.value = ''
  await loadHome()
  agents.value = await listAgents().catch(() => [])
  if (!session.loggedIn) return
  try {
    const synced = await syncInstalledPackages()
    if (synced.error) errorText.value = synced.error
    if (synced.changed) {
      await loadHome()
      agents.value = await listAgents().catch(() => agents.value)
    }
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '同步智能体失败'
  }
  agentId.value = resolveAgentKey(agentId.value, agents.value)
}

async function loadHome(): Promise<void> {
  if (!window.api?.loop?.listAgentHome) return
  const home = await window.api.loop.listAgentHome()
  toolsDir.value = home.toolsDir || ''
  packages.value = Array.isArray(home.packages) ? home.packages : []
  toolNotes.value = Array.isArray(home.toolNotes) ? home.toolNotes : []
}

function replacePackage(pkg: LoopPackage): void {
  const index = packages.value.findIndex((item) => item.id === pkg.id)
  if (index < 0) {
    packages.value = [pkg, ...packages.value]
    return
  }
  packages.value.splice(index, 1, pkg)
}

async function publishPackage(pkg: LoopPackage, select: boolean): Promise<void> {
  if (!session.loggedIn) return
  const remoteId = await syncPackageAgent(pkg)
  if (remoteId !== pkg.remoteId && window.api?.loop?.setPackageRemoteId) {
    const updated = await window.api.loop.setPackageRemoteId(pkg.id, remoteId)
    if (updated) replacePackage(updated)
  }
  agents.value = await listAgents().catch(() => agents.value)
  if (select) {
    const slug = agents.value.find((item) => item.id === remoteId)?.agentId || ''
    if (slug) agentId.value = slug
  }
}

function openInstall(): void {
  installError.value = ''
  installOpen.value = true
}

function closeInstall(): void {
  if (packageBusy.value) return
  installOpen.value = false
}

async function onInstallFiles(files: File[]): Promise<void> {
  const accepted = files.filter((file) => /\.(cpk|agent)$/i.test(file.name))
  if (accepted.length === 0) {
    installError.value = '请选择 .cpk 或 .agent 文件'
    return
  }
  if (!window.api?.files?.pathForFile || !window.api.loop?.installPackage) {
    installError.value = '插件只能在桌面端安装'
    return
  }
  packageBusy.value = true
  installError.value = ''
  try {
    for (const file of accepted) {
      const filePath = window.api.files.pathForFile(file)
      if (!filePath) throw new Error('没有读到插件路径')
      const installed = await window.api.loop.installPackage(filePath)
      if (!installed) continue
      replacePackage(installed)
      focusId.value = installed.id
      if (session.loggedIn) await publishPackage(installed, true)
    }
    installOpen.value = false
  } catch (err) {
    installError.value = err instanceof Error ? err.message : '安装插件失败'
  } finally {
    packageBusy.value = false
  }
}

function onSelectServerAgent(item: AgentItem): void {
  if (!item.agentId) {
    errorText.value = '这个智能体还没有可用的标识'
    return
  }
  agentId.value = item.agentId
  focusId.value = ''
  errorText.value = ''
}

async function onSelectPackage(id: string): Promise<void> {
  const pkg = packages.value.find((item) => item.id === id)
  if (!pkg) return
  focusId.value = id
  if (pkg.remoteId) {
    const slug = slugForPackage(pkg)
    if (!slug) {
      errorText.value = '这个智能体还没有可用的标识'
      return
    }
    agentId.value = slug
    errorText.value = ''
    return
  }
  if (!session.loggedIn) {
    errorText.value = '请先登录，再把这个智能体同步到服务器'
    return
  }
  packageBusy.value = true
  errorText.value = ''
  try {
    await publishPackage(pkg, true)
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '同步智能体失败'
  } finally {
    packageBusy.value = false
  }
}

async function onReregister(id: string): Promise<void> {
  const pkg = packages.value.find((item) => item.id === id)
  if (!pkg || actionId.value || packageBusy.value) return
  if (!session.loggedIn) {
    errorText.value = '请先登录，再把这个智能体同步到服务器'
    return
  }
  actionId.value = id
  errorText.value = ''
  try {
    await publishPackage(pkg, false)
    message.success(`已更新「${pkg.name}」在服务器上的登记`)
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '重新注册失败'
  } finally {
    actionId.value = ''
  }
}

async function onUninstallPackage(id: string): Promise<void> {
  if (!window.api?.loop?.uninstallPackage || actionId.value || packageBusy.value) return
  if (!window.confirm('卸载这个智能体插件？本机文件会删除，服务器上已同步的智能体仍保留。')) return
  actionId.value = id
  errorText.value = ''
  try {
    await window.api.loop.uninstallPackage(id)
    packages.value = packages.value.filter((item) => item.id !== id)
    if (focusId.value === id) focusId.value = ''
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '卸载失败'
  } finally {
    actionId.value = ''
  }
}

async function onSavePackageEnv(): Promise<void> {
  if (!focus.value || !window.api?.loop?.savePackageEnv) return
  packageBusy.value = true
  errorText.value = ''
  try {
    const updated = await window.api.loop.savePackageEnv(focus.value.id, { ...envDraft.value })
    if (updated) replacePackage(updated)
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '保存环境变量失败'
  } finally {
    packageBusy.value = false
  }
}

async function onOpenToolsDir(): Promise<void> {
  if (!window.api?.loop?.openToolsDir) {
    errorText.value = toolsDir.value ? `工具目录：${toolsDir.value}` : '工具目录只能在桌面端打开'
    return
  }
  try {
    await window.api.loop.openToolsDir()
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '打开工具目录失败'
  }
}
</script>

<template>
  <div class="relative flex h-full min-h-0 flex-col text-[var(--ec-fg)]">
    <div class="mx-auto flex min-h-0 w-full max-w-6xl flex-1 flex-col">
      <div class="flex items-center justify-between gap-2 px-6 pt-4">
        <p class="text-sm">智能体</p>
        <n-button size="small" type="primary" :disabled="packageBusy" @click="openInstall">
          安装
        </n-button>
      </div>
      <p
        v-if="errorText"
        class="mx-6 mt-3 rounded-md border border-[color-mix(in_srgb,var(--ec-danger)_35%,transparent)] bg-[color-mix(in_srgb,var(--ec-danger)_12%,transparent)] px-3 py-2 text-sm text-[var(--ec-danger)]"
      >
        {{ errorText }}
      </p>
      <input
        v-model="query"
        type="search"
        placeholder="搜索智能体"
        aria-label="搜索智能体"
        class="mx-6 mt-3 h-8 rounded-md border border-[var(--ec-border)] bg-[var(--ec-canvas)] px-2 text-xs outline-none"
      />
      <div class="mt-3 min-h-0 flex-1 overflow-auto px-6 pb-4">
        <p class="text-xs text-[var(--ec-fg-muted)]">已安装</p>
        <div class="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <article
            v-for="item in shownPackages"
            :key="item.id"
            class="flex flex-col rounded-xl border px-3 py-3"
            :class="
              packageActive(item)
                ? 'border-[var(--ec-accent)] bg-[color-mix(in_srgb,var(--ec-accent)_12%,transparent)]'
                : 'border-[var(--ec-border)] ec-skin-raised'
            "
          >
            <button
              type="button"
              class="flex w-full items-start gap-2 text-left"
              @click="onSelectPackage(item.id)"
            >
              <img
                v-if="item.avatar"
                :src="item.avatar"
                alt=""
                class="h-9 w-9 shrink-0 rounded-full object-cover"
              />
              <span
                v-else
                class="grid h-9 w-9 shrink-0 place-items-center rounded-full text-sm font-medium text-white"
                :style="{ background: colorOf(item.name) }"
              >
                {{ initial(item.name) }}
              </span>
              <span class="min-w-0 flex-1">
                <span class="flex items-baseline gap-2">
                  <span class="truncate text-sm">{{ item.name }}</span>
                  <span class="shrink-0 text-[11px] text-[var(--ec-fg-muted)]">{{
                    item.version
                  }}</span>
                </span>
                <span class="mt-0.5 line-clamp-2 text-xs leading-5 text-[var(--ec-fg-muted)]">
                  {{ item.description || '暂无简介' }}
                </span>
                <span
                  v-if="!item.remoteId"
                  class="mt-0.5 block text-[11px] text-[var(--ec-attention)]"
                >
                  未同步
                </span>
                <span
                  v-else-if="item.env.length && !item.envReady"
                  class="mt-0.5 block text-[11px] text-[var(--ec-attention)]"
                >
                  还有环境变量没填
                </span>
              </span>
            </button>
            <form
              v-if="focus?.id === item.id && item.env.length"
              class="mt-3 border-t border-[var(--ec-border)] pt-3"
              @submit.prevent="onSavePackageEnv"
            >
              <p class="text-xs text-[var(--ec-fg-muted)]">环境变量</p>
              <label v-for="field in item.env" :key="field.name" class="mt-2 block">
                <span class="text-xs">
                  {{ field.label || field.name }}
                  <span v-if="field.required" class="text-[var(--ec-danger)]">*</span>
                </span>
                <textarea
                  v-if="field.type === 'list'"
                  v-model="envDraft[field.name]"
                  rows="3"
                  class="mt-1 w-full resize-y rounded-md border border-[var(--ec-border)] bg-[var(--ec-canvas)] px-2 py-1 text-xs outline-none"
                  :placeholder="columnHint(field)"
                />
                <input
                  v-else
                  v-model="envDraft[field.name]"
                  :type="field.secret ? 'password' : 'text'"
                  class="mt-1 h-8 w-full rounded-md border border-[var(--ec-border)] bg-[var(--ec-canvas)] px-2 text-xs outline-none"
                />
              </label>
              <n-button class="mt-2" size="tiny" attr-type="submit" :loading="packageBusy">
                保存
              </n-button>
            </form>
            <div class="mt-2 flex justify-end gap-1">
              <button
                type="button"
                title="重新注册"
                aria-label="重新注册"
                class="grid h-7 w-7 place-items-center rounded-md text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)] disabled:opacity-40"
                :disabled="packageBusy || actionId !== ''"
                @click="onReregister(item.id)"
              >
                <svg viewBox="0 0 16 16" class="h-3.5 w-3.5" fill="none" aria-hidden="true">
                  <path
                    d="M13.5 8a5.5 5.5 0 0 1-9.4 3.9M2.5 8a5.5 5.5 0 0 1 9.4-3.9"
                    stroke="currentColor"
                    stroke-width="1.4"
                    stroke-linecap="round"
                  />
                  <path
                    d="M13.6 3.2v3.1H10.5M2.4 12.8V9.7h3.1"
                    stroke="currentColor"
                    stroke-width="1.4"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </button>
              <button
                type="button"
                title="卸载"
                aria-label="卸载"
                class="grid h-7 w-7 place-items-center rounded-md text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-danger)] disabled:opacity-40"
                :disabled="packageBusy || actionId !== ''"
                @click="onUninstallPackage(item.id)"
              >
                <svg viewBox="0 0 16 16" class="h-3.5 w-3.5" fill="none" aria-hidden="true">
                  <path
                    d="M3.5 4.5h9M6.5 4.5V3.2h3V4.5M4.5 4.5l.6 8.2h5.8l.6-8.2"
                    stroke="currentColor"
                    stroke-width="1.4"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </button>
            </div>
          </article>
        </div>
        <p v-if="shownPackages.length === 0" class="py-3 text-xs text-[var(--ec-fg-muted)]">
          {{ query.trim() ? '没有匹配的插件' : '还没有安装插件' }}
        </p>

        <p class="pt-5 text-xs text-[var(--ec-fg-muted)]">服务器</p>
        <div class="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <button
            v-for="item in shownAgents"
            :key="item.id"
            type="button"
            class="flex items-start gap-2 rounded-xl border px-3 py-3 text-left"
            :class="
              item.agentId && item.agentId === agentId
                ? 'border-[var(--ec-accent)] bg-[color-mix(in_srgb,var(--ec-accent)_12%,transparent)]'
                : 'border-[var(--ec-border)] ec-skin-raised hover:bg-[color-mix(in_srgb,var(--ec-fg)_6%,transparent)]'
            "
            @click="onSelectServerAgent(item)"
          >
            <span
              class="grid h-9 w-9 shrink-0 place-items-center rounded-full text-sm font-medium text-white"
              :style="{ background: colorOf(item.name) }"
            >
              {{ initial(item.name) }}
            </span>
            <span class="min-w-0">
              <span class="block truncate text-sm">{{ item.name }}</span>
              <span class="mt-0.5 line-clamp-2 text-xs leading-5 text-[var(--ec-fg-muted)]">
                {{ item.description || '暂无简介' }}
              </span>
            </span>
          </button>
        </div>
        <p v-if="shownAgents.length === 0" class="py-3 text-xs text-[var(--ec-fg-muted)]">
          {{
            query.trim()
              ? '没有匹配的智能体'
              : agents.length
                ? '没有其他服务器智能体'
                : '登录后显示服务器上的智能体'
          }}
        </p>

        <p
          v-for="note in toolNotes.slice(0, 3)"
          :key="note"
          class="mt-4 text-xs leading-5 text-[var(--ec-attention)]"
        >
          {{ note }}
        </p>
        <button
          type="button"
          class="mt-3 text-left text-xs text-[var(--ec-fg-muted)]"
          :title="toolsDir"
          @click="onOpenToolsDir"
        >
          自定义工具目录
        </button>
      </div>
    </div>
    <InstallPackagePanel
      :show="installOpen"
      :busy="packageBusy"
      :error="installError"
      @close="closeInstall"
      @files="onInstallFiles"
    />
  </div>
</template>
