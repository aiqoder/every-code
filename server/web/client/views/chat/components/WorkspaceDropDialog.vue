<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  open: boolean
  folderPath: string
}>()

const emit = defineEmits<{
  cancel: []
  confirm: []
}>()

const folderName = computed(() => {
  const p = props.folderPath.replace(/\\/g, '/').replace(/\/+$/, '')
  const i = p.lastIndexOf('/')
  return i >= 0 ? p.slice(i + 1) || p : p
})
</script>

<template>
  <div v-if="open" class="overlay">
    <button type="button" class="backdrop" aria-label="关闭" @click="emit('cancel')" />
    <div class="panel" role="dialog" aria-modal="true" aria-label="添加工作目录">
      <h3>添加工作目录</h3>
      <p>是否将文件夹「{{ folderName }}」添加到工作目录并切换？</p>
      <p v-if="folderPath" class="path" :title="folderPath">{{ folderPath }}</p>
      <footer>
        <button type="button" class="btn ghost" @click="emit('cancel')">取消</button>
        <button type="button" class="btn primary" @click="emit('confirm')">确认</button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 130;
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
  line-height: 1.5;
}

.path {
  font-size: 12px;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 14px;
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
  color: var(--text);
}

.btn.primary {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}
</style>
