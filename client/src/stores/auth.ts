import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as authApi from '@/api/auth'
import type { User, Tokens } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const isAuthenticated = ref(!!localStorage.getItem('access_token'))

  function storeTokens(tokens: Tokens) {
    localStorage.setItem('access_token', tokens.access_token)
    localStorage.setItem('refresh_token', tokens.refresh_token)
    isAuthenticated.value = true
  }

  async function login(username: string, password: string) {
    const res = await authApi.login(username, password)
    user.value = res.user
    storeTokens(res.tokens)
  }

  async function register(username: string, password: string) {
    const res = await authApi.register(username, password)
    user.value = res.user
    storeTokens(res.tokens)
  }

  function logout() {
    user.value = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    isAuthenticated.value = false
  }

  return { user, isAuthenticated, login, register, logout }
})
