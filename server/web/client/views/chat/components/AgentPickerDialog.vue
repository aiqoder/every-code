<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Agent } from '../../../types'
import { skillAvatarColor, skillInitial } from '../../../lib/skillAvatar'
import IconPlus from './icons/IconPlus.vue'

const props = defineProps<{
  open: boolean
  /** 可选智能体（全部可见智能体） */
  allAgents: Agent[]
  /** 已加入当前目录的智能体 ID */
  memberIds: string[]
}>()

const emit = defineEmits<{
  close: []
  /** 确认添加选中的智能体 */
  confirm: [ids: string[]]
}>()

const query = ref('')
const checked = ref<Record<string, boolean>>({})

/** 待选列表：排除已加入当前目录的智能体 */
const candidates = computed(() => {
  const member = new Set(props.memberIds)
  const q = query.value.trim().toLowerCase()
  return props.allAgents.filter((a) => {
    if (member.has(a.id)) return false
    if (!q) return true
    return (
      a.name.toLowerCase().includes(q) ||
      (a.description || '').toLowerCase().includes(q)
    )
  })
})

const checkedIds = computed(() =>
  Object.entries(checked.value)
    .filter(([, v]) => v)
    .map(([k]) => k)
    .filter((id) => candidates.value.some((c) => c.id === id)),
)

function toggle(id: string) {
  checked.value = { ...checked.value, [id]: !checked.value[id] }
}

function close() {
  checked.value = {}
  query.value = ''
  emit('close')
}

function confirm() {
  if (!checkedIds.value.length) return
  emit('confirm', checkedIds.value)
  close()
}

function fallbackColor(a: Agent) {
  return skillAvatarColor(String(a.id))
}

function descText(a: Agent) {
  return a.description?.trim() || '暂无简介'
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="mask" @click.self="close">
      <div class="dialog" role="dialog" aria-label="添加智能体">
        <header class="dialog-head">
          <h3>添加智能体到当前工作目录</h3>
          <button type="button" class="close" aria-label="关闭" @click="close">×</button>
        </header>
        <div class="dialog-body">
          <input
            v-model="query"
            type="search"
            class="search"
            placeholder="搜索智能体"
            autocomplete="off"
            aria-label="搜索智能体"
          />
          <div class="list">
            <button
              v-for="a in candidates"
              :key="a.id"
              type="button"
              class="row"
              :class="{ on: checked[a.id] }"
              :aria-pressed="!!checked[a.id]"
              @click="toggle(a.id)"
            >
              <span class="avatar-wrap">
                <img v-if="a.avatar" :src="a.avatar" class="avatar" alt="" />
                <span v-else class="avatar fallback" :style="{ background: fallbackColor(a) }">
                  {{ skillInitial(a.name) }}
                </span>
              </span>
              <span class="meta">
                <strong class="name">{{ a.name }}</strong>
                <span class="sign">{{ descText(a) }}</span>
              </span>
              <span class="checkbox" aria-hidden="true">{{ checked[a.id] ? '✓' : '' }}</span>
            </button>
            <p v-if="!candidates.length" class="empty">
              {{ query.trim() ? '无匹配智能体' : '没有可添加的智能体' }}
            </p>
          </div>
        </div>
        <footer class="dialog-foot">
          <span class="hint">已选 {{ checkedIds.length }} 个</span>
          <div class="ops">
            <button type="button" class="btn" @click="close">取消</button>
            <button type="button" class="btn primary" :disabled="!checkedIds.length" @click="confirm">
              <IconPlus />
              添加
            </button>
          </div>
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: grid;
  place-items: center;
  background: rgba(0, 0, 0, 0.4);
}

.dialog {
  display: flex;
  flex-direction: column;
  width: min(420px, 90vw);
  max-height: min(560px, 80vh);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg, var(--radius));
  box-shadow: var(--shadow);
  overflow: hidden;
}

.dialog-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px 10px;
  flex-shrink: 0;
}

.dialog-head h3 {
  margin: 0;
  flex: 1;
  min-width: 0;
  font-size: 15px;
  font-weight: 600;
}

.close {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font-size: 16px;
  cursor: pointer;
}

.close:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.dialog-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0 16px;
  min-height: 0;
  flex: 1;
}

.search {
  width: 100%;
  height: 32px;
  margin: 0;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  color: var(--text);
  font: inherit;
  font-size: 12px;
  outline: none;
  flex-shrink: 0;
}

.search:focus {
  border-color: var(--border-strong, var(--border));
}

.list {
  flex: 1;
  min-height: 120px;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-bottom: 8px;
}

.row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.row:hover {
  background: var(--surface-hover);
}

.row.on {
  background: var(--accent-soft);
}

.avatar-wrap {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
}

.avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
  display: block;
  background: var(--surface);
}

.avatar.fallback {
  display: grid;
  place-items: center;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
}

.meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
}

.row.on .name {
  color: var(--accent);
}

.sign {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  line-height: 16px;
  color: var(--text-dim);
}

.checkbox {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  display: grid;
  place-items: center;
  border: 1px solid var(--border-strong, var(--border));
  border-radius: 4px;
  font-size: 12px;
  color: var(--accent);
}

.row.on .checkbox {
  border-color: var(--accent);
}

.empty {
  margin: 0;
  padding: 24px 8px;
  color: var(--text-dim);
  font-size: 12px;
  text-align: center;
}

.dialog-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px 14px;
  flex-shrink: 0;
}

.hint {
  flex: 1;
  min-width: 0;
  color: var(--text-dim);
  font-size: 12px;
}

.ops {
  display: flex;
  gap: 8px;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}

.btn:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.btn.primary {
  background: var(--accent);
  border-color: transparent;
  color: #fff;
}

.btn.primary:hover:not(:disabled) {
  background: var(--accent-hover);
}

.btn.primary:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.btn svg {
  width: 13px;
  height: 13px;
}
</style>
