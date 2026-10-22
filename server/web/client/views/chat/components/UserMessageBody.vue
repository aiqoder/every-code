<script setup lang="ts">
import MarkdownBody from './MarkdownBody.vue'

defineProps<{
  content: string
  nameById?: Map<string, string> | Record<string, string>
  streaming?: boolean
}>()
</script>

<template>
  <MarkdownBody :content="content" :name-by-id="nameById" :streaming="streaming" breaks />
</template>
