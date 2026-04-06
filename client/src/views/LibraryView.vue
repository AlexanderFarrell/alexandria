<template>
  <div class="library-page">
    <NavBar />

    <div class="layout">
      <!-- Sidebar -->
      <aside class="sidebar" :class="{ visible: showFilters }">
        <section class="sidebar-section">
          <h3 class="sidebar-heading">Authors</h3>
          <ul class="filter-list">
            <li>
              <button class="filter-item" :class="{ active: !selectedAuthor }" @click="setAuthorFilter('')">All</button>
            </li>
            <li v-for="a in (expandedSidebar.authors ? books.authors : books.authors.slice(0, SIDEBAR_LIMIT))" :key="a.author">
              <button class="filter-item" :class="{ active: selectedAuthor === a.author }" @click="setAuthorFilter(a.author)">
                <span class="filter-name">{{ a.author }}</span>
                <span class="filter-count">{{ a.count }}</span>
              </button>
            </li>
            <li v-if="books.authors.length > SIDEBAR_LIMIT">
              <button class="filter-item show-more" @click="expandedSidebar.authors = !expandedSidebar.authors">
                {{ expandedSidebar.authors ? 'Show less' : `+${books.authors.length - SIDEBAR_LIMIT} more` }}
              </button>
            </li>
          </ul>
        </section>

        <section class="sidebar-section">
          <h3 class="sidebar-heading">Genres</h3>
          <ul class="filter-list">
            <li>
              <button class="filter-item" :class="{ active: !selectedGenre }" @click="setGenreFilter('')">All</button>
            </li>
            <li v-for="g in (expandedSidebar.genres ? books.genres : books.genres.slice(0, SIDEBAR_LIMIT))" :key="g.genre">
              <button class="filter-item" :class="{ active: selectedGenre === g.genre }" @click="setGenreFilter(g.genre)">
                <span class="filter-name">{{ g.genre }}</span>
                <span class="filter-count">{{ g.count }}</span>
              </button>
            </li>
            <li v-if="books.genres.length > SIDEBAR_LIMIT">
              <button class="filter-item show-more" @click="expandedSidebar.genres = !expandedSidebar.genres">
                {{ expandedSidebar.genres ? 'Show less' : `+${books.genres.length - SIDEBAR_LIMIT} more` }}
              </button>
            </li>
          </ul>
        </section>

        <section v-if="books.publishers.length > 0" class="sidebar-section">
          <h3 class="sidebar-heading">Publishers</h3>
          <ul class="filter-list">
            <li>
              <button class="filter-item" :class="{ active: !selectedPublisher }" @click="setPublisherFilter('')">All</button>
            </li>
            <li v-for="p in (expandedSidebar.publishers ? books.publishers : books.publishers.slice(0, SIDEBAR_LIMIT))" :key="p.publisher">
              <button class="filter-item" :class="{ active: selectedPublisher === p.publisher }" @click="setPublisherFilter(p.publisher)">
                <span class="filter-name">{{ p.publisher }}</span>
                <span class="filter-count">{{ p.count }}</span>
              </button>
            </li>
            <li v-if="books.publishers.length > SIDEBAR_LIMIT">
              <button class="filter-item show-more" @click="expandedSidebar.publishers = !expandedSidebar.publishers">
                {{ expandedSidebar.publishers ? 'Show less' : `+${books.publishers.length - SIDEBAR_LIMIT} more` }}
              </button>
            </li>
          </ul>
        </section>

        <section v-if="books.years.length > 0" class="sidebar-section">
          <h3 class="sidebar-heading">Year</h3>
          <ul class="filter-list">
            <li>
              <button class="filter-item" :class="{ active: !selectedYear }" @click="setYearFilter(0)">All</button>
            </li>
            <li v-for="y in (expandedSidebar.years ? books.years : books.years.slice(0, SIDEBAR_LIMIT))" :key="y.year">
              <button class="filter-item" :class="{ active: selectedYear === y.year }" @click="setYearFilter(y.year)">
                <span class="filter-name">{{ y.year }}</span>
                <span class="filter-count">{{ y.count }}</span>
              </button>
            </li>
            <li v-if="books.years.length > SIDEBAR_LIMIT">
              <button class="filter-item show-more" @click="expandedSidebar.years = !expandedSidebar.years">
                {{ expandedSidebar.years ? 'Show less' : `+${books.years.length - SIDEBAR_LIMIT} more` }}
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
              ref="searchInputRef"
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
              <option value="file_size">File size</option>
              <option value="published_at">Year published</option>
            </select>
            <button class="sort-dir btn-ghost" @click="toggleSortDir" :title="sortOrderVal === 'asc' ? 'Ascending' : 'Descending'">
              {{ sortOrderVal === 'asc' ? '↑' : '↓' }}
            </button>
          </div>
          <button
            class="btn-ghost view-toggle"
            :title="viewMode === 'grid' ? 'Switch to table view' : 'Switch to grid view'"
            @click="viewMode = viewMode === 'grid' ? 'table' : 'grid'"
          >{{ viewMode === 'grid' ? '☰' : '⊞' }}</button>
          <button class="btn-ghost filter-toggle" @click="showFilters = !showFilters">Filters</button>
          <button class="btn-primary" @click="openUploadModal">+ Add book</button>
        </div>

        <div v-if="books.loading" class="state-msg">Loading…</div>
        <div v-else-if="books.books.length === 0" class="state-msg empty">
          <p>No books found.</p>
          <button class="btn-primary" @click="openUploadModal">Add your first book</button>
        </div>
        <div v-else-if="viewMode === 'grid'" class="grid">
          <BookCard
            v-for="book in books.books"
            :key="book.id"
            :book="book"
            :progress="progressMap[book.id]"
            :selectable="true"
            :selected="selection.isSelected(book.id)"
            @show-detail="openDetail"
            @toggle-select="selection.toggle"
          />
        </div>
        <div v-else class="table-wrap">
          <table class="book-table">
            <thead>
              <tr>
                <th class="col-check">
                  <input
                    type="checkbox"
                    :checked="selection.allSelected.value"
                    @change="selection.allSelected.value ? selection.clearSelection() : selection.selectAll()"
                  />
                </th>
                <th>Cover</th>
                <th class="sortable" @click="setTableSort('title')">
                  Title <span class="sort-indicator">{{ sortByVal === 'title' ? (sortOrderVal === 'asc' ? '↑' : '↓') : '' }}</span>
                </th>
                <th>Genres</th>
                <th class="sortable" @click="setTableSort('published_at')">
                  Year <span class="sort-indicator">{{ sortByVal === 'published_at' ? (sortOrderVal === 'asc' ? '↑' : '↓') : '' }}</span>
                </th>
                <th>Publisher</th>
                <th>Type</th>
                <th class="sortable" @click="setTableSort('file_size')">
                  Size <span class="sort-indicator">{{ sortByVal === 'file_size' ? (sortOrderVal === 'asc' ? '↑' : '↓') : '' }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <BookTableRow
                v-for="book in books.books"
                :key="book.id"
                :book="book"
                :selected="selection.isSelected(book.id)"
                @toggle-select="selection.toggle"
                @show-detail="openDetail"
              />
            </tbody>
          </table>
        </div>

        <BatchActionBar
          v-if="selection.hasSelection.value"
          :count="selection.selectedCount.value"
          :lists="listsStore.lists"
          @clear="selection.clearSelection"
          @delete="onBatchDelete"
          @add-to-list="onBatchAddToList"
        />

        <div v-if="books.total > 0" class="pagination">
          <span class="total">{{ books.books.length }} of {{ books.total }} book{{ books.total !== 1 ? 's' : '' }}</span>
        </div>

        <div ref="sentinel" class="sentinel" aria-hidden="true" />
        <div v-if="books.loadingMore" class="state-msg load-more-msg">Loading more…</div>
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
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import BookCard from '@/components/BookCard.vue'
import BookDetailModal from '@/components/BookDetailModal.vue'
import BookEditModal from '@/components/BookEditModal.vue'
import BookTableRow from '@/components/BookTableRow.vue'
import BatchActionBar from '@/components/BatchActionBar.vue'
import { useBooksStore } from '@/stores/books'
import { useListsStore } from '@/stores/lists'
import { useSelection } from '@/composables/useSelection'
import { useHotkeys } from '@/composables/useHotkeys'
import type { Book, ReadingProgress } from '@/types'

const books = useBooksStore()
const listsStore = useListsStore()
const router = useRouter()

const sentinel = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null

const search = ref('')
const selectedAuthor = ref('')
const selectedGenre = ref('')
const selectedPublisher = ref('')
const selectedYear = ref(0)
const sortByVal = ref<'created_at' | 'title' | 'author' | 'rating' | 'file_size' | 'published_at'>('created_at')
const sortOrderVal = ref<'asc' | 'desc'>('desc')
const viewMode = ref<'grid' | 'table'>('grid')
const searchInputRef = ref<HTMLInputElement | null>(null)
const selection = useSelection(() => books.books)

const SIDEBAR_LIMIT = 8
const expandedSidebar = reactive<Record<string, boolean>>({
  authors: false,
  genres: false,
  publishers: false,
  years: false,
})

const showFilters = ref(false)
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
    books.fetchPublishers(),
    books.fetchYears(),
    listsStore.fetchLists(),
  ])

  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting && books.hasMore && !books.loadingMore) {
        books.loadMore()
      }
    },
    { rootMargin: '200px' },
  )
  if (sentinel.value) observer.observe(sentinel.value)
})

