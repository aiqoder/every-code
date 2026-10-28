<script setup lang="ts">
import FeedbackHost from '@/components/ui/FeedbackHost.vue'
import UpdateNotifier from '@/components/UpdateNotifier.vue'
import { useThemeStore } from '@/stores/themeStore'

useThemeStore()
</script>

<template>
  <div class="h-full min-h-0">
    <router-view />
    <UpdateNotifier />
    <FeedbackHost />
  </div>
</template>
