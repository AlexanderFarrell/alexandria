import client from './client'
import type { AuthStatus, Tokens, User } from '@/types'

export async function getStatus(): Promise<AuthStatus> {
  const { data } = await client.get('/auth/status')
  return data
}

export async function register(username: string, password: string): Promise<{ user: User; tokens: Tokens }> {
  const { data } = await client.post('/auth/register', { username, password })
  return data
}

export async function login(username: string, password: string): Promise<{ user: User; tokens: Tokens }> {
  const { data } = await client.post('/auth/login', { username, password })
  return data
}

export async function refresh(refreshToken: string): Promise<{ tokens: Tokens }> {
  const { data } = await client.post('/auth/refresh', { refresh_token: refreshToken })
  return data
}
