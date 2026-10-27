<script setup lang="ts">
import { onUnmounted } from 'vue'
import AppHeader from '@/components/AppHeader.vue'
import LoginDialog from '@/components/LoginDialog.vue'
import { startScheduleRunner } from '@/views/schedules/runner'

const stopScheduleRunner = startScheduleRunner()
onUnmounted(stopScheduleRunner)
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <AppHeader />
    <div class="relative min-h-0 flex-1 overflow-hidden">
      <router-view />
      <LoginDialog />
    </div>
  </div>
</template>
