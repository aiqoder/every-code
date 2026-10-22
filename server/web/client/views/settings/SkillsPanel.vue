<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useClientPort } from '../../ports/clientPort'
import type { SkillItem } from '../../types'
import { skillDescription } from '../../lib/skillFrontmatter'
import { skillAvatarColor, skillInitial } from '../../lib/skillAvatar'
import HoverTip from '../chat/components/HoverTip.vue'
import { useCapabilityBar } from './capabilityBar'

const router = useRouter()
const port = useClientPort()
const bar = useCapabilityBar()
const list = ref<SkillItem[]>([])
const err = ref('')

const shown = computed(() => {
  const q = bar.query.trim().toLowerCase()
  return list.value.filter((item) => {
    if (bar.installedOnly && item.source !== 'local') return false
    if (!q) return true
    const desc = skillDescription(item).toLowerCase()
    return item.name.toLowerCase().includes(q) || item.id.toLowerCase().includes(q) || desc.includes(q)
  })
})

watch(
  list,
  (items) => {
    bar.installedCount = items.filter((item) => item.source === 'local').length
  },
  { immediate: true },
)

async function load() {
  err.value = ''
  try {
    list.value = await port.listSkills()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

function create() {
  router.push({ name: 'settings-skill-new', query: router.currentRoute.value.query })
}

function skillSlug(item: SkillItem) {
  return item.id.replace(/^(local|builtin|claude|project):/, '')
}

function isEditable(item: SkillItem) {
  return item.editable !== false && item.source === 'local'
}

function open(item: SkillItem) {
  router.push({
    name: 'settings-skill-edit',
    params: { slug: skillSlug(item) },
    query: { ...router.currentRoute.value.query, src: item.source || 'local' },
  })
}

async function remove(item: SkillItem) {
  if (!isEditable(item)) return
  if (!confirm(`删除技能「${item.name}」？`)) return
  err.value = ''
  try {
    list.value = await port.deleteSkill(item.id)
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  }
}

function scopeLabel(item: SkillItem) {
  if (item.source === 'builtin') return '内置'
  if (item.source === 'claude') return 'Claude Code'
  if (item.source === 'project') return '项目'
  if (item.source === 'pkg') return '智能体包'
  return '本地'
}

function descText(item: SkillItem) {
  return skillDescription(item)
}

onMounted(() => {
  bar.placeholder = '搜索技能'
  bar.addLabel = '添加技能'
  bar.onAdd = create
  load()
})

onUnmounted(() => {
  bar.onAdd = null
})
</script>

<template>
  <div class="pane">
    <p v-if="err" class="err">{{ err }}</p>

    <div v-if="shown.length" class="grid">
      <article v-for="s in shown" :key="s.id" class="card">
        <div class="card-top">
          <span class="avatar" :style="{ background: skillAvatarColor(s.id) }">
            {{ skillInitial(s.name) }}
          </span>
          <div class="title-wrap">
            <strong class="name" :title="s.name">{{ s.name }}</strong>
            <em>{{ scopeLabel(s) }}</em>
          </div>
        </div>

        <small class="meta">{{ s.id }}</small>

        <HoverTip :content="descText(s)" :disabled="!descText(s)">
          <p class="desc">{{ descText(s) || '暂无简介' }}</p>
        </HoverTip>

        <div class="ops">
          <template v-if="isEditable(s)">
            <button type="button" @click="open(s)">编辑</button>
            <button type="button" class="danger" @click="remove(s)">删除</button>
          </template>
          <template v-else>
            <button type="button" @click="open(s)">查看</button>
            <span class="readonly">只读</span>
          </template>
        </div>
      </article>
    </div>
    <p v-else class="empty">{{ list.length ? '没有匹配的技能' : '还没有技能' }}</p>
  </div>
</template>

<style scoped>
.pane {
  width: 100%;
  max-width: 960px;
  margin: 0 auto;
  align-self: center;
  padding: 20px 24px 28px;
}

.pane-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
}

h2 {
  font-family: var(--display);
  font-weight: 600;
  font-size: 24px;
  margin: 0;
  letter-spacing: -0.02em;
}

.pane-head p {
  margin: 6px 0 0;
  color: var(--text-muted);
  font-size: 13px;
}

.add {
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  padding: 8px 16px;
  background: var(--accent);
  border: 0;
  color: #fff;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.add:hover {
  background: var(--accent-hover);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
}

.card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 16px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  transition: border-color 0.12s;
}

.card:hover {
  border-color: var(--border-strong);
}

.card-top {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.avatar {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  color: #fff;
  font-size: 16px;
  font-weight: 700;
}

.title-wrap {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.name {
  font-size: 14px;
  font-weight: 560;
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

em {
  align-self: flex-start;
  font-style: normal;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-dim);
  padding: 1px 7px;
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
}

.meta {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.desc {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.45;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  word-break: break-word;
  min-height: calc(1.45em * 2);
}

.ops {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-top: auto;
  padding-top: 2px;
}

.ops button {
  font: inherit;
  font-size: 12px;
  padding: 5px 10px;
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text-muted);
  border-radius: var(--radius-sm);
}

.ops button:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.ops button.danger:hover {
  color: var(--error);
  border-color: rgba(248, 113, 113, 0.3);
}

.readonly {
  font-size: 12px;
  color: var(--text-dim);
}

.empty {
  margin: 24px 0 0;
  color: var(--text-dim);
  font-size: 14px;
  text-align: center;
}

.err {
  color: var(--error);
  margin: 0 0 12px;
  font-size: 13px;
}
</style>
