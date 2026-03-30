<template>
  <div class="library-page">
    <NavBar />

    <div class="layout">
      <!-- Sidebar -->
      <aside class="sidebar">
        <section class="sidebar-section">
          <h3 class="sidebar-heading">Authors</h3>
          <ul class="filter-list">
            <li>
              <button
                class="filter-item"
                :class="{ active: !selectedAuthor }"
                @click="setAuthorFilter('')"
              >All</button>
            </li>
            <li v-for="a in books.authors" :key="a.author">
              <button
                class="filter-item"
                :class="{ active: selectedAuthor === a.author }"
                @click="setAuthorFilter(a.author)"
              >
                <span class="filter-name">{{ a.author }}</span>
                <span class="filter-count">{{ a.count }}</span>
              </button>
            </li>
          </ul>
        </section>

        <section class="sidebar-section">
          <h3 class="sidebar-heading">Genres</h3>
          <ul class="filter-list">
            <li>
              <button
                class="filter-item"
                :class="{ active: !selectedGenre }"
                @click="setGenreFilter('')"
              >All</button>
            </li>
            <li v-for="g in books.genres" :key="g.genre">
              <button
                class="filter-item"
                :class="{ active: selectedGenre === g.genre }"
                @click="setGenreFilter(g.genre)"
              >
                <span class="filter-name">{{ g.genre }}</span>
                <span class="filter-count">{{ g.count }}</span>
              </button>
            </li>
          </ul>
        </section>
      </aside>

      <!-- Main content -->
      <main class="content">
        <div class="toolbar">
          <div class="search-wrap">
            <input
              v-model="search"
              type="search"
              placeholder="Search books…"
              @input="onSearch"
            />
          </div>
          <div class="sort-wrap">
            <select v-model="sortByVal" @change="applySort" class="sort-select">
              <option value="created_at">Date added</option>
              <option value="title">Title</option>
              <option value="author">Author</option>
              <option value="rating">My rating</option>
            </select>
            <button class="sort-dir btn-ghost" @click="toggleSortDir" :title="sortOrderVal === 'asc' ? 'Ascending' : 'Descending'">
              {{ sortOrderVal === 'asc' ? '↑' : '↓' }}
            </button>
          </div>
          <button class="btn-primary" @click="openUploadModal">+ Add book</button>
        </div>

        <div v-if="books.loading" class="state-msg">Loading…</div>
        <div v-else-if="books.books.length === 0" class="state-msg empty">
          <p>No books found.</p>
          <button class="btn-primary" @click="openUploadModal">Add your first book</button>
        </div>
        <div v-else class="grid">
          <BookCard
            v-for="book in books.books"
            :key="book.id"
            :book="book"
            :progress="progressMap[book.id]"
            @show-detail="openDetail"
          />
        </div>

        <div v-if="books.total > 0" class="pagination">
          <span class="total">{{ books.total }} book{{ books.total !== 1 ? 's' : '' }}</span>
        </div>
      </main>
    </div>

    <!-- Upload modal -->
    <div v-if="showUpload" class="modal-overlay" @click.self="closeUploadModal">
      <div class="modal card">
        <h2>Add books</h2>
        <form @submit.prevent="uploadBooks">
          <div class="field">
            <label>Files (EPUB or PDF)</label>
            <input
              type="file"
              accept=".epub,.pdf"
              multiple
              :disabled="uploading || uploadFinished"
              @change="onFileChange"
            />
          </div>
          <p class="upload-hint">Books upload one at a time. You can edit metadata after upload.</p>
          <ul v-if="uploadQueue.length > 0" class="upload-queue">
            <li
              v-for="item in uploadQueue"
              :key="item.id"
              class="upload-queue-item"
            >
              <div class="upload-file-meta">
                <span class="upload-file-name">{{ item.file.name }}</span>
                <span class="upload-file-size">{{ formatFileSize(item.file.size) }}</span>
              </div>
              <div class="upload-status-row">
                <span class="upload-status" :class="`status-${item.status}`">
                  {{ uploadStatusLabel(item.status) }}
                </span>
              </div>
              <p v-if="item.error" class="upload-item-error">{{ item.error }}</p>
            </li>
          </ul>
          <p v-if="uploadError" class="error-msg">{{ uploadError }}</p>
          <div class="modal-actions">
            <button
              v-if="!uploading"
              type="button"
              class="btn-ghost"
              @click="closeUploadModal"
            >
              {{ uploadFinished ? 'Close' : 'Cancel' }}
            </button>
            <button
              v-if="!uploadFinished"
              type="submit"
              class="btn-primary"
              :disabled="uploadQueue.length === 0 || uploading"
            >
              {{ uploadButtonLabel }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Book detail modal -->
    <BookDetailModal
      v-if="selectedBook"
      :book="selectedBook"
      :progress="progressMap[selectedBook.id]"
      @close="selectedBook = null"
      @open-reader="openReader"
      @edit="openEdit"
      @deleted="onDeleted"
      @add-to-list="openAddToList"
    />

    <!-- Book edit modal (z-index above detail) -->
    <BookEditModal
      v-if="bookToEdit"
      :book="bookToEdit"
      @close="bookToEdit = null"
      @saved="onSaved"
    />

    <!-- Add to list modal -->
    <div v-if="showAddToList && bookForList" class="modal-overlay" @click.self="showAddToList = false">
      <div class="modal card">
        <h2>Add to list</h2>
        <p class="list-book-title">{{ bookForList.title }}</p>
        <div v-if="listsStore.lists.length === 0" class="state-msg" style="padding: 1rem 0">
          No lists yet. <router-link to="/lists">Create one</router-link>.
        </div>
        <ul v-else class="list-picker">
          <li v-for="list in listsStore.lists" :key="list.id">
            <button class="list-pick-btn btn-ghost" @click="addToList(list.id)">
              {{ list.name }}
            </button>
          </li>
        </ul>
        <div class="modal-actions">
          <button class="btn-ghost" @click="showAddToList = false">Cancel</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import BookCard from '@/components/BookCard.vue'
import BookDetailModal from '@/components/BookDetailModal.vue'
import BookEditModal from '@/components/BookEditModal.vue'
import { useBooksStore } from '@/stores/books'
import { useListsStore } from '@/stores/lists'
import type { Book, ReadingProgress } from '@/types'

const books = useBooksStore()
const listsStore = useListsStore()
const router = useRouter()

const search = ref('')
const selectedAuthor = ref('')
const selectedGenre = ref('')
const sortByVal = ref<'created_at' | 'title' | 'author' | 'rating'>('created_at')
const sortOrderVal = ref<'asc' | 'desc'>('desc')

const selectedBook = ref<Book | null>(null)
const bookToEdit = ref<Book | null>(null)
const showAddToList = ref(false)
const bookForList = ref<Book | null>(null)

const showUpload = ref(false)
type UploadQueueStatus = 'pending' | 'uploading' | 'success' | 'error'

interface UploadQueueItem {
  id: string
  file: File
  status: UploadQueueStatus
  error: string
}

const uploadQueue = ref<UploadQueueItem[]>([])
const uploadError = ref('')
const uploading = ref(false)
const uploadFinished = ref(false)

const uploadButtonLabel = computed(() => {
  if (uploading.value) return 'Uploading…'
  if (uploadQueue.value.length === 0) return 'Upload'
  return uploadQueue.value.length === 1
    ? 'Upload 1 book'
    : `Upload ${uploadQueue.value.length} books`
})

// Progress map keyed by book id — populated lazily when a book is opened
const progressMap = reactive<Record<string, ReadingProgress | null>>({})

let searchTimer: ReturnType<typeof setTimeout>

onMounted(async () => {
  await Promise.all([
    books.fetchBooks(),
    books.fetchAuthors(),
    books.fetchGenres(),
    listsStore.fetchLists(),
  ])
})

function currentFilter() {
  return {
    search: search.value || undefined,
    author: selectedAuthor.value || undefined,
    genre: selectedGenre.value || undefined,
  }
}

function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => books.fetchBooks(currentFilter()), 300)
}

