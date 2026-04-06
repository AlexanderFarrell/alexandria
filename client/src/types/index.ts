export interface BookMetadata {
  isbn?: string
  publisher?: string
  published_at?: string
  language?: string
  genres?: string[]
  tags?: string[]
}

export interface BookLink {
  id: string
  book_id: string
  label: string
  url: string
  created_at: string
  updated_at: string
}

export interface Book {
  id: string
  title: string
  author: string
  description: string
  cover_path: string
  file_path: string
  file_type: 'epub' | 'pdf' | 'url'
  file_size: number
  metadata: BookMetadata
  zealot_ticket_id?: string
  links?: BookLink[]
  created_at: string
  updated_at: string
}

export interface ReadingProgress {
  id: string
  user_id: string
  book_id: string
  section_id: string
  section_progress: number
  block_index?: number
  percentage: number
  rating?: number // 1–5; absent means unrated
  zealot_progress_ref?: string
  started_at: string
  last_read_at: string
  finished_at?: string
}

export interface ReaderSectionSummary {
  id: string
  title: string
  index: number
}

export interface ReaderNavItem {
  label: string
  section_id?: string
  fragment?: string
  children?: ReaderNavItem[]
}

export interface ReaderManifest {
  sections: ReaderSectionSummary[]
  nav: ReaderNavItem[]
  first_section_id: string
}

export interface ReaderSection {
  id: string
  title: string
  section_index: number
  prev_section_id?: string
  next_section_id?: string
  html: string
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

export interface AuthStatus {
  registration_mode: 'disable' | 'single' | 'multi'
  can_register: boolean
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

export interface PublisherSummary {
  publisher: string
  count: number
}

export interface YearSummary {
  year: number
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

export interface MetadataSource {
  id: string
  name: string
  link: string
}

export interface MetadataResult {
  title: string
  authors: string[]
  description?: string
  cover_url?: string
  publisher?: string
  published_date?: string
  isbn?: string
  language?: string
  tags?: string[]
  source: MetadataSource
  external_url?: string
}

export interface MetadataSearchResponse {
  results: MetadataResult[]
  errors: Array<{ provider: string; message: string }>
}
