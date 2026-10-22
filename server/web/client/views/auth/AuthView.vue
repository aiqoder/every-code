<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import BrandLogo from '../../components/BrandLogo.vue'
import { useBrand } from '../../composables/useBrand'
import { brandApiBase } from '../../lib/brand'
import { useSessionStore } from '../../stores/session'
import AuthForm from './components/AuthForm.vue'

const session = useSessionStore()
const router = useRouter()
const serverURL = ref(session.status.serverUrl || 'http://127.0.0.1:8182')
const { brand } = useBrand(() => brandApiBase(serverURL.value))

async function onLogin(p: { serverURL: string; username: string; password: string }) {
  try {
    await session.login(p.serverURL, p.username, p.password)
    await router.push('/bot/chat')
  } catch {
    /* 错误文案由 session.notice 展示 */
  }
}
</script>

<template>
  <div class="gate">
    <aside class="hero">
      <div class="glow" />
      <BrandLogo class="emoji" :logo="brand.logo" :alt="brand.name" />
      <h1>{{ brand.name }}</h1>
      <p class="tagline">让 AI 用上你的电脑</p>
    </aside>
    <section class="card">
      <AuthForm
        :busy="session.busy"
        :notice="session.notice"
        :default-server="session.status.serverUrl"
        @server-change="serverURL = $event"
        @login="onLogin"
      />
    </section>
  </div>
</template>

<style scoped>
.gate {
  min-height: 100%;
  display: grid;
  grid-template-columns: 1fr 420px;
  padding: 48px 56px;
  gap: 48px;
  align-items: center;
}

.hero {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}

.glow {
  position: absolute;
  top: -40px;
  left: -20px;
  width: 200px;
  height: 200px;
  background: radial-gradient(circle, var(--accent-glow), transparent 70%);
  pointer-events: none;
}

.emoji {
  font-size: 56px;
  line-height: 1;
  position: relative;
}

h1 {
  font-family: var(--sans);
  font-weight: 600;
  font-size: 32px;
  line-height: 1.25;
  margin: 0;
  letter-spacing: -0.02em;
  position: relative;
}

.tagline {
  margin: 0;
  color: var(--text-muted);
  font-size: 14px;
  line-height: 1.5;
  position: relative;
}

.card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 24px;
  box-shadow: var(--shadow);
}
</style>
