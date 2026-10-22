<script setup lang="ts">
defineProps<{
  open: boolean
  files: number
  fileList: string[]
  messageCount: number
}>()

const emit = defineEmits<{
  cancel: []
  confirm: []
}>()
</script>

<template>
  <div v-if="open" class="overlay">
    <button type="button" class="backdrop" aria-label="关闭" @click="emit('cancel')" />
    <div class="panel" role="dialog" aria-modal="true" aria-label="确认回退">
      <h3>确认回退</h3>
      <p v-if="files > 0">将还原 {{ files }} 个文件的修改：</p>
      <p v-else>本次回退不涉及工作目录文件修改。</p>
      <ul v-if="fileList.length">
        <li v-for="f in fileList.slice(0, 10)" :key="f">{{ f }}</li>
        <li v-if="fileList.length > 10" class="more">…还有 {{ fileList.length - 10 }} 个文件</li>
      </ul>
      <p v-if="messageCount > 0" class="warn">同时将删除该消息之后的 {{ messageCount }} 条对话记录。</p>
      <footer>
        <button type="button" class="btn ghost" @click="emit('cancel')">取消还原</button>
        <button type="button" class="btn primary" @click="emit('confirm')">确认并发送</button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 110;
  display: grid;
  place-items: center;
  padding: 24px;
}

.backdrop {
  position: absolute;
  inset: 0;
  border: 0;
  background: rgba(0, 0, 0, 0.5);
}

.panel {
  position: relative;
  z-index: 1;
  width: min(480px, 100%);
  padding: 18px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
}

h3 {
  margin: 0 0 10px;
  font-size: 16px;
}

p {
  margin: 0 0 8px;
  font-size: 13px;
  color: var(--text-muted);
}

ul {
  margin: 0 0 12px;
  padding-left: 18px;
  font-size: 12px;
  color: var(--text);
  max-height: 160px;
  overflow: auto;
}

.more {
  list-style: none;
  padding-left: 0;
  color: var(--text-dim);
}

.warn {
  color: var(--error);
}

footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}

.btn {
  font: inherit;
  font-size: 13px;
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  cursor: pointer;
}

.btn.ghost {
  background: transparent;
}

.btn.primary {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}
</style>
