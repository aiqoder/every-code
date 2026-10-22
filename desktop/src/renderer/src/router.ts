import { createRouter, createWebHashHistory } from 'vue-router'
import { useSessionStore } from './stores/session'
import LoginView from './views/login/LoginView.vue'
import ProjectView from './views/project/ProjectView.vue'
import CreateView from './views/create/CreateView.vue'
import ExecuteView from './views/execute/ExecuteView.vue'
import DeployView from './views/deploy/DeployView.vue'
import TaskPlanView from './views/taskplan/TaskPlanView.vue'
import SettingsView from './views/settings/SettingsView.vue'
import ModelsView from './views/models/ModelsView.vue'
import HostsView from './views/hosts/HostsView.vue'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/project' },
    { path: '/login', component: LoginView, meta: { bare: true } },
    { path: '/project', component: ProjectView },
    { path: '/create', component: CreateView },
    { path: '/execute', component: ExecuteView },
    { path: '/deploy', component: DeployView },
    { path: '/tasks/:id', component: TaskPlanView },
    { path: '/models', component: ModelsView },
    { path: '/hosts', component: HostsView },
    { path: '/settings', component: SettingsView }
  ]
})

router.beforeEach(async (to) => {
  const session = useSessionStore()
  try {
    await session.bootstrap()
  } catch {
    if (to.path === '/login') return true
    return { path: '/login' }
  }

  if (!session.loggedIn) {
    if (to.path === '/login') return true
    const redirect = to.fullPath && to.fullPath !== '/' ? to.fullPath : ''
    return redirect ? { path: '/login', query: { redirect } } : { path: '/login' }
  }

  if (to.path === '/login') return session.postLoginPath(typeof to.query.redirect === 'string' ? to.query.redirect : '')

  if (session.modelGateKnown && !session.modelsConfigured && to.path !== '/models') return '/models'
  return true
})

router.afterEach((to) => {
  const titles: Record<string, string> = {
    '/project': '项目概况',
    '/create': '任务',
    '/execute': '执行',
    '/deploy': '部署',
    '/models': '模型',
    '/hosts': '主机',
    '/settings': '设置',
    '/login': '登录'
  }
  const key = Object.keys(titles).find((k) => to.path.startsWith(k))
  document.title = key ? `ADE · ${titles[key]}` : 'ADE'
})
