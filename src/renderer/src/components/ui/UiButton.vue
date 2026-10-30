<script setup lang="ts">
withDefaults(
  defineProps<{
    type?: 'default' | 'primary'
    quaternary?: boolean
    size?: 'tiny' | 'small' | 'medium'
    disabled?: boolean
    loading?: boolean
    block?: boolean
    attrType?: 'button' | 'submit'
  }>(),
  {
    type: 'default',
    quaternary: false,
    size: 'medium',
    disabled: false,
    loading: false,
    block: false,
    attrType: 'button'
  }
)
</script>

<template>
  <button
    class="ui-btn"
    :class="[type, size, { quaternary, block }]"
    :type="attrType"
    :disabled="disabled || loading"
    :aria-busy="loading || undefined"
  >
    <span v-if="loading" class="spinner" aria-hidden="true" />
    <slot />
  </button>
</template>

<style scoped>
.ui-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  box-sizing: border-box;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-weight: 500;
  line-height: 1.2;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background 0.12s,
    border-color 0.12s,
    color 0.12s;
}

.ui-btn:hover:not(:disabled) {
  background: var(--surface-hover);
}

.ui-btn:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.ui-btn.primary {
  border-color: var(--accent);
  background: var(--accent);
  color: var(--ec-on-accent);
}

.ui-btn.primary:hover:not(:disabled) {
  border-color: var(--accent-hover);
  background: var(--accent-hover);
  color: var(--ec-on-accent);
}

.ui-btn.quaternary {
  border-color: transparent;
  background: transparent;
  color: var(--text-muted);
}

.ui-btn.quaternary:hover:not(:disabled) {
  border-color: transparent;
  background: var(--surface-hover);
  color: var(--text-muted);
}

.ui-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.ui-btn.block {
  width: 100%;
}

.ui-btn.tiny {
  min-height: 22px;
  padding: 0 8px;
  font-size: var(--ec-text-meta);
}

.ui-btn.small {
  min-height: 28px;
  padding: 0 10px;
  font-size: var(--ec-text-body);
}

.ui-btn.medium {
  min-height: 34px;
  padding: 6px 14px;
  font-size: var(--ec-text-body);
}

.spinner {
  width: 12px;
  height: 12px;
  box-sizing: border-box;
  border: 1.5px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: ui-spin 0.7s linear infinite;
}

.ui-btn.tiny .spinner {
  width: 10px;
  height: 10px;
}

@keyframes ui-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
