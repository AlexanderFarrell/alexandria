import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as booksApi from '@/api/books'
import type { Book, ReadingProgress } from '@/types'

export const useBooksStore = defineStore('books', () => {
  const books = ref<Book[]>([])
  const total = ref(0)
  const currentBook = ref<Book | null>(null)
  const currentProgress = ref<ReadingProgress | null>(null)
  const loading = ref(false)

  async function fetchBooks(params: booksApi.ListParams = {}) {
    loading.value = true
    try {
      const res = await booksApi.listBooks(params)
      books.value = res.books ?? []
      total.value = res.total
    } finally {
      loading.value = false
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
    books.value.unshift(res.book)
    total.value++
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

  return {
    books,
    total,
    currentBook,
    currentProgress,
    loading,
    fetchBooks,
    fetchBook,
    uploadBook,
    deleteBook,
    fetchProgress,
    saveProgress,
  }
})
