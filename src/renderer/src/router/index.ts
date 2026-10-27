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
        }
      ]
    }
  ]
})

export default router
