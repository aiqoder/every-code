<script setup lang="ts">
withDefaults(
  defineProps<{
    /** 会话名称；为空时显示占位文案。 */
    title?: string
  }>(),
  {
    title: '',
  },
)
</script>

<template>
  <header class="top-bar">
    <slot name="leading" />
    <span class="conv-name" :title="title || '未选择会话'">
      {{ title || '未选择会话' }}
    </span>
  </header>
</template>

<style scoped>
.top-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
  min-height: 40px;
  padding: 4px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-chrome-soft);
}

.conv-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  color: var(--text);
}
</style>
