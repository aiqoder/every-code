<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { bridge, setHttpSession } from '../../api'
import { useSessionStore, messageOf } from '../../stores/session'
import { confirm } from '../../composables/confirm'
import { toast } from '../../composables/toast'
import EmptyState from '../../components/EmptyState.vue'
import FormField from '../../components/FormField.vue'
import TextField from '../../components/TextField.vue'
import { useThemeStore, type ThemePreference } from '../../stores/theme'

const session = useSessionStore()
const router = useRouter()
const theme = useThemeStore()

const themeOptions: { value: ThemePreference; label: string }[] = [
  { value: 'system', label: '跟随系统' },
  { value: 'light', label: '亮色' },
  { value: 'dark', label: '暗色' },
]

const saving = ref(false)
const form = reactive({
  baseUrl: '',
  username: '',
  clientId: '',
  clientName: ''
})

const localSkills = computed(() => session.localSkills)

const conn = computed(() => session.connection)

const connLabel = computed(() => {
  if (!conn.value.loggedIn) return '未登录'
  return conn.value.wsOnline ? '执行面在线' : '已登录，执行面待连'
})

const connTone = computed(() => {
  if (!conn.value.loggedIn) return 'off'
  return conn.value.wsOnline ? 'on' : 'warn'
})

onMounted(async () => {
  const settings = await bridge.settings()
  form.baseUrl = settings.profile.baseUrl
  form.username = settings.profile.username
  form.clientId = settings.profile.clientId
  form.clientName = settings.profile.clientName
  await session.loadLocalSkills()
})

async function save(): Promise<void> {
  saving.value = true
  try {
    await bridge.saveSettings({
      baseUrl: form.baseUrl.trim(),
      username: form.username.trim(),
      clientId: form.clientId.trim(),
      clientName: form.clientName.trim()
    })
    const settings = await bridge.settings()
    session.profile = settings.profile
    setHttpSession({ baseUrl: settings.profile.baseUrl, token: settings.token })
    toast.success('已保存，新地址在下次连接时生效')
  } catch (err) {
    toast.error(messageOf(err))
  } finally {
    saving.value = false
  }
}

async function reconnect(): Promise<void> {
  try {
    const state = await bridge.connect()
    session.connection = state
    toast.show(state.wsOnline ? '执行面已连上' : '已尝试连接，执行面仍不在线')
    if (state.loggedIn) await session.refreshAll()
  } catch (err) {
    toast.error(messageOf(err))
  }
}

async function logout(): Promise<void> {
  const ok = await confirm({
    title: '退出登录',
    message: '本机终端会被关闭，服务端数据保留。',
    confirmText: '退出'
  })
  if (!ok) return
  await session.logout()
  toast.show('已退出登录')
  router.replace('/login')
}

async function openSkillsDir(): Promise<void> {
  const dir = await bridge.openSkillsDir()
  toast.show(`已打开 ${dir}`)
}

async function importSkill(): Promise<void> {
  try {
    const res = await bridge.importSkill()
    if (!res) return
    await session.loadLocalSkills()
    toast.success(`已导入技能 ${res.id}`)
  } catch (err) {
    toast.error(messageOf(err))
  }
}

async function openLocal(skill: { path: string }): Promise<void> {
  await bridge.openExternal(skill.path)
}
</script>

