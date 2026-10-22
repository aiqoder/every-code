<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSessionStore, messageOf } from '../../stores/session'
import { toast } from '../../composables/toast'
import FormField from '../../components/FormField.vue'
import TextField from '../../components/TextField.vue'

const session = useSessionStore()
const router = useRouter()
const route = useRoute()

const form = reactive({
  baseUrl: 'http://127.0.0.1:8183',
  username: '',
  password: ''
})
const busy = ref(false)
const error = ref('')

onMounted(async () => {
  const settings = await window.ade.getSettings()
  form.baseUrl = settings.profile.baseUrl || form.baseUrl
  form.username = settings.profile.username || ''
  if (settings.loggedIn) {
    await session.bootstrap()
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : ''
    router.replace(session.postLoginPath(redirect))
  }
})

async function submit(): Promise<void> {
  error.value = ''
  if (!form.baseUrl.trim()) {
    error.value = '请填写服务地址'
    return
  }
  if (!form.username.trim() || !form.password) {
    error.value = '请填写账号与密码'
    return
  }
  busy.value = true
  try {
    await session.login(form.baseUrl.trim(), form.username.trim(), form.password)
    toast.success('已连接服务端')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : ''
    router.replace(session.postLoginPath(redirect))
  } catch (err) {
    error.value = messageOf(err)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="login">
    <div class="glow" aria-hidden="true" />
    <div class="left">
      <div class="brand">
        <span class="mark serif">A</span>
        <div class="col">
          <span class="wordmark serif">ADE</span>
          <span class="sub">Agent Development Environment</span>
        </div>
      </div>

      <h1>把一条需求，交给<br />一队会自己动手的 Agent。</h1>
      <p class="lede">
        你只写一段描述。名称、概览、待办由 Agent 拆解；开发、验收、复核在它自己的 worktree
        里跑完。测试通过合入 dev，复核通过才进 main。部署前仍由你按下确认。
      </p>

      <div class="flow">
        <div class="node"><span class="num">01</span><span>描述需求</span></div>
        <span class="arrow">→</span>
        <div class="node"><span class="num">02</span><span>拆解待办</span></div>
        <span class="arrow">→</span>
        <div class="node"><span class="num">03</span><span>开发验收</span></div>
        <span class="arrow">→</span>
        <div class="node"><span class="num">04</span><span>复核发布</span></div>
      </div>
    </div>

    <div class="right">
      <form class="panel" @submit.prevent="submit">
        <header>
          <h2 class="serif">连接服务端</h2>
          <p>项目与流水线数据都在服务端，本机只留 Git 工作区、私钥与登录凭证。</p>
        </header>

        <FormField label="服务地址">
          <TextField v-model="form.baseUrl" placeholder="http://127.0.0.1:8183" spellcheck="false" />
        </FormField>

        <FormField label="账号">
          <TextField v-model="form.username" placeholder="user-feedback 账号" autocomplete="username" />
        </FormField>

        <FormField label="密码">
          <TextField
            v-model="form.password"
            type="password"
            placeholder="密码"
            autocomplete="current-password"
          />
        </FormField>

        <p v-if="error" class="err">{{ error }}</p>

        <button class="btn primary submit" type="submit" :disabled="busy">
          {{ busy ? '正在连接…' : '登录并接入执行面' }}
        </button>

        <p class="foot">
          登录后如果还没有可用模型，会先进入模型页完成配置。密钥保存在你的账号下。
        </p>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login {
  height: 100%;
  display: grid;
  grid-template-columns: 1.05fr 0.95fr;
  background: var(--ink-850);
  position: relative;
  overflow: hidden;
  -webkit-app-region: drag;
}
.glow {
  display: none;
}
.left {
  padding: 64px 56px 48px 64px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 26px;
  position: relative;
  z-index: 1;
}
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
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
}
.wordmark {
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--text-strong);
}
.sub {
  font-size: 12px;
  color: var(--text-faint);
}
h1 {
  font-size: 28px;
  font-weight: 600;
  line-height: 1.25;
  letter-spacing: -0.02em;
}
.lede {
  font-size: 14px;
  line-height: 1.5;
  color: var(--text-dim);
  max-width: 32em;
}
.flow {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 6px;
}
.node {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 6px 11px;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: var(--ink-800);
  font-size: 12px;
  color: var(--text-dim);
}
.num {
  font-family: var(--font-mono);
  font-size: 9.5px;
  color: var(--brass-dim);
}
.arrow {
  color: var(--text-faint);
  font-size: 12px;
}

.right {
  display: grid;
  place-items: center;
  padding: 48px;
  border-left: 1px solid var(--line-soft);
  background: var(--ink-800);
  -webkit-app-region: no-drag;
}
.panel {
  width: 100%;
  max-width: 380px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
header {
  margin-bottom: 6px;
}
header h2 {
  font-size: 20px;
  font-weight: 600;
}
header p {
  margin-top: 6px;
  font-size: 14px;
  line-height: 1.5;
  color: var(--text-dim);
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.k {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.err {
  font-size: 12px;
  color: #d98d7d;
  line-height: 1.6;
  padding: 7px 10px;
  border-radius: var(--r-sm);
  background: rgba(180, 91, 74, 0.12);
  border: 1px solid var(--rust-dim);
}
.submit {
  width: 100%;
  margin-top: 4px;
}
.foot {
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-faint);
}
</style>