function setAuthorFilter(author: string) {
  selectedAuthor.value = author
  selectedGenre.value = ''
  books.fetchBooks(currentFilter())
}

function setGenreFilter(genre: string) {
  selectedGenre.value = genre
  selectedAuthor.value = ''
  books.fetchBooks(currentFilter())
}

async function applySort() {
  await books.setSort(sortByVal.value, sortOrderVal.value)
}

async function toggleSortDir() {
  sortOrderVal.value = sortOrderVal.value === 'asc' ? 'desc' : 'asc'
  await applySort()
}

async function openDetail(book: Book) {
  selectedBook.value = book
  if (!(book.id in progressMap)) {
    const p = await books.fetchProgress(book.id)
    progressMap[book.id] = p
  }
}

function openReader(book: Book) {
  selectedBook.value = null
  router.push(`/books/${book.id}`)
}

function openEdit(book: Book) {
  bookToEdit.value = book
}

function onSaved(updatedBook: Book) {
  bookToEdit.value = null
  if (selectedBook.value?.id === updatedBook.id) {
    selectedBook.value = updatedBook
  }
}

function onDeleted() {
  selectedBook.value = null
  books.fetchAuthors()
  books.fetchGenres()
}

function openUploadModal() {
  resetUploadModal()
  showUpload.value = true
}

