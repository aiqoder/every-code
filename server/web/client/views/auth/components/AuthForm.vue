<script setup lang="ts">
import { reactive, watch } from 'vue'

const emit = defineEmits<{
  login: [payload: { serverURL: string; username: string; password: string }]
  serverChange: [url: string]
}>()

const props = defineProps<{
  busy: boolean
  notice: string
  defaultServer: string
}>()

const form = reactive({
  serverURL: props.defaultServer || 'http://127.0.0.1:8182',
  username: '',
  password: '',
})

watch(
  () => form.serverURL,
  (url) => emit('serverChange', url),
  { immediate: true },
)

function submit() {
  emit('login', { serverURL: form.serverURL, username: form.username, password: form.password })
}
</script>

<template>
  <form class="auth-form" @submit.prevent="submit">
    <label>
      服务端
      <input v-model="form.serverURL" required placeholder="http://127.0.0.1:8182" />
    </label>
    <label>
      账号
      <input v-model="form.username" required autocomplete="username" placeholder="请输入账号" />
    </label>
    <label>
      密码
      <input v-model="form.password" type="password" required placeholder="密码" />
    </label>
    <p v-if="notice" class="err">{{ notice }}</p>
    <button class="btn-go" type="submit" :disabled="busy">
      {{ busy ? '请稍候…' : '进入' }}
    </button>
  </form>
</template>

<style scoped>
.auth-form {
  display: grid;
  gap: 16px;
}

label {
  display: grid;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-muted);
}

input {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
  padding: 10px 14px;
  font-size: 14px;
  border-radius: var(--radius-sm);
}

input::placeholder {
  color: var(--text-dim);
}

.btn-go {
  margin-top: 4px;
}

.err {
  color: var(--error);
  margin: 0;
  font-size: 13px;
}
</style>
