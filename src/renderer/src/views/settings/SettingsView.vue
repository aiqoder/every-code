<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NButton, NModal, NSelect, useMessage } from 'naive-ui'
import ThemeSwitch from '@/components/ThemeSwitch.vue'
import { listAgents, type AgentItem } from '@/api/catalog'
import { getAgent, skillsToAttach, toAgentSaveBody, updateAgent } from '@/api/agents'
import { useSessionStore } from '@/stores/sessionStore'
import type { LoopSkillInfo, LoopSkillSource, LoopToolInfo, LoopToolSource } from '@shared/loop'

const localToolNote =
  '本地工具和自定义 .js 工具由这台电脑执行，服务端智能体只接收 HTTP 工具，这次没有写入工具'

const sourceLabel: Record<LoopSkillSource, string> = {
  claude: 'Claude',
  project: '项目',
  agents: 'Agents',
  package: '插件'
}

const toolSourceLabel: Record<LoopToolSource, string> = {
  builtin: '内置',
  user: '自定义',
  package: '插件'
}

type SettingsTab = 'general' | 'agent'

const tabs: { id: SettingsTab; label: string }[] = [
  { id: 'general', label: '通用' },
  { id: 'agent', label: '智能体' }
]

const session = useSessionStore()
const message = useMessage()

const tab = ref<SettingsTab>('general')
const loading = ref(false)
const inventoryError = ref('')
const tools = ref<LoopToolInfo[]>([])
const skills = ref<LoopSkillInfo[]>([])
const toolNotes = ref<string[]>([])
const toolsDir = ref('')
const checkedTools = ref<string[]>([])
const checkedSkills = ref<string[]>([])

const modalOpen = ref(false)
const submitting = ref(false)
const agentsLoading = ref(false)
const agentsError = ref('')
const agentChoices = ref<AgentItem[]>([])
const pickedAgentId = ref<number | null>(null)

const registerDisabled = computed(
  () =>
    !session.loggedIn ||
    submitting.value ||
    modalOpen.value ||
    (checkedTools.value.length === 0 && checkedSkills.value.length === 0)
)

const agentOptions = computed(() =>
  agentChoices.value.map((item) => ({
    label: item.name,
    value: item.id
  }))
)

const canConfirm = computed(
  () => pickedAgentId.value != null && !agentsLoading.value && !agentsError.value
)

onMounted(() => {
  void loadInventory()
})

async function loadInventory(): Promise<void> {
  loading.value = true
  inventoryError.value = ''
  try {
    if (!window.api?.loop?.listInventory) {
      throw new Error('本地工具清单只能在桌面端读取')
    }
    const inventory = await window.api.loop.listInventory()
    tools.value = Array.isArray(inventory?.tools) ? inventory.tools : []
    skills.value = Array.isArray(inventory?.skills) ? inventory.skills : []
    toolNotes.value = Array.isArray(inventory?.toolNotes) ? inventory.toolNotes : []
    toolsDir.value = inventory?.toolsDir || ''
  } catch (err) {
    tools.value = []
    skills.value = []
    toolNotes.value = []
    toolsDir.value = ''
    inventoryError.value = errorText(err)
  } finally {
    loading.value = false
  }
}

function onToolChange(name: string, event: Event): void {
  const input = event.target
  if (!(input instanceof HTMLInputElement)) return
  checkedTools.value = toggle(checkedTools.value, name, input.checked)
}

function onSkillChange(id: string, event: Event): void {
  const input = event.target
  if (!(input instanceof HTMLInputElement)) return
  checkedSkills.value = toggle(checkedSkills.value, id, input.checked)
}

function toggle(list: string[], id: string, checked: boolean): string[] {
  if (checked) return list.includes(id) ? list : [...list, id]
  return list.filter((item) => item !== id)
}

async function openRegister(): Promise<void> {
  if (registerDisabled.value) return
  modalOpen.value = true
  pickedAgentId.value = null
  agentsError.value = ''
  agentsLoading.value = true
  try {
    agentChoices.value = await listAgents()
  } catch (err) {
    agentChoices.value = []
    agentsError.value = errorText(err)
  } finally {
    agentsLoading.value = false
  }
}

