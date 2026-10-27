<script setup lang="ts">
import { ref } from 'vue'
import { useSessionStore } from '@/stores/sessionStore'
import LoginForm from '@/views/chat/components/LoginForm.vue'

const session = useSessionStore()
const error = ref('')
const pending = ref(false)

async function onSubmit(payload: { apiBase: string; apiKey: string }): Promise<void> {
  error.value = ''
  pending.value = true
  try {
    await session.login(payload.apiBase, payload.apiKey)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '登录失败'
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <div
    v-if="!session.loggedIn"
    class="absolute inset-0 z-30 flex items-center justify-center bg-[var(--ec-overlay)] px-6"
    role="presentation"
  >
    <div
      class="ec-skin-raised w-full max-w-sm rounded-2xl border border-[var(--ec-border)] px-6 py-6 text-[var(--ec-fg)] shadow-[0_24px_64px_rgba(0,0,0,0.28)]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="login-dialog-title"
    >
      <LoginForm :api-base="session.apiBase" :error="error" :pending="pending" @submit="onSubmit" />
    </div>
  </div>
</template>
