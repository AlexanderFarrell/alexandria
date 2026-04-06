import client from './client'
import type {
  AuthorSummary,
  Book,
  BookLink,
  BookMetadata,
  BooksListResponse,
  GenreSummary,
  PublisherSummary,
  YearSummary,
  ReaderManifest,
  ReaderSection,
  ReadingProgress,
} from '@/types'
import { useServerConfigStore } from '@/stores/serverConfig'

export interface ListParams {
  search?: string
  author?: string
  genre?: string
  publisher?: string
  year?: number
  sort_by?: 'title' | 'author' | 'created_at' | 'rating' | 'file_size' | 'published_at'
  sort_order?: 'asc' | 'desc'
  page?: number
  limit?: number
}

export async function listBooks(params: ListParams = {}): Promise<BooksListResponse> {
  return client.get('/books', { params: params as Record<string, string | number | boolean | null | undefined> })
}

export async function getBook(id: string): Promise<{ book: Book }> {
  return client.get(`/books/${id}`)
}

export async function uploadBook(formData: FormData): Promise<{ book: Book }> {
  return client.post('/books', formData)
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
  return client.put(`/books/${id}`, updates)
}

export async function deleteBook(id: string): Promise<void> {
  return client.delete(`/books/${id}`)
}

function getServerBase(): string {
  try {
    const cfg = useServerConfigStore()
    if (cfg.serverUrl) return cfg.serverUrl
  } catch {
    // Outside Pinia context
  }
  return ''
}

export function coverUrl(id: string): string {
  return `${getServerBase()}/api/v1/books/${id}/cover`
}

export async function getCoverBlob(id: string): Promise<Blob> {
  return client.get(`/books/${id}/cover`, { blob: true })
}

export function contentUrl(id: string): string {
  return `${getServerBase()}/api/v1/books/${id}/content`
}

export async function getContentBlob(id: string): Promise<Blob> {
  return client.get(`/books/${id}/content`, { blob: true })
}

export async function refreshMetadata(id: string): Promise<{ book: Book }> {
  return client.post(`/books/${id}/refresh`)
}

export async function getProgress(bookId: string): Promise<{ progress: ReadingProgress | null }> {
  return client.get(`/books/${bookId}/progress`)
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
  return client.put(`/books/${bookId}/progress`, body)
}

export async function getReaderManifest(bookId: string): Promise<{ manifest: ReaderManifest }> {
  return client.get(`/books/${bookId}/reader/manifest`)
}

export async function getReaderSection(
  bookId: string,
  sectionId: string,
): Promise<{ section: ReaderSection }> {
  return await client.get(`/books/${bookId}/reader/sections/${encodeURIComponent(sectionId)}`)
}

export async function listAuthors(): Promise<{ authors: AuthorSummary[] }> {
  return client.get('/books/authors')
}

export async function listGenres(): Promise<{ genres: GenreSummary[] }> {
  return client.get('/books/genres')
}

export async function listPublishers(): Promise<{ publishers: PublisherSummary[] }> {
  return client.get('/books/publishers')
}

export async function listYears(): Promise<{ years: YearSummary[] }> {
  return client.get('/books/years')
}

export interface LinkPayload {
  label: string
  url: string
}

export async function addBookLink(bookId: string, payload: LinkPayload): Promise<{ link: BookLink }> {
  return client.post(`/books/${bookId}/links`, payload)
}

export async function updateBookLink(
  bookId: string,
  linkId: string,
  payload: LinkPayload,
): Promise<{ link: BookLink }> {
  return client.put(`/books/${bookId}/links/${linkId}`, payload)
}

export async function deleteBookLink(bookId: string, linkId: string): Promise<void> {
  return client.delete(`/books/${bookId}/links/${linkId}`)
}
