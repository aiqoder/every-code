<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import type { EditorState } from '@tiptap/pm/state'
import type { SkillRef } from '../../../lib/skillToken'
import { SkillChip } from '../skill/SkillChip'
import { docToSkillContent, skillContentToDoc } from '../skill/serialize'

const props = defineProps<{
  disabled?: boolean
  placeholder?: string
  nameById?: Map<string, string>
}>()

const emit = defineEmits<{
  send: []
  openSkillPicker: []
  updateEmpty: [empty: boolean]
  updateContent: [content: string]
}>()

const empty = ref(true)

/** 段首、空白后、或 skill chip 后允许唤起技能菜单。 */
function shouldOpenSkillMenu(state: EditorState, pos: number): boolean {
  const $pos = state.doc.resolve(pos)
  if ($pos.parentOffset === 0) return true
  const prev = state.doc.textBetween(pos - 1, pos, '\n', '\n')
  if (!prev || /\s/.test(prev)) return true
  const nodeBefore = $pos.nodeBefore
  if (nodeBefore?.type.name === 'skillChip') return true
  return false
}

const editor = useEditor({
  extensions: [
    StarterKit.configure({
      heading: false,
      bold: false,
      italic: false,
      strike: false,
      code: false,
      codeBlock: false,
      blockquote: false,
      horizontalRule: false,
      bulletList: false,
      orderedList: false,
      listItem: false,
      dropcursor: false,
      gapcursor: false,
    }),
    SkillChip,
  ],
  editorProps: {
    attributes: {
      class: 'composer-editor',
    },
    handleTextInput(view, from, _to, text) {
      if (text !== '/') return false
      if (!shouldOpenSkillMenu(view.state, from)) return false
      emit('openSkillPicker')
      return true
    },
    handleKeyDown: (view, event) => {
      // 部分键盘/布局下 key 为 Slash，仍以字符插入为准；此处兜底拦截
      if (!event.isComposing && (event.key === '/' || (event.code === 'Slash' && !event.shiftKey && !event.altKey))) {
        const { from } = view.state.selection
        if (shouldOpenSkillMenu(view.state, from)) {
          event.preventDefault()
          emit('openSkillPicker')
          return true
        }
      }
      if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
        event.preventDefault()
        emit('send')
        return true
      }
      return false
    },
  },
  onUpdate: ({ editor: ed }) => {
    empty.value = ed.isEmpty
    emit('updateEmpty', ed.isEmpty)
    emit('updateContent', docToSkillContent(ed.getJSON()))

    // 兜底：若 / 已写入文档（部分输入法会绕过 key/textInput），删掉并唤起面板
    const { state } = ed
    const { from, empty: selEmpty } = state.selection
    if (!selEmpty || from < 1) return
    if (state.doc.textBetween(from - 1, from) !== '/') return
    if (!shouldOpenSkillMenu(state, from - 1)) return
    ed.commands.deleteRange({ from: from - 1, to: from })
    emit('openSkillPicker')
  },
  onCreate: ({ editor: ed }) => {
    empty.value = ed.isEmpty
    emit('updateEmpty', ed.isEmpty)
    emit('updateContent', docToSkillContent(ed.getJSON()))
  },
})

watch(
  () => props.disabled,
  (d) => {
    editor.value?.setEditable(!d)
  },
)

function getContent(): string {
  if (!editor.value) return ''
  return docToSkillContent(editor.value.getJSON())
}

function setContent(text: string) {
  if (!editor.value) return
  const map = props.nameById || new Map<string, string>()
  editor.value.commands.setContent(skillContentToDoc(text, map))
  empty.value = editor.value.isEmpty
  emit('updateEmpty', editor.value.isEmpty)
  emit('updateContent', getContent())
}

function clear() {
  editor.value?.commands.clearContent(true)
  empty.value = true
  emit('updateEmpty', true)
  emit('updateContent', '')
}

function focus() {
  editor.value?.commands.focus('end')
}

function insertSkill(skill: SkillRef) {
  if (!editor.value) return
  editor.value
    .chain()
    .focus()
    .insertSkillChip({ id: skill.id, name: skill.name })
    .insertContent(' ')
    .run()
  empty.value = editor.value.isEmpty
  emit('updateEmpty', editor.value.isEmpty)
  emit('updateContent', getContent())
}

onBeforeUnmount(() => {
  editor.value?.destroy()
})

defineExpose({ getContent, setContent, clear, focus, insertSkill, empty: computed(() => empty.value) })
</script>

<template>
  <div class="composer-tiptap-wrap">
    <EditorContent v-if="!disabled" :editor="editor" class="composer-tiptap" />
    <span v-if="!disabled && empty" class="ph">{{ placeholder || '输入消息…' }}</span>
  </div>
</template>

<style scoped>
.composer-tiptap-wrap {
  position: relative;
  width: 100%;
}

.ph {
  position: absolute;
  left: 14px;
  top: 12px;
  color: var(--text-dim);
  font-size: 14px;
  font-weight: 400;
  line-height: 22px;
  pointer-events: none;
}

.composer-tiptap {
  width: 100%;
}

.composer-tiptap :deep(.composer-editor) {
  min-height: 52px;
  max-height: 180px;
  overflow: auto;
  padding: 12px 14px 8px;
  outline: none;
  font: inherit;
  font-size: 14px;
  font-weight: 400;
  line-height: 22px;
  color: var(--text);
}

.composer-tiptap :deep(.composer-editor p) {
  margin: 0;
}

.composer-tiptap :deep(.composer-editor p + p) {
  margin-top: 0.35em;
}

.composer-tiptap :deep(.skill-chip) {
  display: inline-flex;
  align-items: center;
  margin: 0 2px;
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  vertical-align: baseline;
  user-select: none;
}
</style>
