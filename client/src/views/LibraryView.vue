<template>
  <div class="library-page">
    <NavBar />

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
        <button class="btn-primary" @click="showUpload = true">+ Add book</button>
      </div>

      <div v-if="books.loading" class="state-msg">Loading…</div>
      <div v-else-if="books.books.length === 0" class="state-msg empty">
        <p>No books yet.</p>
        <button class="btn-primary" @click="showUpload = true">Add your first book</button>
      </div>
      <div v-else class="grid">
        <BookCard
          v-for="book in books.books"
          :key="book.id"
          :book="book"
          @open="openBook"
        />
      </div>

      <div v-if="books.total > 0" class="pagination">
        <span class="total">{{ books.total }} book{{ books.total !== 1 ? 's' : '' }}</span>
      </div>
    </main>

    <!-- Upload modal -->
    <div v-if="showUpload" class="modal-overlay" @click.self="showUpload = false">
      <div class="modal card">
        <h2>Add a book</h2>
        <form @submit.prevent="uploadBook">
          <div class="field">
            <label>File (EPUB or PDF)</label>
            <input type="file" accept=".epub,.pdf" @change="onFileChange" required />
          </div>
          <div class="field">
            <label>Title (optional override)</label>
            <input v-model="uploadTitle" type="text" placeholder="Extracted from file if empty" />
          </div>
          <div class="field">
            <label>Author (optional override)</label>
            <input v-model="uploadAuthor" type="text" placeholder="Extracted from file if empty" />
          </div>
          <p v-if="uploadError" class="error-msg">{{ uploadError }}</p>
          <div class="modal-actions">
            <button type="button" class="btn-ghost" @click="showUpload = false">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="!uploadFile || uploading">
              {{ uploading ? 'Uploading…' : 'Upload' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import BookCard from '@/components/BookCard.vue'
import { useBooksStore } from '@/stores/books'
import type { Book } from '@/types'

const books = useBooksStore()
const router = useRouter()

const search = ref('')
const showUpload = ref(false)
const uploadFile = ref<File | null>(null)
const uploadTitle = ref('')
const uploadAuthor = ref('')
const uploadError = ref('')
const uploading = ref(false)

let searchTimer: ReturnType<typeof setTimeout>

onMounted(() => books.fetchBooks())

function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => books.fetchBooks({ search: search.value }), 300)
}

function openBook(book: Book) {
  router.push(`/books/${book.id}`)
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  uploadFile.value = input.files?.[0] ?? null
}

async function uploadBook() {
  if (!uploadFile.value) return
  uploading.value = true
  uploadError.value = ''
  try {
    await books.uploadBook(uploadFile.value, {
      title: uploadTitle.value || undefined,
      author: uploadAuthor.value || undefined,
    })
    showUpload.value = false
    uploadFile.value = null
    uploadTitle.value = ''
    uploadAuthor.value = ''
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    uploadError.value = msg ?? 'Upload failed. Try again.'
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.library-page {
  min-height: 100vh;
  background: var(--bg);
}
.content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
}
.search-wrap { flex: 1; max-width: 420px; }

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
.total { color: var(--text-muted); }

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
  max-width: 460px;
  padding: 2rem;
}
.modal h2 {
  margin-bottom: 1.5rem;
  font-size: 1.1rem;
}
.modal form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
.field label { font-size: 0.85rem; color: var(--text-muted); }
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}
</style>
