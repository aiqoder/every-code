<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Compartment, EditorSelection, EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers, placeholder as cmPlaceholder } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
import { languages } from '@codemirror/language-data'
import { bracketMatching, indentOnInput, syntaxHighlighting, defaultHighlightStyle } from '@codemirror/language'

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
      border: '1px solid var(--el-border-color)',
      borderRadius: 'var(--el-border-radius-base)',
      backgroundColor: 'var(--el-fill-color-blank)',
      color: 'var(--el-text-color-primary)',
    },
    '&.cm-focused': {
      outline: 'none',
      borderColor: 'var(--el-color-primary)',
      boxShadow: '0 0 0 1px var(--el-color-primary) inset',
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
      backgroundColor: 'var(--el-fill-color-light)',
      color: 'var(--el-text-color-placeholder)',
      border: 'none',
      borderRight: '1px solid var(--el-border-color-lighter)',
    },
    '.cm-activeLine': {
      backgroundColor: 'var(--el-fill-color-lighter)',
    },
    '.cm-activeLineGutter': {
      backgroundColor: 'var(--el-fill-color)',
    },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
      backgroundColor: 'var(--el-color-primary-light-7)',
    },
    '.cm-cursor': {
      borderLeftColor: 'var(--el-text-color-primary)',
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
        indentOnInput(),
        bracketMatching(),
        syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
        markdown({ base: markdownLanguage, codeLanguages: languages }),
        EditorView.lineWrapping,
        cmPlaceholder(props.placeholder),
        keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
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
      selection: EditorSelection.cursor(insert.length),
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
</style>
