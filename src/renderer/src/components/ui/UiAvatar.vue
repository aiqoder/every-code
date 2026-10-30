<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  name: string
  image?: string
}>()

const palette = ['#3557f5', '#0a7a56', '#c2410c', '#6d5bd0', '#e11d48', '#0e7490']

const initial = computed(() => {
  const text = props.name.trim()
  return text ? text.slice(0, 1).toUpperCase() : '?'
})

const color = computed(() => {
  let index = 0
  for (const char of props.name) index = (index + char.charCodeAt(0)) % palette.length
  return palette[index]
})
</script>

<template>
  <img v-if="image" :src="image" alt="" class="ui-avatar" />
  <span v-else class="ui-avatar fallback" :style="{ background: color }">{{ initial }}</span>
</template>

<style scoped>
.ui-avatar {
  display: grid;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  place-items: center;
  overflow: hidden;
  border-radius: 999px;
  object-fit: cover;
}

.fallback {
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  line-height: 1;
}
</style>
