<script setup lang="ts">
defineProps<{
  label: string
  error?: string
  required?: boolean
}>()
</script>

<template>
  <div class="form-field" :class="{ invalid: !!error }">
    <span class="form-field-label">
      {{ label }}
      <span v-if="required" class="required">*</span>
    </span>
    <slot />
    <p v-if="error" class="form-field-error">{{ error }}</p>
  </div>
</template>

<style scoped>
.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-field-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-muted);
}

.required {
  margin-left: 2px;
  color: var(--error);
}

.form-field-error {
  margin: 0;
  font-size: 12px;
  color: var(--error);
}
</style>
