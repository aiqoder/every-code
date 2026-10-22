<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Agent, Conversation } from '../../../types'
import IconPlus from './icons/IconPlus.vue'
import IconChevronDown from './icons/IconChevronDown.vue'

const props = defineProps<{
  /** 全部会话（父组件已按工作目录过滤） */
  conversations: Conversation[]
  /** 当前选中的智能体（null 表示未选中） */
  activeAgent: Agent | null
  /** 当前打开的会话 ID */
  activeId: number | null
}>()

const emit = defineEmits<{
  /** 打开会话 */
  open: [id: number]
  /** 新建会话 */
  create: []
  /** 删除会话 */
  remove: [id: number]
}>()

const collapsed = ref(false)

/** 当前智能体下的会话（按 ID 倒序 = 最新在前） */
const items = computed(() => {
  const agentId = props.activeAgent?.id
  if (!agentId) return []
  return props.conversations
    .filter((c) => (c.agentId || 0) === agentId)
    .sort((a, b) => b.id - a.id)
})
</script>

<template>
  <aside class="panel" aria-label="历史会话">
    <div class="panel-head">
      <button
        type="button"
        class="collapse-btn"
        :aria-expanded="!collapsed"
        :aria-label="collapsed ? '展开会话列表' : '折叠会话列表'"
        @click="collapsed = !collapsed"
      >
        <IconChevronDown class="chevron" :class="{ collapsed }" />
      </button>
      <span class="panel-title">历史会话</span>
      <span class="panel-count">{{ items.length }}</span>
      <button
        type="button"
        class="head-act"
        title="新建会话"
        aria-label="新建会话"
        :disabled="!activeAgent"
        @click="emit('create')"
      >
        <IconPlus />
      </button>
    </div>
    <p v-if="activeAgent" class="agent-name" :title="activeAgent.name">{{ activeAgent.name }}</p>
    <template v-if="!collapsed">
      <ul v-if="items.length" class="conv-list">
        <li
          v-for="c in items"
          :key="c.id"
          :class="{ on: c.id === activeId }"
          @click="emit('open', c.id)"
        >
          <strong :title="c.title">{{ c.title }}</strong>
          <button type="button" class="del" aria-label="删除会话" @click.stop="emit('remove', c.id)">×</button>
        </li>
      </ul>
      <p v-else-if="activeAgent" class="empty">该智能体下暂无会话</p>
      <p v-else class="empty">请先选择智能体</p>
    </template>
  </aside>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  gap: 6px;
  padding: 12px 12px 16px;
  border-left: 1px solid var(--border);
  background: var(--bg-chrome-soft);
}

.panel-head {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  padding: 0 2px;
}

.collapse-btn {
  display: grid;
  place-items: center;
  width: 18px;
  height: 22px;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
}

.collapse-btn:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.chevron {
  width: 14px;
  height: 14px;
  transition: transform 0.16s ease;
}

.chevron.collapsed {
  transform: rotate(-90deg);
}

.panel-title {
  flex: 1;
  min-width: 0;
  color: var(--text-dim);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.panel-count {
  flex-shrink: 0;
  color: var(--text-dim);
  font-size: 12px;
  line-height: 18px;
}

.head-act {
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  flex-shrink: 0;
  cursor: pointer;
}

.head-act svg {
  width: 14px;
  height: 14px;
}

.head-act:hover:not(:disabled) {
  color: var(--accent);
  background: var(--surface-hover);
}

.head-act:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.agent-name {
  margin: 0;
  padding: 0 4px;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 18px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conv-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 4px;
  min-width: 0;
  overflow: auto;
}

.conv-list li {
  position: relative;
  min-width: 0;
  overflow: hidden;
  padding: 6px 28px 6px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.12s;
}

.conv-list li:hover {
  background: var(--surface-hover);
}

.conv-list li.on {
  background: var(--accent-soft);
}

.conv-list li.on strong {
  color: var(--accent);
}

.conv-list strong {
  display: block;
  min-width: 0;
  max-width: 100%;
  font-size: 13px;
  font-weight: 400;
  line-height: 20px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conv-list .del {
  position: absolute;
  right: 4px;
  top: 50%;
  transform: translateY(-50%);
  background: transparent;
  border: 0;
  color: var(--text-dim);
  font-size: 15px;
  line-height: 1;
  padding: 2px 5px;
  border-radius: var(--radius-sm);
  opacity: 0;
  transition: opacity 0.12s;
}

.conv-list li:hover .del {
  opacity: 1;
}

.conv-list .del:hover {
  color: var(--error);
  background: rgba(248, 113, 113, 0.1);
}

.empty {
  margin: 0;
  padding: 12px 4px;
  color: var(--text-dim);
  font-size: 12px;
  line-height: 18px;
}
</style>