async function confirmRegister(): Promise<void> {
  const agentId = pickedAgentId.value
  if (agentId == null || submitting.value || !canConfirm.value) return
  const selectedSkills = skills.value.filter((skill) => checkedSkills.value.includes(skill.id))
  const toolCount = checkedTools.value.length
  submitting.value = true
  try {
    if (selectedSkills.length === 0) {
      message.info(localToolNote, { duration: 6000 })
      modalOpen.value = false
      return
    }
    const agent = await getAgent(agentId)
    const extra = skillsToAttach(agent, selectedSkills)
    if (extra.length === 0) {
      const skipped = `所选技能已在「${agent.name}」中`
      message.info(toolCount > 0 ? `${skipped}。${localToolNote}` : skipped, { duration: 6000 })
      modalOpen.value = false
      return
    }
    await updateAgent(agentId, toAgentSaveBody(agent, extra))
    const skippedCount = selectedSkills.length - extra.length
    let text = `已把 ${extra.length} 个技能注册到「${agent.name}」`
    if (skippedCount > 0) text += `，已跳过 ${skippedCount} 个同名或同目录技能`
    if (toolCount > 0) text += `。${localToolNote}`
    message.success(text, { duration: 6000 })
    checkedSkills.value = []
    modalOpen.value = false
  } catch (err) {
    message.error(errorText(err))
  } finally {
    submitting.value = false
  }
}

function errorText(err: unknown): string {
  if (err instanceof Error && err.message) return err.message
  return String(err)
}
</script>

