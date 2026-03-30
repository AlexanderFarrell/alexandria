import axios, { AxiosHeaders, type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import type { Tokens } from '@/types'
import { clearStoredSession, getAccessToken, getRefreshToken, storeTokens } from '@/utils/auth'

const baseConfig = {
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
}

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

const client = axios.create(baseConfig)
const refreshClient = axios.create(baseConfig)
let refreshPromise: Promise<string> | null = null

function setAuthorizationHeader(config: InternalAxiosRequestConfig, token: string) {
  const value = `Bearer ${token}`
  if (config.headers && typeof config.headers.set === 'function') {
    config.headers.set('Authorization', value)
    return
  }
  const headers = AxiosHeaders.from(config.headers ?? {})
  headers.set('Authorization', value)
  config.headers = headers
}

function isPublicAuthRequest(url?: string): boolean {
  return url === '/auth/login' || url === '/auth/register' || url === '/auth/status'
}

function isRefreshRequest(url?: string): boolean {
  return url === '/auth/refresh'
}

function clearSessionAndRedirect() {
  clearStoredSession()
  if (window.location.pathname !== '/login') {
    window.location.href = '/login'
  }
}

async function requestTokenRefresh(): Promise<string> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) {
    throw new Error('missing refresh token')
  }

  const { data } = await refreshClient.post<{ tokens: Tokens }>('/auth/refresh', {
    refresh_token: refreshToken,
  })
  storeTokens(data.tokens)
  return data.tokens.access_token
}

// Attach access token to every request
client.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) {
    setAuthorizationHeader(config, token)
  }
  return config
})

// Refresh expired access tokens once, then retry the failed request.
client.interceptors.response.use(
  (res) => res,
  async (err: AxiosError) => {
    const originalRequest = err.config as RetryableRequestConfig | undefined
    const status = err.response?.status
    const url = originalRequest?.url

    if (status !== 401 || !originalRequest || isPublicAuthRequest(url)) {
      return Promise.reject(err)
    }

    if (originalRequest._retry || isRefreshRequest(url) || !getRefreshToken()) {
      clearSessionAndRedirect()
      return Promise.reject(err)
    }

    originalRequest._retry = true

    try {
      if (!refreshPromise) {
        refreshPromise = requestTokenRefresh().finally(() => {
          refreshPromise = null
        })
      }

      const accessToken = await refreshPromise
      setAuthorizationHeader(originalRequest, accessToken)
      return client(originalRequest)
    } catch (refreshErr) {
      clearSessionAndRedirect()
      return Promise.reject(refreshErr)
    }
  },
)

export default client
