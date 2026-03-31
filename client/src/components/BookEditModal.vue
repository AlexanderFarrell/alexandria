<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal card">
      <div class="modal-title-row">
        <h2>Edit book</h2>
        <button type="button" class="btn-ghost btn-sm" @click="showMetadataSearch = true">
          Search Online
        </button>
      </div>
      <form @submit.prevent="onSubmit">
        <div class="fields">
          <div class="field">
            <label>Title</label>
            <input v-model="form.title" type="text" />
          </div>
          <div class="field">
            <label>Author</label>
            <input v-model="form.author" type="text" />
          </div>
          <div class="field">
            <label>Description</label>
            <textarea v-model="form.description" rows="3" />
          </div>
          <div class="field">
            <label>Publisher</label>
            <input v-model="form.publisher" type="text" />
          </div>
          <div class="field-row">
            <div class="field">
              <label>Language</label>
              <input v-model="form.language" type="text" placeholder="e.g. en" />
            </div>
            <div class="field">
              <label>Published date</label>
              <input v-model="form.publishedAt" type="date" />
            </div>
          </div>
          <div class="field">
            <label>ISBN</label>
            <input v-model="form.isbn" type="text" />
          </div>
          <div class="field">
            <label>Genres <span class="hint">(comma-separated)</span></label>
            <input v-model="form.genresRaw" type="text" placeholder="e.g. Fantasy, Science Fiction" />
          </div>
          <div class="field">
            <label>Tags <span class="hint">(comma-separated)</span></label>
            <input v-model="form.tagsRaw" type="text" placeholder="e.g. classic, recommended" />
          </div>
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>

        <div class="modal-actions">
          <button type="button" class="btn-ghost" @click="$emit('close')">Cancel</button>
          <button type="submit" class="btn-primary" :disabled="saving">
            {{ saving ? 'Saving…' : 'Save changes' }}
          </button>
        </div>
      </form>
    </div>
  </div>

  <MetadataSearchModal
    v-if="showMetadataSearch"
    :initial-title="form.title"
    :initial-author="form.author"
    @close="showMetadataSearch = false"
    @apply="onApplyMetadata"
  />
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useBooksStore } from '@/stores/books'
import type { Book, MetadataResult } from '@/types'
import MetadataSearchModal from './MetadataSearchModal.vue'

const props = defineProps<{ book: Book }>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved', book: Book): void
}>()

const booksStore = useBooksStore()
const saving = ref(false)
const error = ref('')
const showMetadataSearch = ref(false)
const pendingCoverURL = ref<string | null>(null)

const meta = props.book.metadata

function toDateInput(iso?: string) {
  if (!iso) return ''
  try { return iso.slice(0, 10) } catch { return '' }
}

const form = reactive({
  title: props.book.title,
  author: props.book.author,
  description: props.book.description,
  publisher: meta.publisher ?? '',
  language: meta.language ?? '',
  isbn: meta.isbn ?? '',
  publishedAt: toDateInput(meta.published_at),
  genresRaw: (meta.genres ?? []).join(', '),
  tagsRaw: (meta.tags ?? []).join(', '),
})

function splitComma(raw: string): string[] {
  return raw.split(',').map((s) => s.trim()).filter(Boolean)
}

function onApplyMetadata(result: MetadataResult, replaceCover: boolean) {
  if (result.title) form.title = result.title
  if (result.authors.length) form.author = result.authors.join(', ')
  if (result.description) form.description = result.description
  if (result.publisher) form.publisher = result.publisher
  if (result.language) form.language = result.language
  if (result.isbn) form.isbn = result.isbn
  if (result.published_date) {
    const d = result.published_date
    form.publishedAt = d.length === 4 ? d + '-01-01' : d.slice(0, 10)
  }
  if (result.tags?.length) form.genresRaw = result.tags.join(', ')
  pendingCoverURL.value = replaceCover && result.cover_url ? result.cover_url : null
  showMetadataSearch.value = false
}

async function onSubmit() {
  saving.value = true
  error.value = ''
  try {
    const updatedBook = await booksStore.updateBook(props.book.id, {
      title: form.title,
      author: form.author,
      description: form.description,
      metadata: {
        publisher: form.publisher || undefined,
        language: form.language || undefined,
        isbn: form.isbn || undefined,
        published_at: form.publishedAt || undefined,
        genres: splitComma(form.genresRaw),
        tags: splitComma(form.tagsRaw),
      },
      ...(pendingCoverURL.value ? { cover_url: pendingCoverURL.value } : {}),
    })
    emit('saved', updatedBook)
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg ?? 'Save failed. Try again.'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
  padding: 1rem;
}
.modal {
  width: 100%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
  padding: 1.5rem;
}
.modal-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.25rem;
}
.modal-title-row h2 {
  font-size: 1.1rem;
  margin: 0;
}
.btn-sm {
  font-size: 0.8rem;
  padding: 0.3rem 0.7rem;
}
.fields {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}
.field-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}
.field label {
  font-size: 0.82rem;
  color: var(--text-muted);
}
.hint {
  font-weight: 400;
  font-size: 0.75rem;
}
textarea {
  resize: vertical;
  min-height: 70px;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.25rem;
}
</style>
