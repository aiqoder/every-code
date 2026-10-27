<script setup lang="ts">
import AppHeader from '@/components/AppHeader.vue'
import LoginDialog from '@/components/LoginDialog.vue'
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
