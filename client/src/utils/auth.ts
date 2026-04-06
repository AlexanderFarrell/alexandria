import type { Tokens } from '@/types'
import { storageAdapter } from './storage'

const ACCESS_TOKEN_KEY = 'access_token'
const REFRESH_TOKEN_KEY = 'refresh_token'

let _accessToken: string | null = null
let _refreshToken: string | null = null

export async function initAuthStorage(): Promise<void> {
  _accessToken = await storageAdapter.get(ACCESS_TOKEN_KEY)
  _refreshToken = await storageAdapter.get(REFRESH_TOKEN_KEY)
}

export function getAccessToken(): string | null {
  return _accessToken
}

export function getRefreshToken(): string | null {
  return _refreshToken
}

export function hasStoredSession(): boolean {
  return !!_accessToken || !!_refreshToken
}

export function storeTokens(tokens: Tokens): void {
  _accessToken = tokens.access_token
  _refreshToken = tokens.refresh_token
  storageAdapter.set(ACCESS_TOKEN_KEY, tokens.access_token)
  storageAdapter.set(REFRESH_TOKEN_KEY, tokens.refresh_token)
}

export function clearStoredSession(): void {
  _accessToken = null
  _refreshToken = null
  storageAdapter.remove(ACCESS_TOKEN_KEY)
  storageAdapter.remove(REFRESH_TOKEN_KEY)
}
