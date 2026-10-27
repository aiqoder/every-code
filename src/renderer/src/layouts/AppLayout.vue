<script setup lang="ts">
import AppHeader from '@/components/AppHeader.vue'
</script>

<template>
  <div class="flex h-full flex-col">
    <AppHeader />
    <router-view />
  </div>
</template>
