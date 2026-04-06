import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useServerConfigStore } from '@/stores/serverConfig'
import { hasStoredSession } from '@/utils/auth'

function isTauri(): boolean {
  return typeof (window as Window & { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__ !== 'undefined'
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/setup',
      name: 'setup',
      component: () => import('@/views/SetupView.vue'),
      meta: { public: true },
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      redirect: '/library',
    },
    {
      path: '/library',
      name: 'library',
      component: () => import('@/views/LibraryView.vue'),
    },
    {
      path: '/books/:id',
      name: 'reader',
      component: () => import('@/views/ReaderView.vue'),
    },
    {
      path: '/lists',
      name: 'lists',
      component: () => import('@/views/ListsView.vue'),
    },
  ],
})

// Guard: redirect to /setup if no server configured (Tauri only), then /login if not authenticated
router.beforeEach((to) => {
  const cfg = useServerConfigStore()
  if (isTauri() && !cfg.serverUrl && to.name !== 'setup') {
    return { name: 'setup' }
  }

  const auth = useAuthStore()
  auth.isAuthenticated = hasStoredSession()
  if (!to.meta.public && !auth.isAuthenticated) {
    return { name: 'login' }
  }
})

export default router
