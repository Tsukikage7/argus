import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import AppLayout from '@/components/Layout/AppLayout.vue'

const routes: RouteRecordRaw[] = [
  // 主布局（侧边栏 + 顶栏）
  {
    path: '/',
    component: AppLayout,
    children: [
      { path: '', redirect: '/dashboard' },
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('@/views/DashboardView.vue'),
      },
      {
        path: 'topology',
        name: 'topology',
        component: () => import('@/views/TopologyView.vue'),
      },
      {
        path: 'logs',
        name: 'logs',
        component: () => import('@/views/LogExplorerView.vue'),
      },
      {
        path: 'traces',
        name: 'traces',
        component: () => import('@/views/TracesView.vue'),
      },
      {
        path: 'traces/:uuid',
        name: 'trace-detail',
        component: () => import('@/views/TraceDetailView.vue'),
        props: true,
      },
      {
        path: 'alerts',
        name: 'alerts',
        component: () => import('@/views/AlertsView.vue'),
      },
      {
        path: 'diagnose',
        redirect: (to) => ({ path: '/chat', query: to.query }),
      },
      {
        path: 'chat',
        name: 'chat',
        component: () => import('@/views/AgentChatView.vue'),
      },
      {
        path: 'chat/:sessionId',
        name: 'chat-session',
        component: () => import('@/views/AgentChatView.vue'),
        props: true,
      },
      {
        path: 'replay',
        name: 'replay',
        component: () => import('@/views/ReplayView.vue'),
      },
      {
        path: 'tasks',
        name: 'tasks',
        component: () => import('@/views/TaskListView.vue'),
      },
      {
        path: 'tasks/:id',
        name: 'task-detail',
        component: () => import('@/views/TaskDetailView.vue'),
        props: true,
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/views/SettingsView.vue'),
      },
      {
        path: 'capability',
        name: 'capability',
        component: () => import('@/views/AgentCapabilityView.vue'),
      },
    ],
  },
  // 管理控制台登录
  {
    path: '/admin/login',
    name: 'admin-login',
    component: () => import('@/views/admin/AdminLogin.vue'),
  },
  // 管理控制台
  {
    path: '/admin',
    component: () => import('@/views/admin/AdminLayout.vue'),
    meta: { requiresAdmin: true },
    children: [
      {
        path: '',
        redirect: '/admin/tenants',
      },
      {
        path: 'tenants',
        name: 'admin-tenants',
        component: () => import('@/views/admin/TenantList.vue'),
      },
      {
        path: 'tenants/:id',
        name: 'admin-tenant-detail',
        component: () => import('@/views/admin/TenantDetail.vue'),
        props: true,
      },
      {
        path: 'integration',
        name: 'admin-integration',
        component: () => import('@/views/admin/IntegrationGuide.vue'),
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫：管理页面需要 AdminKey
router.beforeEach((to) => {
  if (to.matched.some(r => r.meta.requiresAdmin)) {
    const adminKey = localStorage.getItem('argus-admin-key')
    if (!adminKey) {
      return { name: 'admin-login', query: { redirect: to.fullPath } }
    }
  }
})

export default router
