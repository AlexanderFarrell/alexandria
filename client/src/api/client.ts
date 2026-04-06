import { getAccessToken, getRefreshToken, storeTokens, clearStoredSession } from '@/utils/auth'
import { useServerConfigStore } from '@/stores/serverConfig'
import type { Tokens } from '@/types'

let _onSessionExpired = () => {
  if (window.location.pathname !== '/login') {
    window.location.href = '/login'
  }
}

export function setSessionExpiredHandler(fn: () => void) {
  _onSessionExpired = fn
}

function getBaseURL(): string {
  try {
    const cfg = useServerConfigStore()
    if (cfg.serverUrl) return cfg.serverUrl + '/api/v1'
  } catch {
    // Pinia not yet initialized
  }
  return '/api/v1'
}

function buildURL(path: string, params?: Record<string, string | number | boolean | null | undefined>): string {
  let url = getBaseURL() + path
  if (params) {
    const qs = new URLSearchParams(
      Object.entries(params)
        .filter(([, v]) => v != null && v !== '')
        .map(([k, v]) => [k, String(v)]),
    ).toString()
    if (qs) url += '?' + qs
  }
  return url
}

function isPublicPath(path: string): boolean {
  return path === '/auth/login' || path === '/auth/register' || path === '/auth/status'
}

// Deduplicates concurrent refresh attempts
let refreshPromise: Promise<void> | null = null

async function ensureTokenRefresh(): Promise<void> {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      const refreshToken = getRefreshToken()
      if (!refreshToken) throw new Error('no refresh token')
      const res = await fetch(getBaseURL() + '/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      })
      if (!res.ok) throw new Error('refresh failed')
      const { tokens } = (await res.json()) as { tokens: Tokens }
      storeTokens(tokens)
    })().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

function clearSessionAndRedirect() {
  clearStoredSession()
  _onSessionExpired()
}

interface RequestOptions {
  params?: Record<string, string | number | boolean | null | undefined>
  body?: unknown
  headers?: Record<string, string>
  blob?: boolean
  arrayBuffer?: boolean
}

// Throws an axios-shaped error so existing view error handlers keep working:
// (e as { response?: { data?: { error?: string } } })?.response?.data?.error
class ApiError extends Error {
  response: { status: number; data: unknown }
  constructor(status: number, data: unknown) {
    super(`API error ${status}`)
    this.response = { status, data }
  }
}

async function rawRequest(
  method: string,
  path: string,
  options: RequestOptions,
): Promise<Response> {
  const headers: Record<string, string> = { ...options.headers }

  const token = getAccessToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const isFormData = options.body instanceof FormData
  if (!isFormData && options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  return fetch(buildURL(path, options.params), {
    method,
    headers,
    body: isFormData
      ? (options.body as FormData)
      : options.body !== undefined
        ? JSON.stringify(options.body)
        : undefined,
  })
}

async function request<T>(
  method: string,
  path: string,
  options: RequestOptions = {},
  isRetry = false,
): Promise<T> {
  const res = await rawRequest(method, path, options)

  // 401: attempt token refresh then retry once
  if (res.status === 401 && !isRetry && !isPublicPath(path)) {
    if (!getRefreshToken()) {
      clearSessionAndRedirect()
      throw new ApiError(401, { error: 'Session expired' })
    }
    try {
      await ensureTokenRefresh()
      return request<T>(method, path, options, true)
    } catch {
      clearSessionAndRedirect()
      throw new ApiError(401, { error: 'Session expired' })
    }
  }

  if (!res.ok) {
    const data = await res.json().catch(() => ({ error: `HTTP ${res.status}` }))
    throw new ApiError(res.status, data)
  }

  if (options.blob) return res.blob() as T
  if (options.arrayBuffer) return res.arrayBuffer() as T
  if (res.status === 204) return undefined as T
  return res.json() as T
}

const client = {
  get<T>(path: string, options: Omit<RequestOptions, 'body'> = {}): Promise<T> {
    return request<T>('GET', path, options)
  },
  post<T>(path: string, body?: unknown, options: Omit<RequestOptions, 'body'> = {}): Promise<T> {
    return request<T>('POST', path, { ...options, body })
  },
  put<T>(path: string, body?: unknown, options: Omit<RequestOptions, 'body'> = {}): Promise<T> {
    return request<T>('PUT', path, { ...options, body })
  },
  delete<T = void>(path: string): Promise<T> {
    return request<T>('DELETE', path)
  },
}

export default client
