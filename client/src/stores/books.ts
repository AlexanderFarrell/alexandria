import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as booksApi from '@/api/books'
import type { AuthorSummary, Book, GenreSummary, ReadingProgress } from '@/types'

const PAGE_SIZE = 24

export const useBooksStore = defineStore('books', () => {
  const books = ref<Book[]>([])
  const total = ref(0)
  const page = ref(1)
  const hasMore = ref(false)
  const currentBook = ref<Book | null>(null)
  const currentProgress = ref<ReadingProgress | null>(null)
  const loading = ref(false)
  const loadingMore = ref(false)

  const authors = ref<AuthorSummary[]>([])
  const genres = ref<GenreSummary[]>([])
  const sortBy = ref<booksApi.ListParams['sort_by']>('created_at')
  const sortOrder = ref<booksApi.ListParams['sort_order']>('desc')

  // Track last used filter params (author/genre/search) so setSort/loadMore can re-use them
  const lastFilter = ref<booksApi.ListParams>({})

  async function fetchBooks(params: booksApi.ListParams = {}) {
    lastFilter.value = params
    page.value = 1
    loading.value = true
    try {
      const res = await booksApi.listBooks({
        ...params,
        sort_by: sortBy.value,
        sort_order: sortOrder.value,
        page: 1,
        limit: PAGE_SIZE,
      })
      books.value = res.books ?? []
      total.value = res.total
      hasMore.value = books.value.length < res.total
    } finally {
      loading.value = false
    }
  }

  async function loadMore() {
    if (loadingMore.value || !hasMore.value) return
    loadingMore.value = true
    const nextPage = page.value + 1
    try {
      const res = await booksApi.listBooks({
        ...lastFilter.value,
        sort_by: sortBy.value,
        sort_order: sortOrder.value,
        page: nextPage,
        limit: PAGE_SIZE,
      })
      books.value = [...books.value, ...(res.books ?? [])]
      total.value = res.total
      page.value = nextPage
      hasMore.value = books.value.length < res.total
    } finally {
      loadingMore.value = false
    }
  }

  async function fetchBook(id: string) {
    const res = await booksApi.getBook(id)
    currentBook.value = res.book
    return res.book
  }

  async function uploadBook(file: File, meta?: { title?: string; author?: string }) {
    const fd = new FormData()
    fd.append('file', file)
    if (meta?.title) fd.append('title', meta.title)
    if (meta?.author) fd.append('author', meta.author)
    const res = await booksApi.uploadBook(fd)
    return res.book
  }

  async function updateBook(id: string, updates: booksApi.BookUpdatePayload): Promise<Book> {
    const res = await booksApi.updateBook(id, updates)
    const idx = books.value.findIndex((b) => b.id === id)
    if (idx !== -1) books.value[idx] = res.book
    if (currentBook.value?.id === id) currentBook.value = res.book
    return res.book
  }

  async function deleteBook(id: string) {
    await booksApi.deleteBook(id)
    books.value = books.value.filter((b) => b.id !== id)
    total.value--
  }

  async function fetchProgress(bookId: string) {
    const res = await booksApi.getProgress(bookId)
    currentProgress.value = res.progress
    return res.progress
  }

  async function saveProgress(bookId: string, cfi: string, percentage: number) {
    const res = await booksApi.saveProgress(bookId, cfi, percentage)
    currentProgress.value = res.progress
  }

  async function rateBook(bookId: string, rating: number) {
    // Fetch latest progress first so we don't clobber cfi/percentage
    const existing = await booksApi.getProgress(bookId)
    const p = existing.progress
    const res = await booksApi.saveProgress(
      bookId,
      p?.cfi ?? '',
      p?.percentage ?? 0,
      rating,
    )
    currentProgress.value = res.progress
    return res.progress
  }

  async function refreshBookMetadata(id: string): Promise<Book> {
    const res = await booksApi.refreshMetadata(id)
    const idx = books.value.findIndex((b) => b.id === id)
    if (idx !== -1) books.value[idx] = res.book
    if (currentBook.value?.id === id) currentBook.value = res.book
    return res.book
  }

  async function fetchAuthors() {
    const res = await booksApi.listAuthors()
    authors.value = res.authors ?? []
  }

  async function fetchGenres() {
    const res = await booksApi.listGenres()
    genres.value = res.genres ?? []
  }

  async function setSort(
    by: booksApi.ListParams['sort_by'],
    order: booksApi.ListParams['sort_order'],
  ) {
    sortBy.value = by
    sortOrder.value = order
    await fetchBooks(lastFilter.value)
  }

  return {
    books,
    total,
    page,
    hasMore,
    currentBook,
    currentProgress,
    loading,
    loadingMore,
    authors,
    genres,
    sortBy,
    sortOrder,
    fetchBooks,
    loadMore,
    fetchBook,
    uploadBook,
    updateBook,
    deleteBook,
    fetchProgress,
    saveProgress,
    rateBook,
    refreshBookMetadata,
    fetchAuthors,
    fetchGenres,
    setSort,
  }
})
