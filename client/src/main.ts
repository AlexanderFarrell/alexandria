import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './style.css'
import { initAuthStorage } from './utils/auth'
import { useServerConfigStore } from './stores/serverConfig'
import { setSessionExpiredHandler } from './api/client'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)

// Initialize storage before mounting — loads tokens and server URL into memory.
// Wrapped in an IIFE because top-level await requires ES2022+ target.
;(async () => {
  await initAuthStorage()
  const serverConfig = useServerConfigStore()
  await serverConfig.load()

  app.use(router)
  setSessionExpiredHandler(() => router.push('/login'))
  app.mount('#app')
})()
