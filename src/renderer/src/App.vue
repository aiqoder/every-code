<script setup lang="ts">
import {
  NConfigProvider,
  NDialogProvider,
  NMessageProvider,
  NNotificationProvider,
  darkTheme,
  dateZhCN,
  zhCN
} from 'naive-ui'
import UpdateNotifier from '@/components/UpdateNotifier.vue'
import { useThemeStore } from '@/stores/themeStore'

const themeStore = useThemeStore()
</script>

<template>
  <n-config-provider
    class="h-full min-h-0"
    :locale="zhCN"
    :date-locale="dateZhCN"
    :theme="themeStore.isDark ? darkTheme : null"
    :theme-overrides="themeStore.overrides"
  >
    <n-message-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <div class="h-full min-h-0">
            <router-view />
            <UpdateNotifier />
          </div>
        </n-notification-provider>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<style>
.n-config-provider {
  height: 100%;
  min-height: 0;
}
</style>
