<script setup lang="ts">
import { onUnmounted } from 'vue'
import LoginDialog from '@/components/LoginDialog.vue'
import { startScheduleRunner } from '@/views/schedules/runner'

const stopScheduleRunner = startScheduleRunner()
onUnmounted(stopScheduleRunner)
</script>

<template>
  <div class="relative h-full min-h-0 overflow-hidden">
    <router-view />
    <LoginDialog />
  </div>
</template>
