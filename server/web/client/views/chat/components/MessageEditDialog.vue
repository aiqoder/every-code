<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { SkillRef } from '../../../lib/skillToken'
import ChatComposer from './ChatComposer.vue'
import SkillPicker from './SkillPicker.vue'

const props = defineProps<{
  open: boolean
  content: string
  skills: SkillRef[]
  nameById?: Map<string, string>
}>()

const emit = defineEmits<{
  close: []
  send: [text: string]
}>()

const composerRef = ref<{
  getContent: () => string
  setContent: (text: string) => void
  focus: () => void
  insertSkill: (skill: SkillRef) => void
} | null>(null)
const skillOpen = ref(false)
const empty = ref(true)

watch(
  () => props.open,
  async (open) => {
    skillOpen.value = false
    if (!open) return
    await nextTick()
    composerRef.value?.setContent(props.content)
    composerRef.value?.focus()
  },
)

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (skillOpen.value) {
      skillOpen.value = false
      return
    }
    emit('close')
  }
}

function onSend() {
  const text = composerRef.value?.getContent().trim() || ''
  if (!text) return
  emit('send', text)
}

function onSelectSkill(skill: SkillRef) {
  skillOpen.value = false
  composerRef.value?.insertSkill(skill)
  composerRef.value?.focus()
}
</script>

<template>
  <div v-if="open" class="overlay" @keydown="onKeydown">
    <button type="button" class="backdrop" aria-label="关闭" @click="emit('close')" />
    <div class="panel" role="dialog" aria-modal="true" aria-label="编辑消息">
      <header>
        <strong>编辑消息</strong>
        <button type="button" class="close" aria-label="关闭" @click="emit('close')">×</button>
      </header>
      <div class="composer composer-dialog">
        <div class="composer-wrap">
          <ChatComposer
            ref="composerRef"
            :name-by-id="nameById"
            placeholder="输入消息，或按 / 选择技能…"
            @send="onSend"
            @open-skill-picker="skillOpen = true"
            @update-empty="empty = $event"
          />
          <SkillPicker
            :open="skillOpen"
            :skills="skills"
            @select="onSelectSkill"
            @close="skillOpen = false"
          />
        </div>
      </div>
      <footer>
        <button type="button" class="btn ghost" @click="emit('close')">取消</button>
        <button type="button" class="btn primary" :disabled="empty" @click="onSend">发送</button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: grid;
  place-items: center;
  padding: 24px;
}

.backdrop {
  position: absolute;
  inset: 0;
  border: 0;
  background: rgba(0, 0, 0, 0.45);
  cursor: default;
}

.panel {
  position: relative;
  z-index: 1;
  width: min(640px, 100%);
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-elevated);
  box-shadow: var(--shadow);
}

header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

header strong {
  font-size: 15px;
}

.close {
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font-size: 20px;
  line-height: 1;
}

.close:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.composer {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  overflow: visible;
}

.composer-wrap {
  position: relative;
}

footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
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

.btn.primary:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
</style>
