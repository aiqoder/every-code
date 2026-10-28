<script setup lang="ts">
import { feedbackState } from './feedback'
import UiButton from './UiButton.vue'
import UiDialog from './UiDialog.vue'

function dismiss(id: number): void {
  const index = feedbackState.toasts.findIndex((item) => item.id === id)
  if (index >= 0) feedbackState.toasts.splice(index, 1)
}

function onShow(value: boolean): void {
  if (feedbackState.confirmBusy) return
  if (!value) feedbackState.confirm = null
}

function cancel(): void {
  if (feedbackState.confirmBusy) return
  feedbackState.confirm = null
}

async function confirm(): Promise<void> {
  const current = feedbackState.confirm
  if (!current || feedbackState.confirmBusy) return
  const action = current.onPositiveClick
  if (!action) {
    feedbackState.confirm = null
    return
  }
  feedbackState.confirmBusy = true
  try {
    await action()
    if (feedbackState.confirm === current) feedbackState.confirm = null
  } catch {
    // 失败时留在对话框里，调用方可以再次确认
  } finally {
    feedbackState.confirmBusy = false
  }
}
</script>

<template>
  <div class="toaster" aria-live="polite">
    <TransitionGroup name="toast">
      <button
        v-for="toast in feedbackState.toasts"
        :key="toast.id"
        type="button"
        class="toast"
        :class="toast.type"
        @click="dismiss(toast.id)"
      >
        {{ toast.text }}
      </button>
    </TransitionGroup>
  </div>
  <UiDialog
    :show="feedbackState.confirm !== null"
    :title="feedbackState.confirm?.title"
    width="420px"
    :mask-closable="!feedbackState.confirmBusy"
    :close-on-esc="!feedbackState.confirmBusy"
    @update:show="onShow"
  >
    <p class="content">{{ feedbackState.confirm?.content }}</p>
    <template #footer>
      <UiButton :disabled="feedbackState.confirmBusy" @click="cancel">
        {{ feedbackState.confirm?.negativeText || '取消' }}
      </UiButton>
      <UiButton
        type="primary"
        :class="{ 'is-danger': feedbackState.confirm?.danger }"
        :loading="feedbackState.confirmBusy"
        :disabled="feedbackState.confirmBusy"
        @click="confirm"
      >
        {{ feedbackState.confirm?.positiveText || '确定' }}
      </UiButton>
    </template>
  </UiDialog>
</template>

<style scoped>
.toaster {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 5000;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
  pointer-events: none;
}

.toast {
  pointer-events: auto;
  max-width: min(360px, calc(100vw - 40px));
  padding: 10px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
  color: var(--text);
  font: inherit;
  font-size: var(--ec-text-body);
  line-height: 1.45;
  text-align: left;
  white-space: pre-wrap;
  cursor: pointer;
}

.toast.success {
  border-color: color-mix(in srgb, var(--success) 50%, var(--border));
  background: color-mix(in srgb, var(--success) 14%, var(--bg-elevated));
  color: var(--success);
}

.toast.error {
  border-color: color-mix(in srgb, var(--error) 50%, var(--border));
  background: color-mix(in srgb, var(--error) 14%, var(--bg-elevated));
  color: var(--error);
}

.toast.info {
  border-color: color-mix(in srgb, var(--accent) 50%, var(--border));
  background: color-mix(in srgb, var(--accent) 14%, var(--bg-elevated));
  color: var(--accent);
}

.toast.warning {
  border-color: color-mix(in srgb, var(--ec-attention) 50%, var(--border));
  background: color-mix(in srgb, var(--ec-attention) 14%, var(--bg-elevated));
  color: var(--ec-attention);
}

.toast-enter-active,
.toast-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

.content {
  margin: 0;
  white-space: pre-wrap;
}

:deep(.ui-btn.primary.is-danger) {
  border-color: var(--error);
  background: var(--error);
  color: #fff;
}

:deep(.ui-btn.primary.is-danger:hover:not(:disabled)) {
  border-color: color-mix(in srgb, var(--error) 86%, #000);
  background: color-mix(in srgb, var(--error) 86%, #000);
  color: #fff;
}
</style>
