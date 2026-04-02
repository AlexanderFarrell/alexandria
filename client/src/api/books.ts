import client from './client'
import type {
  AuthorSummary,
  Book,
  BookMetadata,
  BooksListResponse,
  GenreSummary,
  ReaderManifest,
  ReaderSection,
  ReadingProgress,
} from '@/types'

export interface ListParams {
  search?: string
  author?: string
  genre?: string
  sort_by?: 'title' | 'author' | 'created_at' | 'rating' | 'file_size'
  sort_order?: 'asc' | 'desc'
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

export interface BookUpdatePayload {
  title?: string
  author?: string
  description?: string
  zealot_ticket_id?: string
  metadata?: Partial<BookMetadata>
  cover_url?: string
}

export async function updateBook(id: string, updates: BookUpdatePayload): Promise<{ book: Book }> {
  const { data } = await client.put(`/books/${id}`, updates)
  return data
}

export async function deleteBook(id: string): Promise<void> {
  await client.delete(`/books/${id}`)
}

export function coverUrl(id: string): string {
  return `/api/v1/books/${id}/cover`
}

export async function getCoverBlob(id: string): Promise<Blob> {
  const { data } = await client.get(`/books/${id}/cover`, {
    responseType: 'blob',
  })
  return data
}

export function contentUrl(id: string): string {
  return `/api/v1/books/${id}/content`
}

export async function getContentBlob(id: string): Promise<Blob> {
  const { data } = await client.get(`/books/${id}/content`, {
    responseType: 'blob',
  })
  return data
}

export async function refreshMetadata(id: string): Promise<{ book: Book }> {
  const { data } = await client.post(`/books/${id}/refresh`)
  return data
}

export async function getProgress(bookId: string): Promise<{ progress: ReadingProgress | null }> {
  const { data } = await client.get(`/books/${bookId}/progress`)
  return data
}

export interface SaveProgressPayload {
  section_id: string
  section_progress: number
  block_index?: number | null
  percentage: number
  rating?: number
}

export async function saveProgress(bookId: string, payload: SaveProgressPayload): Promise<{ progress: ReadingProgress }> {
  const body: Record<string, unknown> = {
    section_id: payload.section_id,
    section_progress: payload.section_progress,
    percentage: payload.percentage,
  }
  if (payload.block_index !== undefined) body.block_index = payload.block_index
  if (payload.rating !== undefined) body.rating = payload.rating
  const { data } = await client.put(`/books/${bookId}/progress`, body)
  return data
}

export async function getReaderManifest(bookId: string): Promise<{ manifest: ReaderManifest }> {
  const { data } = await client.get(`/books/${bookId}/reader/manifest`)
  return data
}

export async function getReaderSection(
  bookId: string,
  sectionId: string,
): Promise<{ section: ReaderSection }> {
  const { data } = await client.get(`/books/${bookId}/reader/sections/${encodeURIComponent(sectionId)}`)
  return data
}

export async function listAuthors(): Promise<{ authors: AuthorSummary[] }> {
  const { data } = await client.get('/books/authors')
  return data
}

export async function listGenres(): Promise<{ genres: GenreSummary[] }> {
  const { data } = await client.get('/books/genres')
  return data
}
