import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      component: () => import('@/components/AppLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
        },
        {
          path: 'domains',
          name: 'domains',
          component: () => import('@/views/DomainsView.vue'),
        },
        {
          path: 'domains/:id',
          name: 'domain-detail',
          component: () => import('@/views/DomainDetailView.vue'),
          props: true,
        },
        {
          path: 'databases',
          name: 'databases',
          component: () => import('@/views/DatabasesView.vue'),
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('@/views/UsersView.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'files',
          name: 'files',
          component: () => import('@/views/FileManagerView.vue'),
        },
        {
          path: 'files/:domain',
          name: 'file-manager',
          component: () => import('@/views/FileManagerView.vue'),
          props: true,
        },
        {
          path: 'monitoring',
          name: 'monitoring',
          component: () => import('@/views/MonitoringView.vue'),
        },
        {
          path: 'terminal',
          name: 'terminal',
          component: () => import('@/views/TerminalView.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'backups',
          name: 'backups',
          component: () => import('@/views/BackupsView.vue'),
        },
        {
          path: 'installer',
          name: 'installer',
          component: () => import('@/views/InstallerView.vue'),
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/SettingsView.vue'),
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()

  if (auth.token && !auth.user) {
    await auth.fetchUser()
  }

  if (to.meta.requiresAuth !== false && !auth.isAuthenticated) {
    return { name: 'login' }
  }

  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return { name: 'dashboard' }
  }

  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }
})

export default router
