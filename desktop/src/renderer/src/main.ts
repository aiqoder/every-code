import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import './styles/tokens.css'
import './styles/base.css'
import { setHttpSession } from './api'
import { router } from './router'
import { installPageSurfaceHost } from './surfaces/host'
import { useThemeStore } from './stores/theme'

async function boot(): Promise<void> {
  if (window.ade) {
    try {
      const settings = await window.ade.getSettings()
      setHttpSession({
        baseUrl: settings.profile.baseUrl,
        token: settings.loggedIn ? settings.token : ''
      })
    } catch {
      /* 界面启动时会再取一次 */
    }
  }
  const app = createApp(App)
  app.use(createPinia())
  useThemeStore().init()
  app.use(router)
  installPageSurfaceHost()
  app.mount('#app')
}

void boot()
