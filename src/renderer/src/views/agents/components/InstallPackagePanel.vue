<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { NButton } from 'naive-ui'

const props = defineProps<{
  show: boolean
  busy: boolean
  error: string
}>()

const emit = defineEmits<{
  close: []
  files: [files: File[]]
}>()

const dialogRef = ref<HTMLElement | null>(null)
const zoneRef = ref<HTMLElement | null>(null)
const dragging = ref(false)

watch(
  () => props.show,
  async (open) => {
    dragging.value = false
    if (!open) return
    await nextTick()
    dialogRef.value?.focus()
  }
)

function requestClose(): void {
  if (props.busy) return
  emit('close')
}

function onDragEnter(event: DragEvent): void {
  event.preventDefault()
  if (props.busy) return
  dragging.value = true
}

function onDragOver(event: DragEvent): void {
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = props.busy ? 'none' : 'copy'
}

function onDragLeave(event: DragEvent): void {
  const next = event.relatedTarget
  if (next instanceof Node && zoneRef.value?.contains(next)) return
  dragging.value = false
}

function onDrop(event: DragEvent): void {
  event.preventDefault()
  event.stopPropagation()
  dragging.value = false
  if (props.busy) return
  const files = event.dataTransfer?.files
  if (files?.length) emit('files', [...files])
}

function onPick(event: Event): void {
  const input = event.target as HTMLInputElement
  const files = input.files ? [...input.files] : []
  input.value = ''
  if (files.length) emit('files', files)
}
</script>

<template>
  <div
    v-if="show"
    class="absolute inset-0 z-30 flex items-center justify-center bg-[var(--ec-overlay)] px-6"
    role="presentation"
    @click.self="requestClose"
    @dragover.prevent
    @drop.prevent
  >
    <div
      ref="dialogRef"
      class="ec-skin-raised w-full max-w-md rounded-2xl border border-[var(--ec-border)] px-6 py-6 text-[var(--ec-fg)] shadow-[0_24px_64px_rgba(0,0,0,0.28)] outline-none"
      role="dialog"
      aria-modal="true"
      aria-labelledby="install-package-title"
      tabindex="-1"
      @keydown.esc.prevent="requestClose"
    >
      <h2 id="install-package-title" class="text-base font-medium">安装智能体</h2>
      <p class="mt-1 text-sm text-[var(--ec-fg-muted)]">支持 .cpk 和 .agent</p>
      <label
        ref="zoneRef"
        class="mt-4 flex min-h-44 cursor-pointer flex-col items-center justify-center rounded-xl border border-dashed px-6 py-8 text-center"
        :class="
          dragging
            ? 'border-[var(--ec-accent)] bg-[color-mix(in_srgb,var(--ec-accent)_14%,transparent)]'
            : 'border-[var(--ec-border)] bg-[var(--ec-canvas)] hover:border-[var(--ec-accent)]'
        "
        @dragenter="onDragEnter"
        @dragover="onDragOver"
        @dragleave="onDragLeave"
        @drop="onDrop"
      >
        <input
          type="file"
          accept=".cpk,.agent"
          class="sr-only"
          aria-label="选择 .cpk 或 .agent 文件"
          :disabled="busy"
          @change="onPick"
        />
        <span class="text-sm">
          {{ busy ? '正在安装…' : dragging ? '松开即可安装' : '把文件拖到这里' }}
        </span>
        <span class="mt-2 text-xs text-[var(--ec-fg-muted)]">或点击选择</span>
      </label>
      <p v-if="error" class="mt-3 text-sm leading-6 text-[var(--ec-danger)]">{{ error }}</p>
      <div class="mt-5 flex justify-end">
        <n-button :disabled="busy" @click="requestClose">取消</n-button>
      </div>
    </div>
  </div>
</template>