onUnmounted(() => {
  observer?.disconnect()
})

watch(sentinel, (el) => {
  if (el && observer) observer.observe(el)
})

function currentFilter() {
  return {
    search: search.value || undefined,
    author: selectedAuthor.value || undefined,
    genre: selectedGenre.value || undefined,
    publisher: selectedPublisher.value || undefined,
    year: selectedYear.value || undefined,
  }
}

function onSearch() {
  selection.clearSelection()
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => books.fetchBooks(currentFilter()), 300)
}

function setAuthorFilter(author: string) {
  selectedAuthor.value = author
  selectedGenre.value = ''
  selection.clearSelection()
  books.fetchBooks(currentFilter())
}

function setGenreFilter(genre: string) {
  selectedGenre.value = genre
  selectedAuthor.value = ''
  selection.clearSelection()
  books.fetchBooks(currentFilter())
}

function setPublisherFilter(publisher: string) {
  selectedPublisher.value = publisher
  selection.clearSelection()
  books.fetchBooks(currentFilter())
}

function setYearFilter(year: number) {
  selectedYear.value = year
  selection.clearSelection()
  books.fetchBooks(currentFilter())
}

async function applySort() {
  await books.setSort(sortByVal.value, sortOrderVal.value)
}

function setTableSort(col: typeof sortByVal.value) {
  if (sortByVal.value === col) {
    sortOrderVal.value = sortOrderVal.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortByVal.value = col
    sortOrderVal.value = 'asc'
  }
  applySort()
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
  books.fetchPublishers()
  books.fetchYears()
}

