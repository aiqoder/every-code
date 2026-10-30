<script setup lang="ts">
import { ref } from 'vue'
import { onClickOutside } from '@vueuse/core'
import UiButton from '@/components/ui/UiButton.vue'
import UiField from '@/components/ui/UiField.vue'
import { normalizeApiBase, readApiBaseHistory, rememberApiBase } from '@/lib/sessionConfig'

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
const history = ref(readApiBaseHistory())
const historyOpen = ref(false)
const addressRoot = ref<HTMLElement | null>(null)

onClickOutside(addressRoot, () => {
  historyOpen.value = false
})

function toggleHistory(): void {
  historyOpen.value = !historyOpen.value
}

function pickAddress(value: string): void {
  apiBase.value = value
  historyOpen.value = false
}

function onAddressFocus(): void {
  if (history.value.length > 0) historyOpen.value = true
}

function onAddressKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') historyOpen.value = false
  if (event.key === 'ArrowDown' && history.value.length > 0) {
    event.preventDefault()
    historyOpen.value = true
  }
}

function onSubmit(): void {
  const base = apiBase.value.trim()
  try {
    rememberApiBase(normalizeApiBase(base))
    history.value = readApiBaseHistory()
  } catch {
    // 空地址交给登录流程提示。
  }
  emit('submit', { apiBase: base, apiKey: apiKey.value.trim() })
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
      <div ref="addressRoot" class="relative mt-2" :class="{ 'z-20': historyOpen }">
        <UiField
          v-model="apiBase"
          placeholder="http://127.0.0.1:8182"
          autocomplete="off"
          required
          :style="history.length ? { paddingRight: '36px' } : undefined"
          @focus="onAddressFocus"
          @keydown="onAddressKeydown"
        />
        <button
          v-if="history.length"
          type="button"
          class="address-toggle"
          :aria-expanded="historyOpen"
          aria-haspopup="listbox"
          aria-label="选择曾经用过的接口地址"
          @mousedown.prevent
          @click="toggleHistory"
        >
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path
              d="M4 6.5 8 10.5 12 6.5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
        <ul v-if="historyOpen" class="address-menu" role="listbox">
          <li v-for="item in history" :key="item">
            <button
              type="button"
              class="address-option"
              :class="{ current: item === apiBase.trim() }"
              role="option"
              :aria-selected="item === apiBase.trim()"
              :title="item"
              @mousedown.prevent="pickAddress(item)"
            >
              {{ item }}
            </button>
          </li>
        </ul>
      </div>
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
      class="mt-10"
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

<style scoped>
.address-toggle {
  position: absolute;
  top: 50%;
  right: 6px;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--text-muted);
  transform: translateY(-50%);
  cursor: pointer;
}

.address-toggle:hover,
.address-toggle[aria-expanded='true'] {
  background: var(--surface-hover);
  color: var(--text);
}

.address-toggle svg {
  width: 14px;
  height: 14px;
}

.address-toggle[aria-expanded='true'] svg {
  transform: rotate(180deg);
}

.address-menu {
  position: absolute;
  z-index: 20;
  top: calc(100% + 4px);
  right: 0;
  left: 0;
  max-height: 180px;
  margin: 0;
  overflow: auto;
  padding: 4px;
  list-style: none;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
}

.address-option {
  display: block;
  width: 100%;
  overflow: hidden;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--ec-text-body);
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.address-option:hover {
  background: var(--surface-hover);
}

.address-option.current {
  background: var(--accent-soft);
  color: var(--accent);
}
</style>
