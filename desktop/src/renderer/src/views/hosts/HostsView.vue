<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { adeApi, bridge } from '../../api'
import { useSessionStore, messageOf } from '../../stores/session'
import { useShellStore } from '../../stores/shell'
import { confirm } from '../../composables/confirm'
import { toast } from '../../composables/toast'
import Dialog from '../../components/Dialog.vue'
import FormField from '../../components/FormField.vue'
import NumberField from '../../components/NumberField.vue'
import TextArea from '../../components/TextArea.vue'
import TextField from '../../components/TextField.vue'
import EmptyState from '../../components/EmptyState.vue'
import { parsePlaybook, stepsSummary } from '../../utils/playbook'
import type { DeployHost } from '../../../../shared/types'

const session = useSessionStore()
const shell = useShellStore()

const hosting = ref(false)
const hostOpen = ref(false)
const editingHost = ref<DeployHost | null>(null)
const hostForm = reactive({
  name: '',
  host: '',
  port: 22,
  username: 'root',
  remoteDir: '/srv/app',
  playbook: ''
})
const hostErr = ref('')
const keyPath = ref('')

const hosts = computed(() => session.hosts)
const playbookSteps = computed(() => stepsSummary(parsePlaybook(hostForm.playbook || '')))

function openHostForm(host?: DeployHost): void {
  editingHost.value = host ?? null
  hostErr.value = ''
  if (host) {
    hostForm.name = host.name
    hostForm.host = host.host
    hostForm.port = host.port
    hostForm.username = host.username
    hostForm.remoteDir = host.remoteDir
    hostForm.playbook = host.playbook
    void bridge.getSshKey(host.id).then((p) => (keyPath.value = p))
  } else {
    hostForm.name = ''
    hostForm.host = ''
    hostForm.port = 22
    hostForm.username = 'root'
    hostForm.remoteDir = '/srv/app'
    hostForm.playbook = ''
    keyPath.value = ''
  }
  hostOpen.value = true
}

async function saveHost(): Promise<void> {
  hostErr.value = ''
  if (!hostForm.name.trim() || !hostForm.host.trim()) {
    hostErr.value = '主机名称与地址不能为空'
    return
  }
  hosting.value = true
  try {
    const saved = await adeApi.saveHost({
      id: editingHost.value?.id ?? 0,
      name: hostForm.name.trim(),
      host: hostForm.host.trim(),
      port: hostForm.port || 22,
      username: hostForm.username.trim(),
      remoteDir: hostForm.remoteDir.trim(),
      playbook: hostForm.playbook
    })
    const idx = session.hosts.findIndex((h) => h.id === saved.id)
    if (idx >= 0) session.hosts[idx] = saved
    else session.hosts.push(saved)
    hostOpen.value = false
    toast.success('主机档案已保存')
  } catch (err) {
    hostErr.value = messageOf(err)
  } finally {
    hosting.value = false
  }
}

async function removeHost(host: DeployHost): Promise<void> {
  const ok = await confirm({
    title: '删除主机',
    message: `删除「${host.name}」后，本机私钥保留。`,
    confirmText: '删除',
    danger: true
  })
  if (!ok) return
  try {
    await adeApi.deleteHost(host.id)
    session.hosts = session.hosts.filter((h) => h.id !== host.id)
    toast.show('主机已删除')
  } catch (err) {
    toast.error(messageOf(err))
  }
}

async function chooseKey(host: DeployHost): Promise<void> {
  const picked = await bridge.pickSshKey(host.id)
  if (!picked) return
  keyPath.value = picked
  toast.success('私钥已记录在本机')
}

async function openTerminal(host: DeployHost): Promise<void> {
  try {
    const res = await bridge.openSshTab({
      hostId: host.id,
      host: host.host,
      port: host.port,
      username: host.username,
      fingerprint: host.fingerprint,
      remoteDir: host.remoteDir,
      title: `SSH ${host.name}`
    })
    await shell.adopt(res.tab)
    if (!host.fingerprint && res.fingerprint) {
      const saved = await adeApi.saveHost({
        id: host.id,
        name: host.name,
        host: host.host,
        port: host.port,
        username: host.username,
        remoteDir: host.remoteDir,
        fingerprint: res.fingerprint
      })
      const idx = session.hosts.findIndex((h) => h.id === saved.id)
      if (idx >= 0) session.hosts[idx] = saved
      toast.show(`已记录主机指纹 ${res.fingerprint.slice(0, 24)}…`)
    }
  } catch (err) {
    toast.error(messageOf(err))
  }
}

onMounted(async () => {
  if (!session.hosts.length) await session.loadHosts()
})
</script>

