<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Compartment, EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers, placeholder as cmPlaceholder } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
import { languages } from '@codemirror/language-data'
import { oneDark } from '@codemirror/theme-one-dark'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    minHeight?: number
    maxHeight?: number
    readonly?: boolean
  }>(),
  {
    modelValue: '',
    placeholder: '定义智能体的角色、约束与行为',
    minHeight: 180,
    maxHeight: 520,
    readonly: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const host = ref<HTMLDivElement>()
const readonlyCompartment = new Compartment()
let view: EditorView | null = null

function buildTheme() {
  return EditorView.theme({
    '&': {
      fontSize: '13px',
      backgroundColor: 'transparent',
      color: 'var(--text)',
    },
    '&.cm-focused': {
      outline: 'none',
    },
    '.cm-scroller': {
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
      lineHeight: '1.65',
    },
    '.cm-content': {
      padding: '10px 0',
    },
    '.cm-gutters': {
      backgroundColor: 'transparent',
      color: 'var(--text-dim)',
      border: 'none',
      borderRight: '1px solid var(--border)',
    },
    '.cm-activeLine': {
      backgroundColor: 'var(--surface)',
    },
    '.cm-activeLineGutter': {
      backgroundColor: 'transparent',
      color: 'var(--text-muted)',
    },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
      backgroundColor: 'var(--accent-soft)',
    },
    '.cm-cursor': {
      borderLeftColor: 'var(--accent)',
    },
  })
}

function applyHeight() {
  if (!host.value) return
  host.value.style.setProperty('--cm-min-height', `${props.minHeight}px`)
  host.value.style.setProperty('--cm-max-height', `${props.maxHeight}px`)
}

function createEditor() {
  if (!host.value) return
  applyHeight()
  view = new EditorView({
    parent: host.value,
    state: EditorState.create({
      doc: props.modelValue ?? '',
      extensions: [
        lineNumbers(),
        history(),
        markdown({ base: markdownLanguage, codeLanguages: languages }),
        EditorView.lineWrapping,
        cmPlaceholder(props.placeholder),
        keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
        oneDark,
        buildTheme(),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            emit('update:modelValue', update.state.doc.toString())
          }
        }),
        readonlyCompartment.of([
          EditorState.readOnly.of(props.readonly),
          EditorView.editable.of(!props.readonly),
        ]),
      ],
    }),
  })
}

watch(
  () => props.modelValue,
  (value) => {
    if (!view) return
    const current = view.state.doc.toString()
    if (value === current) return
    const insert = value ?? ''
    view.dispatch({
      changes: { from: 0, to: current.length, insert },
    })
  },
)

watch(
  () => props.readonly,
  (value) => {
    if (!view) return
    view.dispatch({
      effects: readonlyCompartment.reconfigure([
        EditorState.readOnly.of(value),
        EditorView.editable.of(!value),
      ]),
    })
  },
)

watch(
  () => [props.minHeight, props.maxHeight],
  () => applyHeight(),
)

onMounted(createEditor)

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})
</script>

<template>
  <div ref="host" class="markdown-editor" />
</template>

<style scoped>
.markdown-editor {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  overflow: hidden;
  transition: border-color 0.15s;
}

.markdown-editor:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.markdown-editor :deep(.cm-editor) {
  min-height: var(--cm-min-height, 180px);
  max-height: var(--cm-max-height, 520px);
}

.markdown-editor :deep(.cm-scroller) {
  min-height: var(--cm-min-height, 180px);
  max-height: var(--cm-max-height, 520px);
  overflow: auto;
}

.markdown-editor :deep(.cm-editor.cm-focused) {
  outline: none;
}
</style>
