import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const title = ref(import.meta.env.VITE_APP_TITLE)
  const apiBaseUrl = ref(import.meta.env.VITE_API_BASE_URL)

  return { title, apiBaseUrl }
})
