import client from './client'
import type { BookList, BookListItem } from '@/types'

export async function getLists(): Promise<{ lists: BookList[] }> {
  const { data } = await client.get('/lists')
  return data
}

export async function createList(name: string, description = ''): Promise<{ list: BookList }> {
  const { data } = await client.post('/lists', { name, description })
  return data
}

export async function updateList(
  id: string,
  updates: { name?: string; description?: string },
): Promise<{ list: BookList }> {
  const { data } = await client.put(`/lists/${id}`, updates)
  return data
}

export async function deleteList(id: string): Promise<void> {
  await client.delete(`/lists/${id}`)
}

export async function getListItems(listId: string): Promise<{ items: BookListItem[] }> {
  const { data } = await client.get(`/lists/${listId}/books`)
  return data
}

export async function addBookToList(listId: string, bookId: string): Promise<void> {
  await client.post(`/lists/${listId}/books`, { book_id: bookId })
}

export async function removeBookFromList(listId: string, bookId: string): Promise<void> {
  await client.delete(`/lists/${listId}/books/${bookId}`)
}
