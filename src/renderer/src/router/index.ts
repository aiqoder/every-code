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
          component: () => import('@/views/chat/ChatView.vue'),
          children: [
            {
              path: '',
              name: 'chat',
              component: { render: () => null }
            },
            {
              path: 'schedules',
              name: 'schedules',
              component: () => import('@/views/schedules/ScheduleView.vue')
            },
            {
              path: 'schedules/:id',
              name: 'schedule-detail',
              component: () => import('@/views/schedules/ScheduleDetailView.vue')
            },
            {
              path: 'agents',
              name: 'agents',
              component: () => import('@/views/agents/AgentsView.vue')
            },
            {
              path: 'settings',
              component: () => import('@/views/settings/SettingsView.vue'),
              children: [
                {
                  path: '',
                  name: 'settings',
                  component: () => import('@/views/settings/GeneralPanel.vue')
                },
                {
                  path: 'models',
                  name: 'settings-models',
                  component: () => import('@/views/settings/ModelsPanel.vue')
                }
              ]
            }
          ]
        }
      ]
    }
  ]
})

export default router