function onDeleted() {
  selectedBook.value = null
  books.fetchAuthors()
  books.fetchGenres()
  books.fetchPublishers()
  books.fetchYears()
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
    books.fetchPublishers(),
    books.fetchYears(),
  ])
}

async function onBatchAddToList(listId: string) {
  const ids = [...selection.selectedIds.value]
  await Promise.all(ids.map((id) => listsStore.addBook(listId, id)))
  selection.clearSelection()
}

async function onBatchDelete() {
  const ids = [...selection.selectedIds.value]
  await Promise.all(ids.map((id) => books.deleteBook(id)))
  selection.clearSelection()
  books.fetchAuthors()
  books.fetchGenres()
  books.fetchPublishers()
  books.fetchYears()
}

useHotkeys([
  {
    key: '/',
    handler: () => searchInputRef.value?.focus(),
  },
  {
    key: 'k',
    ctrl: true,
    allowInInput: true,
    handler: () => { searchInputRef.value?.focus(); searchInputRef.value?.select() },
  },
  {
    key: 'u',
    handler: () => openUploadModal(),
  },
  {
    key: 't',
    handler: () => { viewMode.value = viewMode.value === 'grid' ? 'table' : 'grid' },
  },
  {
    key: 'a',
    handler: () => {
      if (selection.allSelected.value) {
        selection.clearSelection()
      } else {
        selection.selectAll()
      }
    },
  },
  {
    key: 'Escape',
    allowInInput: true,
    handler: () => {
      if (selection.hasSelection.value) {
        selection.clearSelection()
      } else if (showUpload.value) {
        closeUploadModal()
      } else if (showAddToList.value) {
        showAddToList.value = false
      } else if (bookToEdit.value) {
        bookToEdit.value = null
      } else if (selectedBook.value) {
        selectedBook.value = null
      }
    },
  },
])

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
.filter-item.show-more { color: var(--accent); font-size: 0.78rem; justify-content: center; }
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
  flex-wrap: wrap;
  position: sticky;
  top: 56px;
  z-index: 90;
  background: var(--bg);
  padding: 0.75rem 0;
  margin-bottom: 0.75rem;
  border-bottom: 1px solid var(--border);
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
  margin-top: 1.5rem;
  text-align: center;
  color: var(--text-muted);
  font-size: 0.85rem;
}

.sentinel {
  height: 1px;
}

.load-more-msg {
  padding: 1.5rem 0;
  font-size: 0.9rem;
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
  max-height: 280px;
  overflow-y: auto;
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

/* Table view */
.table-wrap {
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  margin-top: 0.75rem;
}
.book-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}
.book-table thead tr {
  background: var(--surface);
  border-bottom: 2px solid var(--border);
}
.book-table th {
  padding: 0.65rem 0.85rem;
  text-align: left;
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  white-space: nowrap;
}
.book-table th.sortable {
  cursor: pointer;
  user-select: none;
}
.book-table th.sortable:hover { color: var(--text); }
.col-check { width: 2.5rem; text-align: center; }
.col-check input[type='checkbox'] {
  cursor: pointer;
  width: 1rem;
  height: 1rem;
  accent-color: var(--accent);
}
.sort-indicator { color: var(--accent); margin-left: 0.2rem; }
.view-toggle { font-size: 1.1rem; padding: 0.4rem 0.6rem; }

/* Mobile responsiveness */
.filter-toggle { display: none; }

@media (max-width: 768px) {
  .layout { flex-direction: column; padding: 1rem; gap: 1rem; }
  .sidebar { display: none; width: 100%; position: static; max-height: 300px; overflow-y: auto; }
  .sidebar.visible { display: flex; }
  .filter-toggle { display: inline-flex; }
  .grid { grid-template-columns: repeat(auto-fill, minmax(130px, 1fr)); gap: 1rem; }
  .search-wrap { min-width: 0; max-width: none; }
}

@media (max-width: 400px) {
  .grid { grid-template-columns: repeat(2, 1fr); }
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