function closeUploadModal() {
  if (uploading.value) return
  resetUploadModal()
  showUpload.value = false
}

function resetUploadModal() {
  uploadQueue.value = []
  uploadError.value = ''
  uploadFinished.value = false
}

function openAddToList(book: Book) {
  bookForList.value = book
  showAddToList.value = true
}

async function addToList(listId: string) {
  if (!bookForList.value) return
  await listsStore.addBook(listId, bookForList.value.id)
  showAddToList.value = false
  bookForList.value = null
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  uploadQueue.value = Array.from(input.files ?? []).map((file, index) => ({
    id: `${file.name}-${file.size}-${file.lastModified}-${index}`,
    file,
    status: 'pending',
    error: '',
  }))
  uploadError.value = ''
  uploadFinished.value = false
}

function uploadStatusLabel(status: UploadQueueStatus) {
  switch (status) {
    case 'uploading':
      return 'Uploading'
    case 'success':
      return 'Uploaded'
    case 'error':
      return 'Failed'
    default:
      return 'Pending'
  }
}

function formatFileSize(size: number) {
  const units = ['B', 'KB', 'MB', 'GB']
  let value = size
  let unitIndex = 0

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex++
  }

  const digits = unitIndex === 0 ? 0 : 1
  return `${value.toFixed(digits)} ${units[unitIndex]}`
}

function getUploadErrorMessage(error: unknown) {
  const msg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
  return msg ?? 'Upload failed. Try again.'
}

async function refreshLibraryData() {
  await Promise.all([
    books.fetchBooks(currentFilter()),
    books.fetchAuthors(),
    books.fetchGenres(),
  ])
}

async function uploadBooks() {
  if (uploadQueue.value.length === 0 || uploading.value || uploadFinished.value) return

  uploading.value = true
  uploadError.value = ''
  uploadFinished.value = false
  let successfulUploads = 0

  try {
    for (const item of uploadQueue.value) {
      item.status = 'uploading'
      item.error = ''

      try {
        await books.uploadBook(item.file)
        item.status = 'success'
        successfulUploads++
      } catch (error: unknown) {
        item.status = 'error'
        item.error = getUploadErrorMessage(error)
      }
    }

    if (successfulUploads > 0) {
      try {
        await refreshLibraryData()
      } catch {
        uploadError.value = 'Books uploaded, but the library could not be refreshed.'
      }
    }
  } finally {
    uploading.value = false
    uploadFinished.value = true
  }

  if (successfulUploads === uploadQueue.value.length && uploadError.value === '') {
    closeUploadModal()
  }
}
</script>

