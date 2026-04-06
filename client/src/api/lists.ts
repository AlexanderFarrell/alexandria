import client from './client'
import type { BookList, BookListItem } from '@/types'

export async function getLists(): Promise<{ lists: BookList[] }> {
  return client.get('/lists')
}

export async function createList(name: string, description = ''): Promise<{ list: BookList }> {
  return client.post('/lists', { name, description })
}

export async function updateList(
  id: string,
  updates: { name?: string; description?: string },
): Promise<{ list: BookList }> {
  return client.put(`/lists/${id}`, updates)
}

export async function deleteList(id: string): Promise<void> {
  return client.delete(`/lists/${id}`)
}

export async function getListItems(listId: string): Promise<{ items: BookListItem[] }> {
  return client.get(`/lists/${listId}/books`)
}

export async function addBookToList(listId: string, bookId: string): Promise<void> {
  return client.post(`/lists/${listId}/books`, { book_id: bookId })
}

export async function removeBookFromList(listId: string, bookId: string): Promise<void> {
  return client.delete(`/lists/${listId}/books/${bookId}`)
}
