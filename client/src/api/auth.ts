import client from './client'
import type { AuthStatus, Tokens, User } from '@/types'

export async function getStatus(): Promise<AuthStatus> {
  return client.get('/auth/status')
}

export async function register(username: string, password: string): Promise<{ user: User; tokens: Tokens }> {
  return client.post('/auth/register', { username, password })
}

export async function login(username: string, password: string): Promise<{ user: User; tokens: Tokens }> {
  return client.post('/auth/login', { username, password })
}

export async function refresh(refreshToken: string): Promise<{ tokens: Tokens }> {
  return client.post('/auth/refresh', { refresh_token: refreshToken })
}
