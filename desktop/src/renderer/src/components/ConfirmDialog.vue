<script setup lang="ts">
import { computed, onBeforeUnmount } from 'vue'
import Dialog from './Dialog.vue'
import { acceptConfirm, cancelConfirm, confirmState, dismissConfirm } from '../composables/confirm'

const current = computed(() => confirmState.current)
const busy = computed(() => confirmState.busy)

onBeforeUnmount(dismissConfirm)
</script>

<template>
  <Dialog
    :open="!!current"
    :title="current?.title"
    :subtitle="current?.subtitle"
    width="420px"
    :persistent="busy"
    @close="cancelConfirm"
  >
    <p v-if="current" class="copy">{{ current.message }}</p>
    <p v-if="current?.detail" class="detail ellipsis" :title="current.detail">{{ current.detail }}</p>
    <template v-if="current" #footer>
      <button class="btn ghost" type="button" :disabled="busy" @click="cancelConfirm">
        {{ current.cancelText }}
      </button>
      <div class="spacer" />
      <button
        class="btn"
        :class="current.danger ? 'danger' : 'primary'"
        type="button"
        :disabled="busy"
        @click="acceptConfirm"
      >
        {{ busy ? current.busyText : current.confirmText }}
      </button>
    </template>
  </Dialog>
</template>

<style scoped>
.copy {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
}
.detail {
  margin: 10px 0 0;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-dim);
}
</style>
