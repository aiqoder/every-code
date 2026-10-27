import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      component: () => import('@/layouts/AppLayout.vue'),
      children: [
        {
          path: '',
          name: 'chat',
          component: () => import('@/views/chat/ChatView.vue')
        },
        {
          path: 'agents',
          name: 'agents',
          component: () => import('@/views/agents/AgentsView.vue')
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/settings/SettingsView.vue')
        }
      ]
    }
  ]
})

export default router
