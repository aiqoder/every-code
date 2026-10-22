<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useClientPort } from '../../../ports/clientPort'
import type { SkillItem } from '../../../types'
import EditorsBreadcrumb from '../components/EditorsBreadcrumb.vue'

const route = useRoute()
const router = useRouter()
const port = useClientPort()

const rawSlug = Array.isArray(route.params.slug) ? route.params.slug[0] : route.params.slug
const slugParam = rawSlug ? decodeURIComponent(rawSlug) : ''
const editing = !!slugParam
const srcQuery = computed(() => String(route.query.src || ''))

const defaultContent = `---
name: 我的技能
description: 简要说明该技能的用途。
---

# 我的技能

在这里编写技能说明，指导模型如何完成特定任务。
`

const form = reactive({
  slug: editing ? slugParam.replace(/^(local|builtin):/, '') : 'my-skill',
  content: editing ? '' : defaultContent,
})
const readonly = ref(false)
const loading = ref(true)
const err = ref('')
const saving = ref(false)

function matchSkill(list: SkillItem[], slug: string, src: string): SkillItem | undefined {
  const bare = slug.replace(/^(local|builtin|claude|project):/, '')
  const candidates = list.filter(
    (s) =>
      s.id === slug ||
      s.id === `local:${bare}` ||
      s.id === `builtin:${bare}` ||
      s.id === `claude:${bare}` ||
      s.id === `project:${bare}` ||
      s.id.replace(/^(local|builtin|claude|project):/, '') === bare,
  )
  if (!candidates.length) return undefined
  if (src) {
    const hit = candidates.find((s) => s.source === src)
    if (hit) return hit
  }
  return candidates[0]
}

async function load() {
  if (!editing) {
    loading.value = false
    return
  }
  loading.value = true
  err.value = ''
  try {
    const list = await port.listSkills()
    const row = matchSkill(list ?? [], slugParam, srcQuery.value)
    if (row) {
      form.slug = row.id.replace(/^(local|builtin|claude|project):/, '')
      form.content = row.content
      readonly.value = row.editable === false || row.source !== 'local'
    }
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

function goBack() {
  const query = { ...route.query }
  delete query.src
  router.push({ name: 'settings-skills', query })
}

async function submit() {
  if (readonly.value) {
    goBack()
    return
  }
  const slug = form.slug.trim()
  if (!slug) {
    err.value = '请输入 Slug'
    return
  }
  saving.value = true
  err.value = ''
  try {
    await port.saveSkill(slug, form.content)
    goBack()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="editor-page">
    <EditorsBreadcrumb
      :items="[
        { label: '技能', to: { name: 'settings-skills' } },
        { label: readonly ? '查看技能' : editing ? '编辑技能' : '新建技能' },
      ]"
    />

    <form class="dlg" @submit.prevent="submit">
      <p v-if="err" class="err">{{ err }}</p>
          <p v-if="readonly" class="hint">该技能为只读（内置或 Claude Code），仅可查看，不可在此修改。</p>
      <label>
        Slug
        <input v-model="form.slug" :disabled="editing || readonly" required />
      </label>
      <label>
        SKILL.md
        <textarea v-model="form.content" rows="16" required :readonly="readonly" />
      </label>
      <div class="actions">
        <button type="button" @click="goBack">{{ readonly ? '返回' : '取消' }}</button>
        <button v-if="!readonly" type="submit" class="go" :disabled="saving || loading">
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.editor-page {
  width: 100%;
  max-width: 680px;
  margin: 0 auto;
  align-self: center;
  padding: 20px 24px 28px;
}

.dlg {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 24px;
  display: grid;
  gap: 14px;
}

label {
  display: grid;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-muted);
}

input,
textarea {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
  padding: 10px 12px;
  font: 13px/1.5 var(--mono);
}

textarea {
  resize: vertical;
  min-height: 280px;
}

textarea[readonly] {
  opacity: 0.92;
  cursor: default;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 4px;
}

button {
  font: inherit;
  font-size: 13px;
  padding: 8px 16px;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
  border-radius: var(--radius-sm);
}

button:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.go {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}

.go:hover {
  background: var(--accent-hover);
}

.go:disabled {
  opacity: 0.6;
}

.hint {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}

.err {
  margin: 0;
  color: var(--error);
  font-size: 13px;
}
</style>
