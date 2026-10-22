import { createRouter, createWebHistory } from 'vue-router'

import AdminLayout from '@/layouts/AdminLayout.vue'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', redirect: '/admin/dashboard' },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/login.vue'),
      meta: { title: '登录', public: true, guestOnly: true },
    },
    {
      path: '/oauth/callback',
      name: 'oauth-callback',
      component: () => import('@/views/auth/oauth-callback.vue'),
      meta: { title: '授权登录', public: true },
    },
    {
      path: '/admin',
      component: AdminLayout,
      redirect: '/admin/dashboard',
      meta: { requiresAuth: true, requiresAdmin: true },
      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/dashboard/index.vue'),
          meta: { title: '概览', requiresAuth: true, requiresAdmin: true },
        },
        {
          path: 'llm-providers',
          name: 'llm-providers',
          component: () => import('@/views/llm-provider/index.vue'),
          meta: { title: '大模型供应商', requiresAuth: true, requiresAdmin: true },
        },
        {
          path: 'llm-models',
          name: 'llm-models',
          component: () => import('@/views/llm-model/index.vue'),
          meta: { title: '大模型维护', requiresAuth: true, requiresAdmin: true },
        },
        {
          path: 'system-skills',
          name: 'system-skills',
          component: () => import('@/views/admin/system-skills/index.vue'),
          meta: { title: '系统技能', requiresAuth: true, requiresAdmin: true },
        },
        {
          path: 'scenes',
          name: 'scenes',
          component: () => import('@/views/admin/scenes/index.vue'),
          meta: { title: '场景', requiresAuth: true, requiresAdmin: true },
        },
        {
          path: 'tools',
          name: 'admin-tools',
          component: () => import('@/views/admin/tool/index.vue'),
          meta: { title: '模型工具', requiresAuth: true, requiresAdmin: true },
        },
        {
          path: 'conversations',
          name: 'conversations',
          component: () => import('@/views/conversation/index.vue'),
          meta: { title: '会话查看', requiresAuth: true, requiresAdmin: true },
        },
        {
          path: 'conversations/:id',
          name: 'conversation-detail',
          component: () => import('@/views/conversation/detail.vue'),
          meta: { title: '会话详情', requiresAuth: true, requiresAdmin: true },
        },
      ],
    },
    {
      path: '/guest',
      name: 'guest',
      component: () => import('@/views/guest/index.vue'),
      meta: { requiresAuth: true, title: '桌面端' },
    },
    { path: '/conversations', redirect: '/admin/conversations' },
    { path: '/llm-providers', redirect: '/admin/llm-providers' },
    { path: '/llm-models', redirect: '/admin/llm-models' },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (auth.token && !auth.user) {
    await auth.loadMe()
  }

  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (to.meta.guestOnly && auth.isLoggedIn) {
    const redirect = typeof to.query.redirect === 'string' ? to.query.redirect : ''
    if (redirect.startsWith('/') && !redirect.startsWith('/login')) return redirect
    return auth.isAdmin ? '/admin/dashboard' : '/guest'
  }

  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return '/guest'
  }

  return true
})

router.afterEach((to) => {
  const appTitle = import.meta.env.VITE_APP_TITLE || '橙子社区'
  const pageTitle = (to.meta.title as string) || ''
  document.title = pageTitle ? `${pageTitle} · ${appTitle}` : appTitle
})

export default router
