<script setup lang="ts">
defineProps<{
  open: boolean
  empty: string
  addLabel: string
  items: { id: string; name: string }[]
}>()

defineEmits<{
  add: []
  remove: [id: string]
}>()
</script>

<template>
  <div v-if="open" class="bound" @mousedown.stop>
    <ul>
      <li v-for="item in items" :key="item.id">
        <span>{{ item.name }}</span>
        <button type="button" :aria-label="`移除${item.name}`" @click="$emit('remove', item.id)">×</button>
      </li>
      <li v-if="!items.length" class="empty">{{ empty }}</li>
    </ul>
    <button type="button" class="add" @click="$emit('add')">{{ addLabel }}</button>
  </div>
</template>

<style scoped>
.bound {
  position: absolute;
  left: 0;
  bottom: calc(100% + 8px);
  z-index: 30;
  width: 280px;
  display: flex;
  flex-direction: column;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.18);
  overflow: hidden;
}

ul {
  list-style: none;
  margin: 0;
  padding: 4px;
  max-height: 240px;
  overflow: auto;
}

li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  font-size: 13px;
}

li.empty {
  color: var(--text-dim);
  justify-content: center;
}

li button {
  border: 0;
  background: transparent;
  color: var(--text-dim);
  font: inherit;
  cursor: pointer;
}

.add {
  margin: 0;
  border: 0;
  border-top: 1px solid var(--border);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: 13px;
  padding: 10px 12px;
  cursor: pointer;
  text-align: left;
}

.add:hover {
  background: var(--surface-hover);
}
</style>
