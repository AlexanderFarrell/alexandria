import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as listsApi from '@/api/lists'
import type { Book, BookList, BookListItem } from '@/types'
import { getBook } from '@/api/books'

export const useListsStore = defineStore('lists', () => {
  const lists = ref<BookList[]>([])
  const currentList = ref<BookList | null>(null)
  const currentItems = ref<BookListItem[]>([])
  const currentBooks = ref<Book[]>([])
  const loading = ref(false)

  async function fetchLists() {
    loading.value = true
    try {
      const res = await listsApi.getLists()
      lists.value = res.lists ?? []
    } finally {
      loading.value = false
    }
  }

  async function createList(name: string, description = '') {
    const res = await listsApi.createList(name, description)
    lists.value.unshift(res.list)
    return res.list
  }

  async function updateList(id: string, updates: { name?: string; description?: string }) {
    const res = await listsApi.updateList(id, updates)
    const idx = lists.value.findIndex((l) => l.id === id)
    if (idx !== -1) lists.value[idx] = res.list
    if (currentList.value?.id === id) currentList.value = res.list
    return res.list
  }

  async function deleteList(id: string) {
    await listsApi.deleteList(id)
    lists.value = lists.value.filter((l) => l.id !== id)
    if (currentList.value?.id === id) {
      currentList.value = null
      currentItems.value = []
      currentBooks.value = []
    }
  }

  async function selectList(list: BookList) {
    currentList.value = list
    loading.value = true
    try {
      const res = await listsApi.getListItems(list.id)
      currentItems.value = res.items ?? []
      // Fetch the full Book objects for each item
      const bookResults = await Promise.all(currentItems.value.map((item) => getBook(item.book_id)))
      currentBooks.value = bookResults.map((r) => r.book)
    } finally {
      loading.value = false
    }
  }

  async function addBook(listId: string, bookId: string) {
    await listsApi.addBookToList(listId, bookId)
    // If this is the currently selected list, refresh its items
    if (currentList.value?.id === listId) {
      const res = await listsApi.getListItems(listId)
      currentItems.value = res.items ?? []
      const bookResults = await Promise.all(currentItems.value.map((item) => getBook(item.book_id)))
      currentBooks.value = bookResults.map((r) => r.book)
    }
  }

  async function removeBook(listId: string, bookId: string) {
    await listsApi.removeBookFromList(listId, bookId)
    if (currentList.value?.id === listId) {
      currentItems.value = currentItems.value.filter((i) => i.book_id !== bookId)
      currentBooks.value = currentBooks.value.filter((b) => b.id !== bookId)
    }
  }

  return {
    lists,
    currentList,
    currentItems,
    currentBooks,
    loading,
    fetchLists,
    createList,
    updateList,
    deleteList,
    selectList,
    addBook,
    removeBook,
  }
})
