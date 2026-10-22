<script setup lang="ts">
import type { FavoritePrompt } from '../../../stores/favoritePrompts'
import { parseContentParts } from '../../../lib/skillToken'

const props = defineProps<{
  open: boolean
  items: FavoritePrompt[]
  nameById?: Map<string, string> | Record<string, string>
}>()

const emit = defineEmits<{
  close: []
  pick: [content: string]
  send: [content: string]
  create: []
  edit: [item: FavoritePrompt]
  remove: [id: string]
}>()

function skillName(id: string) {
  const map = props.nameById
  if (!map) return id
  if (map instanceof Map) return map.get(id) || id
  return map[id] || id
}

function parts(content: string) {
  return parseContentParts(content || '')
}
</script>

<template>
  <div v-if="open" class="pop">
    <div class="toolbar">
      <button type="button" class="text-btn accent" @click="emit('create')">新增</button>
    </div>
    <ul>
      <li v-if="!items.length" class="empty">暂无常用提示词</li>
      <li v-for="(item, index) in items" :key="item.id" class="item-row">
        <span class="index">{{ index + 1 }}.</span>
        <button type="button" class="pick" :title="item.content" @click="emit('pick', item.content)">
          <span class="preview">
            <template v-for="(p, i) in parts(item.content)" :key="i">
              <span v-if="p.type === 'skill'" class="skill-chip">/{{ skillName(p.id) }}</span>
              <template v-else>{{ p.text }}</template>
            </template>
          </span>
        </button>
        <div class="actions">
          <button type="button" class="text-btn" @click="emit('send', item.content)">发送</button>
          <button type="button" class="text-btn" @click="emit('edit', item)">编辑</button>
          <button type="button" class="text-btn danger" @click="emit('remove', item.id)">删除</button>
        </div>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.pop {
  position: absolute;
  right: 0;
  bottom: calc(100% + 6px);
  z-index: 30;
  width: min(280px, 72vw);
  max-height: 360px;
  overflow: auto;
  padding: 6px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
}

.toolbar {
  display: flex;
  justify-content: flex-end;
  padding: 2px 4px 6px;
}

ul {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 4px;
}

li.empty {
  color: var(--text-dim);
  font-size: 12px;
  padding: 8px;
}

.item-row {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  background: var(--surface);
}

.index {
  flex-shrink: 0;
  width: 18px;
  font-size: 12px;
  color: var(--text-dim);
  text-align: right;
}

.pick {
  flex: 1;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.preview {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  color: var(--text);
  line-height: 1.5;
}

.skill-chip {
  display: inline;
  margin: 0 2px;
  color: var(--accent);
  font-weight: 600;
}

.actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 4px;
}

.text-btn {
  font: inherit;
  font-size: 12px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  white-space: nowrap;
}

.text-btn:hover {
  color: var(--text);
}

.text-btn.accent {
  color: var(--accent);
}

.text-btn.accent:hover {
  color: var(--accent-hover, var(--accent));
}

.text-btn.danger:hover {
  color: var(--error);
}
</style>
