<script setup lang="ts">
import { ref } from 'vue'
import UiButton from '@/components/ui/UiButton.vue'
import UiField from '@/components/ui/UiField.vue'

const props = defineProps<{
  apiBase: string
  error: string
  pending: boolean
}>()

const emit = defineEmits<{
  submit: [payload: { apiBase: string; apiKey: string }]
}>()

const apiBase = ref(props.apiBase)
const apiKey = ref('')

function onSubmit(): void {
  emit('submit', { apiBase: apiBase.value.trim(), apiKey: apiKey.value.trim() })
}
</script>

<template>
  <form class="w-full max-w-sm" @submit.prevent="onSubmit">
    <p class="text-xs tracking-[0.22em] text-[var(--ec-accent)] uppercase">Every Code</p>
    <h1 id="login-dialog-title" class="mt-3 text-2xl font-semibold tracking-tight text-[var(--ec-fg)]">
      密钥
    </h1>
    <p class="mt-2 text-sm leading-6 text-[var(--ec-fg-muted)]">
      填写接口地址和密钥后进入。聊天和工具都发往这个地址。
    </p>
    <label class="mt-8 block text-sm text-[var(--ec-fg)]">
      接口地址
      <UiField
        v-model="apiBase"
        class="mt-2"
        placeholder="http://127.0.0.1:8182"
        autocomplete="url"
        required
      />
    </label>
    <label class="mt-4 block text-sm text-[var(--ec-fg)]">
      密钥
      <UiField
        v-model="apiKey"
        class="mt-2"
        type="password"
        autocomplete="off"
        required
      />
    </label>
    <p v-if="props.error" class="mt-4 text-sm text-[var(--ec-danger)]">{{ props.error }}</p>
    <UiButton
      class="mt-6"
      type="primary"
      attr-type="submit"
      block
      :loading="props.pending"
      :disabled="props.pending"
    >
      {{ props.pending ? '登录中…' : '进入' }}
    </UiButton>
  </form>
</template>
