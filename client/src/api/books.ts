import client from './client'
import type { Book, BooksListResponse, ReadingProgress } from '@/types'

export interface ListParams {
  search?: string
  author?: string
  genre?: string
  page?: number
  limit?: number
}

export async function listBooks(params: ListParams = {}): Promise<BooksListResponse> {
  const { data } = await client.get('/books', { params })
  return data
}

export async function getBook(id: string): Promise<{ book: Book }> {
  const { data } = await client.get(`/books/${id}`)
  return data
}

export async function uploadBook(formData: FormData): Promise<{ book: Book }> {
  const { data } = await client.post('/books', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data
}

export async function updateBook(
  id: string,
  updates: Partial<Pick<Book, 'title' | 'author' | 'description' | 'zealot_ticket_id'>>,
): Promise<{ book: Book }> {
  const { data } = await client.put(`/books/${id}`, updates)
  return data
}

export async function deleteBook(id: string): Promise<void> {
  await client.delete(`/books/${id}`)
}

export function coverUrl(id: string): string {
  return `/api/v1/books/${id}/cover`
}

export function contentUrl(id: string): string {
  return `/api/v1/books/${id}/content`
}

export async function getProgress(bookId: string): Promise<{ progress: ReadingProgress | null }> {
  const { data } = await client.get(`/books/${bookId}/progress`)
  return data
}

export async function saveProgress(
  bookId: string,
  cfi: string,
  percentage: number,
): Promise<{ progress: ReadingProgress }> {
  const { data } = await client.put(`/books/${bookId}/progress`, { cfi, percentage })
  return data
}
