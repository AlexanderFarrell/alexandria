export interface BookMetadata {
  isbn?: string
  publisher?: string
  published_at?: string
  language?: string
  genres?: string[]
  tags?: string[]
}

export interface Book {
  id: string
  title: string
  author: string
  description: string
  cover_path: string
  file_path: string
  file_type: 'epub' | 'pdf' | 'url'
  metadata: BookMetadata
  zealot_ticket_id?: string
  created_at: string
  updated_at: string
}

export interface ReadingProgress {
  id: string
  user_id: string
  book_id: string
  cfi: string
  percentage: number
  rating?: number // 1–5; absent means unrated
  zealot_progress_ref?: string
  started_at: string
  last_read_at: string
  finished_at?: string
}

export interface User {
  id: string
  username: string
  email?: string
  created_at: string
  updated_at: string
}

export interface Tokens {
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface BooksListResponse {
  books: Book[]
  total: number
  page: number
  limit: number
}

export interface AuthorSummary {
  author: string
  count: number
}

export interface GenreSummary {
  genre: string
  count: number
}

export interface BookList {
  id: string
  user_id: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

export interface BookListItem {
  list_id: string
  book_id: string
  added_at: string
}
