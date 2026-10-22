<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Agent } from '../../../types'
import { skillAvatarColor, skillInitial } from '../../../lib/skillAvatar'

const props = defineProps<{
  agents: Agent[]
  activeId: string | null
}>()

const emit = defineEmits<{
  select: [id: string]
}>()

const query = ref('')

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.agents
  return props.agents.filter((a) => {
    const name = a.name.toLowerCase()
    const desc = (a.description || '').toLowerCase()
    return name.includes(q) || desc.includes(q)
  })
})

function descText(a: Agent) {
  return a.description?.trim() || '暂无简介'
}

function fallbackColor(a: Agent) {
  return skillAvatarColor(String(a.id))
}
</script>

<template>
  <aside class="panel" aria-label="智能体列表">
    <div class="panel-head">
      <span class="panel-title">智能体</span>
      <span class="panel-count">{{ agents.length }}</span>
    </div>
    <div class="search-wrap">
      <input
        v-model="query"
        type="search"
        class="search"
        placeholder="搜索智能体"
        autocomplete="off"
        aria-label="搜索智能体"
      />
    </div>
    <div class="list-scroll">
      <template v-if="filtered.length">
        <ul class="buddy-list">
          <li v-for="a in filtered" :key="a.id">
              <button
                type="button"
                class="buddy"
                :class="{ on: a.id === activeId }"
                :title="a.name"
                :aria-pressed="a.id === activeId"
                @click="emit('select', a.id)"
              >
                <span class="avatar-wrap">
                  <img v-if="a.avatar" :src="a.avatar" class="avatar" alt="" />
                  <span
                    v-else
                    class="avatar fallback"
                    :style="{ background: fallbackColor(a) }"
                  >
                    {{ skillInitial(a.name) }}
                  </span>
                  <span
                    v-if="a.id === activeId"
                    class="status"
                    title="当前选用"
                    aria-hidden="true"
                  />
                </span>
                <span class="meta">
                  <strong class="name">{{ a.name }}</strong>
                  <span class="sign">{{ descText(a) }}</span>
                </span>
              </button>
            </li>
        </ul>
      </template>
      <p v-else class="empty">{{ query.trim() ? '无匹配智能体' : '暂无智能体' }}</p>
    </div>
  </aside>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  gap: 8px;
  padding: 12px 8px 12px;
  border-left: 1px solid var(--border);
  background: var(--bg-chrome-soft);
}

.panel-head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  padding: 0 6px;
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
  font-weight: 400;
  line-height: 18px;
}

.search-wrap {
  flex-shrink: 0;
  padding: 0 4px;
}

.search {
  width: 100%;
  height: 30px;
  margin: 0;
  padding: 0 10px;
  border: 0;
  border-radius: 6px;
  background: var(--surface);
  color: var(--text);
  font: inherit;
  font-size: 12px;
  outline: none;
}

.search::placeholder {
  color: var(--text-dim);
}

.search:focus {
  box-shadow: inset 0 0 0 1px var(--border-strong);
}

.list-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.buddy-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.buddy {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-width: 0;
  padding: 8px 8px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.buddy:hover {
  background: var(--surface-hover);
}

.buddy.on {
  background: var(--accent-soft);
}

.avatar-wrap {
  position: relative;
  flex-shrink: 0;
  width: 36px;
  height: 36px;
}

.avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  display: block;
  background: var(--surface);
}

.avatar.fallback {
  display: grid;
  place-items: center;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0;
}

.status {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: #34c77b;
  box-shadow: 0 0 0 2px var(--bg-elevated);
}

.buddy.on .status {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--accent) 12%, var(--bg-elevated));
}

.meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
  color: var(--text);
}

.buddy.on .name {
  color: var(--accent);
}

.sign {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 400;
  line-height: 16px;
  color: var(--text-dim);
}

.buddy.on .sign {
  color: var(--text-muted);
}

.empty {
  margin: 0;
  padding: 16px 8px;
  color: var(--text-dim);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  text-align: center;
}
</style>