<style scoped>
.library-page {
  min-height: 100vh;
  background: var(--bg);
}

.layout {
  display: flex;
  max-width: 1400px;
  margin: 0 auto;
  padding: 1.5rem;
  gap: 1.5rem;
  align-items: flex-start;
}

/* Sidebar */
.sidebar {
  width: 200px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  position: sticky;
  top: 72px; /* below navbar */
  max-height: calc(100vh - 90px);
  overflow-y: auto;
}
.sidebar-heading {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  margin-bottom: 0.5rem;
}
.filter-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
}
.filter-item {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.35rem 0.6rem;
  border-radius: 6px;
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 0.85rem;
  text-align: left;
  cursor: pointer;
  transition: background 0.1s, color 0.1s;
}
.filter-item:hover { background: var(--surface-hover); color: var(--text); }
.filter-item.active { background: var(--surface-hover); color: var(--accent); }
.filter-name {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  flex: 1;
  min-width: 0;
}
.filter-count {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-left: 0.4rem;
  flex-shrink: 0;
}

/* Main content */
.content {
  flex: 1;
  min-width: 0;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}
.search-wrap { flex: 1; min-width: 160px; max-width: 380px; }
.sort-wrap {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
.sort-select {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--text);
  font-family: inherit;
  font-size: 0.88rem;
  padding: 0.5rem 0.75rem;
  cursor: pointer;
  outline: none;
  width: auto;
}
.sort-select:focus { border-color: var(--accent); }
.sort-dir {
  padding: 0.5rem 0.65rem;
  font-size: 1rem;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 1.25rem;
}

.state-msg {
  text-align: center;
  color: var(--text-muted);
  padding: 4rem 0;
}
.state-msg.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.pagination {
  margin-top: 2rem;
  text-align: center;
  color: var(--text-muted);
  font-size: 0.85rem;
}

/* Upload / add-to-list modal (shared styles) */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  padding: 1rem;
}
.modal {
  width: 100%;
  max-width: 460px;
  padding: 1.5rem;
}
.modal h2 {
  margin-bottom: 1.25rem;
  font-size: 1.1rem;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 0.9rem;
}
.field label { font-size: 0.85rem; color: var(--text-muted); }
.upload-hint {
  margin: 0 0 0.9rem;
  font-size: 0.85rem;
  color: var(--text-muted);
}
.upload-queue {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin: 0;
  padding: 0;
}
.upload-queue-item {
  padding: 0.85rem 0.95rem;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}
.upload-file-meta {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  align-items: baseline;
}
.upload-file-name {
  font-size: 0.92rem;
  font-weight: 600;
  color: var(--text);
  word-break: break-word;
}
.upload-file-size {
  flex-shrink: 0;
  font-size: 0.8rem;
  color: var(--text-muted);
}
.upload-status-row {
  margin-top: 0.5rem;
}
.upload-status {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 0.2rem 0.55rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.upload-status.status-pending {
  background: rgba(148, 163, 184, 0.14);
  color: var(--text-muted);
}
.upload-status.status-uploading {
  background: rgba(59, 130, 246, 0.14);
  color: #60a5fa;
}
.upload-status.status-success {
  background: rgba(34, 197, 94, 0.14);
  color: #4ade80;
}
.upload-status.status-error {
  background: rgba(248, 113, 113, 0.14);
  color: #f87171;
}
.upload-item-error,
.error-msg {
  margin-top: 0.5rem;
  font-size: 0.82rem;
  color: #f87171;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.75rem;
}

/* Add-to-list picker */
.list-book-title {
  font-size: 0.9rem;
  color: var(--text-muted);
  margin-bottom: 0.75rem;
}
.list-picker {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  margin-bottom: 0.75rem;
}
.list-pick-btn {
  width: 100%;
  text-align: left;
}
</style>