<template>
  <div class="hosts">
    <header class="page-head">
      <div class="col">
        <span class="eyebrow">主机</span>
        <h1 class="serif">SSH 档案</h1>
        <p class="lede">地址和剧本在服务端，私钥只留在这台机器。从这里打开 SSH 终端。</p>
      </div>
      <div class="spacer" />
      <button class="btn primary sm" @click="openHostForm()">＋ 添加主机</button>
    </header>

    <div class="scroll">
      <EmptyState
        v-if="!hosts.length"
        empty-title="还没有主机档案"
        empty-hint="建一台主机：名称、地址、端口、用户、远端目录。"
      />
      <div v-else class="host-grid">
        <article v-for="h in hosts" :key="h.id" class="host card">
          <div class="h-head">
            <span class="h-name serif ellipsis">{{ h.name }}</span>
            <span class="tag tiny">{{ h.port }}</span>
          </div>
          <div class="h-addr mono ellipsis">{{ h.username }}@{{ h.host }}</div>
          <div class="h-dir mono ellipsis">{{ h.remoteDir || '/srv/app' }}</div>
          <div class="h-fp mono ellipsis" :class="{ ok: h.fingerprint }">
            {{ h.fingerprint ? h.fingerprint : '指纹未记录，首次连接会核对' }}
          </div>
          <div class="h-actions">
            <button class="btn ghost sm" @click="openTerminal(h)">SSH 终端</button>
            <button class="btn ghost sm" @click="chooseKey(h)">私钥</button>
            <button class="btn ghost sm" @click="openHostForm(h)">编辑</button>
            <button class="btn ghost sm danger" @click="removeHost(h)">删除</button>
          </div>
        </article>
      </div>
    </div>

    <Dialog
      :open="hostOpen"
      :title="editingHost ? '编辑主机' : '添加主机'"
      subtitle="私钥文件在本机选择，不上传服务端。"
      width="600px"
      :persistent="hosting"
      @close="hostOpen = false"
    >
      <div class="form">
        <div class="grid2">
          <FormField label="名称"><TextField v-model="hostForm.name" /></FormField>
          <FormField label="地址"><TextField v-model="hostForm.host" spellcheck="false" /></FormField>
          <FormField label="端口"><NumberField v-model="hostForm.port" :min="1" :max="65535" /></FormField>
          <FormField label="用户"><TextField v-model="hostForm.username" spellcheck="false" /></FormField>
        </div>
        <FormField label="远端目录">
          <TextField v-model="hostForm.remoteDir" spellcheck="false" />
        </FormField>
        <FormField label="本机私钥">
          <div class="row">
            <TextField class="grow" :model-value="keyPath" readonly placeholder="尚未选择" spellcheck="false" />
            <button class="btn sm" :disabled="!editingHost" @click="editingHost && chooseKey(editingHost)">选择</button>
          </div>
        </FormField>
        <FormField label="备注剧本">
          <TextArea v-model="hostForm.playbook" :rows="8" spellcheck="false" placeholder="可选。部署对话不再自动跑剧本。" />
          <span v-if="hostForm.playbook.trim()" class="tip">{{ playbookSteps || '未解析到步骤' }}</span>
        </FormField>
      </div>
      <p v-if="hostErr" class="err">{{ hostErr }}</p>
      <template #footer>
        <button class="btn ghost" @click="hostOpen = false">取消</button>
        <div class="spacer" />
        <button class="btn primary" :disabled="hosting" @click="saveHost">{{ hosting ? '保存中…' : '保存' }}</button>
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
.hosts { flex: 1; display: flex; flex-direction: column; min-height: 0; }
.page-head {
  display: flex; align-items: flex-end; gap: 22px;
  padding: 22px 26px 14px; border-bottom: 1px solid var(--line-soft); flex-wrap: wrap;
}
.eyebrow { font-size: 10.5px; letter-spacing: 0.18em; color: var(--brass-dim); }
h1 { font-size: 21px; }
.lede { margin-top: 6px; font-size: 12px; color: var(--text-dim); max-width: 46em; }
.scroll { flex: 1; overflow: auto; padding: 20px 26px 36px; }
.host-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 12px; }
.host { display: flex; flex-direction: column; gap: 6px; padding: 14px 16px; }
.h-head, .h-actions, .row { display: flex; align-items: center; gap: 8px; }
.row .grow { flex: 1; min-width: 0; width: auto; }
.h-name { flex: 1; font-size: 14px; color: var(--text-strong); }
.h-addr { font-size: 11.5px; }
.h-dir, .h-fp { font-size: 10.5px; color: var(--text-faint); }
.h-fp.ok { color: var(--moss-dim); }
.h-actions { margin-top: 4px; padding-top: 6px; border-top: 1px solid var(--line-soft); flex-wrap: wrap; }
.form { display: flex; flex-direction: column; gap: 12px; }
.grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.k { font-size: 11px; color: var(--text-faint); }
.err { color: var(--rust); font-size: 12px; margin-top: 8px; }
textarea { min-height: 120px; }
</style>
