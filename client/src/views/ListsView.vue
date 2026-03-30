<template>
  <div class="lists-page">
    <NavBar />

    <div class="layout">
      <!-- Left panel: list of lists -->
      <aside class="lists-panel card">
        <div class="panel-header">
          <h2>My Lists</h2>
          <button class="btn-primary new-btn" @click="showCreate = true">+ New</button>
        </div>

        <div v-if="listsStore.loading && listsStore.lists.length === 0" class="state-msg">Loading…</div>
        <div v-else-if="listsStore.lists.length === 0" class="state-msg">No lists yet.</div>
        <ul v-else class="list-index">
          <li
            v-for="list in listsStore.lists"
            :key="list.id"
            class="list-entry"
            :class="{ active: listsStore.currentList?.id === list.id }"
            @click="listsStore.selectList(list)"
          >
            <span class="list-name">{{ list.name }}</span>
            <button
              class="list-delete btn-ghost"
              title="Delete list"
              @click.stop="onDeleteList(list.id)"
            >✕</button>
          </li>
        </ul>
      </aside>

      <!-- Right panel: selected list contents -->
      <main class="list-content">
        <template v-if="listsStore.currentList">
          <div class="list-header">
            <div>
              <h1>{{ listsStore.currentList.name }}</h1>
              <p v-if="listsStore.currentList.description" class="list-desc">
                {{ listsStore.currentList.description }}
              </p>
            </div>
            <button class="btn-ghost" @click="openRename">Rename</button>
          </div>

          <div v-if="listsStore.loading" class="state-msg">Loading…</div>
          <div v-else-if="listsStore.currentBooks.length === 0" class="state-msg">
            No books in this list yet. Add books from the library.
          </div>
          <div v-else class="grid">
            <div
              v-for="book in listsStore.currentBooks"
              :key="book.id"
              class="book-item"
            >
              <BookCard
                :book="book"
                @show-detail="openDetail(book)"
              />
              <button
                class="remove-book btn-ghost"
                title="Remove from list"
                @click="onRemoveBook(book.id)"
              >Remove</button>
            </div>
          </div>
        </template>
        <div v-else class="state-msg placeholder">
          Select a list to see its books.
        </div>
      </main>
    </div>

    <!-- Create list modal -->
    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
      <div class="modal card">
        <h2>New list</h2>
        <div class="field">
          <label>Name</label>
          <input v-model="newName" type="text" placeholder="e.g. To Read" autofocus />
        </div>
        <div class="field">
          <label>Description (optional)</label>
          <input v-model="newDesc" type="text" />
        </div>
        <p v-if="createError" class="error-msg">{{ createError }}</p>
        <div class="modal-actions">
          <button class="btn-ghost" @click="showCreate = false">Cancel</button>
          <button class="btn-primary" :disabled="!newName.trim() || creating" @click="onCreate">
            {{ creating ? 'Creating…' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Rename list modal -->
    <div v-if="showRename" class="modal-overlay" @click.self="showRename = false">
      <div class="modal card">
        <h2>Rename list</h2>
        <div class="field">
          <label>Name</label>
          <input v-model="renameName" type="text" />
        </div>
        <div class="field">
          <label>Description</label>
          <input v-model="renameDesc" type="text" />
        </div>
        <div class="modal-actions">
          <button class="btn-ghost" @click="showRename = false">Cancel</button>
          <button class="btn-primary" :disabled="!renameName.trim()" @click="onRename">Save</button>
        </div>
      </div>
    </div>

    <!-- Book detail (open from list) -->
    <BookDetailModal
      v-if="selectedBook"
      :book="selectedBook"
      @close="selectedBook = null"
      @open-reader="openReader"
      @edit="() => {}"
      @deleted="selectedBook = null"
      @add-to-list="() => {}"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import BookCard from '@/components/BookCard.vue'
import BookDetailModal from '@/components/BookDetailModal.vue'
import { useListsStore } from '@/stores/lists'
import type { Book } from '@/types'

const listsStore = useListsStore()
const router = useRouter()

const showCreate = ref(false)
const newName = ref('')
const newDesc = ref('')
const creating = ref(false)
const createError = ref('')

const showRename = ref(false)
const renameName = ref('')
const renameDesc = ref('')

const selectedBook = ref<Book | null>(null)

onMounted(() => listsStore.fetchLists())

async function onCreate() {
  creating.value = true
  createError.value = ''
  try {
    const list = await listsStore.createList(newName.value.trim(), newDesc.value.trim())
    showCreate.value = false
    newName.value = ''
    newDesc.value = ''
    listsStore.selectList(list)
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    createError.value = msg ?? 'Failed to create list.'
  } finally {
    creating.value = false
  }
}

async function onDeleteList(id: string) {
  if (!confirm('Delete this list? Books are not removed from the library.')) return
  await listsStore.deleteList(id)
}

function openRename() {
  if (!listsStore.currentList) return
  renameName.value = listsStore.currentList.name
  renameDesc.value = listsStore.currentList.description
  showRename.value = true
}

async function onRename() {
  if (!listsStore.currentList) return
  await listsStore.updateList(listsStore.currentList.id, {
    name: renameName.value.trim(),
    description: renameDesc.value.trim(),
  })
  showRename.value = false
}

async function onRemoveBook(bookId: string) {
  if (!listsStore.currentList) return
  await listsStore.removeBook(listsStore.currentList.id, bookId)
}

function openDetail(book: Book) {
  selectedBook.value = book
}

function openReader(book: Book) {
  selectedBook.value = null
  router.push(`/books/${book.id}`)
}
</script>

<style scoped>
.lists-page {
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

/* Lists panel */
.lists-panel {
  width: 240px;
  flex-shrink: 0;
  position: sticky;
  top: 72px;
  max-height: calc(100vh - 90px);
  overflow-y: auto;
  padding: 1rem;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}
.panel-header h2 {
  font-size: 1rem;
  font-weight: 600;
}
.new-btn {
  padding: 0.35rem 0.75rem;
  font-size: 0.8rem;
}
.list-index {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}
.list-entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.45rem 0.6rem;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-muted);
  font-size: 0.9rem;
  transition: background 0.1s, color 0.1s;
}
.list-entry:hover { background: var(--surface-hover); color: var(--text); }
.list-entry.active { background: var(--surface-hover); color: var(--accent); }
.list-name {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  flex: 1;
}
.list-delete {
  opacity: 0;
  padding: 0.1rem 0.4rem;
  font-size: 0.7rem;
  flex-shrink: 0;
  margin-left: 0.4rem;
}
.list-entry:hover .list-delete { opacity: 1; }

/* Main content */
.list-content {
  flex: 1;
  min-width: 0;
}
.list-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.5rem;
}
.list-header h1 {
  font-size: 1.4rem;
  font-weight: 700;
}
.list-desc {
  color: var(--text-muted);
  font-size: 0.9rem;
  margin-top: 0.25rem;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 1.25rem;
}
.book-item {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.remove-book {
  font-size: 0.75rem;
  padding: 0.25rem 0.6rem;
  width: 100%;
}

.state-msg {
  text-align: center;
  color: var(--text-muted);
  padding: 3rem 0;
}
.placeholder { padding: 6rem 0; }

/* Modal */
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
  max-width: 400px;
  padding: 1.5rem;
}
.modal h2 { margin-bottom: 1rem; font-size: 1.1rem; }
.field {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  margin-bottom: 0.75rem;
}
.field label { font-size: 0.82rem; color: var(--text-muted); }
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.75rem;
}
</style>