<template>
  <div class="settings">
    <header class="page-head">
      <div class="col">
        <span class="eyebrow">设置</span>
        <h1 class="serif">连接与本机技能</h1>
        <p class="lede">
          服务端持有项目与流水线。这台机器保留登录凭证、自定义技能和 SSH 私钥。模型在「模型」页配置，项目目录在新建项目时选择。
        </p>
      </div>
      <div class="spacer" />
      <span class="conn" :class="connTone"><span class="cdot" />{{ connLabel }}</span>
    </header>

    <div class="scroll">
      <section class="block">
        <div class="block-head">
          <h2 class="serif">连接</h2>
          <span class="faint">{{ conn.username || '未登录' }} · {{ conn.baseUrl || '未设置地址' }}</span>
          <div class="spacer" />
          <button class="btn ghost sm" @click="reconnect">重新连接</button>
        </div>

        <div class="card pad">
          <div class="grid2">
            <FormField label="服务地址">
              <TextField v-model="form.baseUrl" placeholder="http://127.0.0.1:8183" spellcheck="false" />
            </FormField>
            <FormField label="账号">
              <TextField v-model="form.username" placeholder="user-feedback 账号" spellcheck="false" />
            </FormField>
            <FormField label="本机 clientId">
              <TextField v-model="form.clientId" spellcheck="false" />
              <span class="tip">项目绑定这个 id 后，服务端才知道把任务派给哪台桌面。</span>
            </FormField>
            <FormField label="客户端名称">
              <TextField v-model="form.clientName" spellcheck="false" />
            </FormField>
          </div>

          <div class="actions">
            <button class="btn primary sm" :disabled="saving" @click="save">
              {{ saving ? '保存中…' : '保存设置' }}
            </button>
            <div class="spacer" />
            <button v-if="conn.loggedIn" class="btn ghost sm danger" @click="logout">退出登录</button>
          </div>
        </div>
      </section>

      <section class="block">
        <div class="block-head">
          <h2 class="serif">外观</h2>
          <span class="faint">亮色 / 暗色 / 跟随系统</span>
        </div>
        <div class="card pad theme-row">
          <button
            v-for="opt in themeOptions"
            :key="opt.value"
            type="button"
            class="btn sm theme-opt"
            :class="{ on: theme.preference === opt.value }"
            @click="theme.setPreference(opt.value)"
          >
            {{ opt.label }}
          </button>
        </div>
      </section>

      <section class="block">
        <div class="block-head">
          <h2 class="serif">本机自定义技能</h2>
          <span class="faint">{{ localSkills.length }} 个 · 只在这台机器生效</span>
          <div class="spacer" />
          <button class="btn ghost sm" @click="openSkillsDir">打开目录</button>
          <button class="btn sm" @click="importSkill">导入文件夹</button>
        </div>

        <EmptyState
          v-if="!localSkills.length"
          compact
          empty-title="还没有自定义技能"
          empty-hint="目录里放 <id>/SKILL.md 即可被 catalog 上报，Agent 会按项目技术绑定对应技能。"
        />

        <div v-else class="local-list">
          <div v-for="s in localSkills" :key="s.id" class="local-row">
            <div class="col lr-main">
              <span class="lr-name">{{ s.name }}</span>
              <span class="lr-meta mono ellipsis">{{ s.id }} · {{ s.path }}</span>
            </div>
            <p v-if="s.description" class="lr-desc ellipsis">{{ s.description }}</p>
            <button class="btn ghost sm" @click="openLocal(s)">打开</button>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.settings {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.page-head {
  display: flex;
  align-items: flex-end;
  gap: 22px;
  padding: 22px 26px 14px;
  border-bottom: 1px solid var(--line-soft);
  -webkit-app-region: drag;
  flex-wrap: wrap;
}
.eyebrow {
  font-size: 10.5px;
  letter-spacing: 0.18em;
  color: var(--brass-dim);
  text-transform: uppercase;
  margin-bottom: 5px;
}
h1 {
  font-size: 21px;
}
.lede {
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-dim);
  max-width: 64em;
  line-height: 1.7;
}
.conn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  border: 1px solid var(--line);
  font-size: 11.5px;
  color: var(--text-faint);
  -webkit-app-region: no-drag;
}
.cdot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}
.conn.on {
  color: var(--moss);
  border-color: var(--moss-dim);
}
.conn.warn {
  color: var(--amber);
  border-color: rgba(201, 154, 78, 0.4);
}
.scroll {
  flex: 1;
  overflow: auto;
  padding: 20px 26px 36px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}
.block {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.block-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}
.block-head h2 {
  font-size: 15.5px;
}
.block-head .faint {
  font-size: 11.5px;
}
.card {
  background: var(--paper-900);
  border: 1px solid var(--line-soft);
  border-radius: var(--r-lg);
}
.pad {
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.theme-row {
  flex-direction: row;
  flex-wrap: wrap;
  gap: 8px;
}
.theme-opt.on {
  border-color: var(--brass);
  color: var(--brass-bright);
  background: var(--brass-glow);
}
.grid2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
}
.k {
  font-size: 11.5px;
  color: var(--text-dim);
}
.tip {
  font-size: 10.5px;
  color: var(--text-faint);
  line-height: 1.6;
}
.tiny {
  font-size: 10.5px;
}
.actions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--line-soft);
  flex-wrap: wrap;
}
.skill-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}
.skill {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 13px 15px;
}
.sk-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.sk-name {
  flex: 1;
  font-size: 13.5px;
  color: var(--text-strong);
}
.sk-id {
  font-size: 10px;
  color: var(--text-faint);
}
.sk-desc {
  font-size: 11.5px;
  line-height: 1.65;
  color: var(--text-dim);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: 2.6em;
}
.sk-actions {
  display: flex;
  gap: 2px;
  margin-top: 4px;
  padding-top: 6px;
  border-top: 1px solid var(--line-soft);
}
.local-list {
  display: flex;
  flex-direction: column;
  gap: 1px;
  border: 1px solid var(--line-soft);
  border-radius: var(--r-lg);
  overflow: hidden;
  background: var(--line-soft);
}
.local-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 15px;
  background: var(--paper-900);
}
.lr-main {
  min-width: 0;
  gap: 2px;
}
.lr-name {
  font-size: 12.5px;
  color: var(--text-strong);
}
.lr-meta {
  font-size: 10px;
  color: var(--text-faint);
}
.lr-desc {
  font-size: 11.5px;
  color: var(--text-dim);
  max-width: 36%;
}
.pad.faint {
  padding: 12px 4px;
}
.preview {
  display: grid;
  grid-template-columns: 240px 1fr;
  gap: 12px;
  min-height: 380px;
  max-height: calc(100vh - 320px);
}
.files {
  display: flex;
  flex-direction: column;
  gap: 1px;
  overflow-y: auto;
  border-right: 1px solid var(--line-soft);
  padding-right: 10px;
}
.file {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 7px;
  border-radius: var(--r-sm);
  text-align: left;
  font-size: 11.5px;
  color: var(--text-dim);
}
.file:hover:not(:disabled) {
  background: var(--ink-750);
  color: var(--text);
}
.file.on {
  background: var(--ink-700);
  color: var(--text-strong);
}
.file.dir {
  color: var(--text-faint);
  cursor: default;
}
.f-name {
  flex: 1;
  font-family: var(--font-mono);
  font-size: 10.5px;
}
.f-size {
  font-size: 9px;
  color: var(--text-faint);
}
.content {
  overflow: auto;
  max-height: calc(100vh - 320px);
  font-size: 11.5px;
  line-height: 1.7;
}
</style>
