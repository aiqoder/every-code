<script setup lang="ts">
defineProps<{
  open?: boolean
}>()
</script>

<template>
  <svg
    viewBox="0 0 16 16"
    class="h-3 w-3 shrink-0 transition-transform"
    :class="open ? 'rotate-90' : ''"
    fill="none"
    aria-hidden="true"
  >
    <path
      d="M6 4.5 9.5 8 6 11.5"
      stroke="currentColor"
      stroke-width="1.6"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
</template>