<template>
  <div
    class="relative flex h-full min-h-0 w-full flex-1 flex-col overflow-hidden text-[var(--ec-fg)]"
  >
    <div class="flex shrink-0 gap-1 border-b border-[var(--ec-border)] px-6" role="tablist">
      <button
        v-for="item in tabs"
        :key="item.id"
        type="button"
        role="tab"
        class="border-b-2 px-3 py-3 text-sm"
        :class="
          tab === item.id
            ? 'border-[var(--ec-accent)] font-medium text-[var(--ec-fg)]'
            : 'border-transparent text-[var(--ec-fg-muted)]'
        "
        :aria-selected="tab === item.id"
        @click="tab = item.id"
      >
        {{ item.label }}
      </button>
    </div>

    <div class="min-h-0 flex-1 overflow-auto" :class="tab === 'agent' ? 'pb-24' : ''">
      <div v-if="tab === 'general'" class="px-6 py-6">
        <div
          class="flex max-w-xl items-center justify-between gap-4 rounded-md border border-[var(--ec-border)] ec-skin-raised px-4 py-3"
        >
          <span class="text-sm text-[var(--ec-fg)]">外观</span>
          <ThemeSwitch />
        </div>
      </div>

      <div v-else class="flex flex-col gap-8 px-6 py-6">
        <p
          v-if="!session.loggedIn"
          class="rounded-md border border-[var(--ec-border)] ec-skin-raised px-4 py-3 text-sm leading-6 text-[var(--ec-fg)]"
        >
          请先在聊天页登录，再把本地技能注册到远程智能体。
        </p>
        <p v-if="loading" class="text-sm text-[var(--ec-fg-muted)]">正在读取本地工具和技能…</p>
        <p v-else-if="inventoryError" class="text-sm leading-6 text-[var(--ec-danger)]">
          {{ inventoryError }}
        </p>
        <template v-else>
          <section>
            <h2 class="text-sm font-medium">本地工具</h2>
            <p v-if="toolsDir" class="mt-1 text-xs break-all text-[var(--ec-fg-muted)]">
              自定义工具目录：{{ toolsDir }}。只加载直接放在目录里的 .js 文件。
            </p>
            <p
              v-for="note in toolNotes"
              :key="note"
              class="mt-2 text-sm leading-6 text-[var(--ec-attention)]"
            >
              {{ note }}
            </p>
            <p
              v-if="tools.length === 0"
              class="mt-3 rounded-md border border-dashed border-[var(--ec-border)] px-3 py-6 text-center text-sm text-[var(--ec-fg-muted)]"
            >
              暂无本地工具
            </p>
            <div v-else class="mt-3 flex flex-col gap-2">
              <label
                v-for="tool in tools"
                :key="tool.name"
                class="flex cursor-pointer items-start gap-3 rounded-md border border-[var(--ec-border)] ec-skin-raised px-3 py-2.5"
              >
                <input
                  type="checkbox"
                  class="inventory-check mt-0.5 h-4 w-4 shrink-0"
                  :checked="checkedTools.includes(tool.name)"
                  @change="onToolChange(tool.name, $event)"
                />
                <span class="min-w-0">
                  <span class="flex flex-wrap items-baseline gap-2">
                    <span class="text-sm text-[var(--ec-fg)]">{{ tool.name }}</span>
                    <span class="text-xs text-[var(--ec-fg-muted)]">{{
                      toolSourceLabel[tool.source]
                    }}</span>
                  </span>
                  <span
                    v-if="tool.description"
                    class="mt-0.5 block text-xs leading-5 break-words text-[var(--ec-fg-muted)]"
                  >
                    {{ tool.description }}
                  </span>
                </span>
              </label>
            </div>
          </section>

          <section>
            <h2 class="text-sm font-medium">本地技能</h2>
            <p
              v-if="skills.length === 0"
              class="mt-3 rounded-md border border-dashed border-[var(--ec-border)] px-3 py-6 text-center text-sm text-[var(--ec-fg-muted)]"
            >
              暂无本地技能
            </p>
            <div v-else class="mt-3 flex flex-col gap-2">
              <label
                v-for="skill in skills"
                :key="skill.id"
                class="flex cursor-pointer items-start gap-3 rounded-md border border-[var(--ec-border)] ec-skin-raised px-3 py-2.5"
              >
                <input
                  type="checkbox"
                  class="inventory-check mt-0.5 h-4 w-4 shrink-0"
                  :checked="checkedSkills.includes(skill.id)"
                  @change="onSkillChange(skill.id, $event)"
                />
                <span class="min-w-0">
                  <span class="flex flex-wrap items-baseline gap-2">
                    <span class="text-sm text-[var(--ec-fg)]">{{ skill.name }}</span>
                    <span class="text-xs text-[var(--ec-fg-muted)]">{{
                      sourceLabel[skill.source]
                    }}</span>
                  </span>
                  <span
                    v-if="skill.description"
                    class="mt-0.5 block text-xs leading-5 break-words text-[var(--ec-fg-muted)]"
                  >
                    {{ skill.description }}
                  </span>
                </span>
              </label>
            </div>
          </section>
        </template>
      </div>
    </div>

    <button
      v-if="tab === 'agent'"
      type="button"
      class="absolute right-6 bottom-6 z-10 rounded-md bg-[var(--ec-accent)] px-4 py-2 text-sm font-medium text-[var(--ec-on-accent)] disabled:cursor-not-allowed disabled:opacity-40"
      :disabled="registerDisabled"
      @click="openRegister"
    >
      注册
    </button>

    <n-modal
      v-model:show="modalOpen"
      preset="card"
      title="注册到智能体"
      :bordered="false"
      :mask-closable="!submitting"
      :close-on-esc="!submitting"
      :style="{ width: '28rem', maxWidth: 'calc(100% - 2rem)' }"
    >
      <p v-if="agentsError" class="text-sm leading-6 text-[var(--ec-danger)]">{{ agentsError }}</p>
      <p
        v-else-if="!agentsLoading && agentChoices.length === 0"
        class="text-sm text-[var(--ec-fg-muted)]"
      >
        还没有智能体
      </p>
      <n-select
        v-else
        v-model:value="pickedAgentId"
        :options="agentOptions"
        :loading="agentsLoading"
        placeholder="选择智能体"
      />
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button :disabled="submitting" @click="modalOpen = false">取消</n-button>
          <n-button
            type="primary"
            :loading="submitting"
            :disabled="!canConfirm"
            @click="confirmRegister"
          >
            注册到该智能体
          </n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.inventory-check {
  accent-color: var(--ec-accent);
}
</style>
