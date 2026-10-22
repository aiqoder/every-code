<script setup lang="ts">
import { computed } from 'vue'

import { isLogoImage } from '../lib/brand'

const props = defineProps<{
  logo: string
  alt?: string
}>()

const image = computed(() => isLogoImage(props.logo))
</script>

<template>
  <span class="brand-logo">
    <img v-if="image" :src="logo" :alt="alt || ''" />
    <template v-else>{{ logo }}</template>
  </span>
</template>

<style scoped>
.brand-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
  flex-shrink: 0;
}

.brand-logo img {
  height: 1.15em;
  width: auto;
  max-width: 6em;
  object-fit: contain;
  display: block;
}
</style>
