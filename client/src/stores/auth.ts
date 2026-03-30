import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as authApi from '@/api/auth'
import type { User } from '@/types'
import { clearStoredSession, hasStoredSession, storeTokens } from '@/utils/auth'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const isAuthenticated = ref(hasStoredSession())

  async function login(username: string, password: string) {
    const res = await authApi.login(username, password)
    user.value = res.user
    storeTokens(res.tokens)
    isAuthenticated.value = true
  }

  async function register(username: string, password: string) {
    const res = await authApi.register(username, password)
    user.value = res.user
    storeTokens(res.tokens)
    isAuthenticated.value = true
  }

  function logout() {
    user.value = null
    clearStoredSession()
    isAuthenticated.value = false
  }

  return { user, isAuthenticated, login, register, logout }
})
