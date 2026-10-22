import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'

import App from './App.vue'
import router from './router'
import './assets/main.css'

import { useThemeStore } from '@client/stores/theme'

const app = createApp(App)

app.use(createPinia())
useThemeStore().init()
app.use(router)
app.use(ElementPlus, { locale: zhCn })

app.mount('#app')
