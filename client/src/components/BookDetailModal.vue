<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal card">
      <button class="close-btn btn-ghost" @click="$emit('close')">✕</button>

      <div class="modal-body">
        <!-- Left column: cover + rating -->
        <div class="left-col">
          <div class="cover-wrap">
            <img
              v-if="coverSrc"
              :src="coverSrc"
              :alt="book.title"
              class="cover"
            />
            <div v-else class="cover-placeholder">
              <span>{{ initials }}</span>
            </div>
          </div>
          <RatingStars
            :rating="localRating"
            :interactive="true"
            class="rating"
            @rate="onRate"
          />
          <p v-if="localRating" class="rating-label">{{ localRating }}/5</p>
          <p v-else class="rating-label unrated">Not rated</p>
        </div>

        <!-- Right column: metadata -->
        <div class="right-col">
          <h2 class="title">{{ book.title }}</h2>
          <p class="author">{{ book.author || 'Unknown author' }}</p>

          <p v-if="book.description" class="description">{{ book.description }}</p>

          <div class="meta-grid">
            <template v-if="book.metadata.publisher">
              <span class="meta-label">Publisher</span>
              <span>{{ book.metadata.publisher }}</span>
            </template>
            <template v-if="book.metadata.language">
              <span class="meta-label">Language</span>
              <span>{{ book.metadata.language }}</span>
            </template>
            <template v-if="book.metadata.published_at">
              <span class="meta-label">Published</span>
              <span>{{ formatDate(book.metadata.published_at) }}</span>
            </template>
            <template v-if="book.metadata.isbn">
              <span class="meta-label">ISBN</span>
              <span>{{ book.metadata.isbn }}</span>
            </template>
          </div>

          <div v-if="book.metadata.genres?.length" class="chips-row">
            <span
              v-for="g in book.metadata.genres"
              :key="g"
              class="chip"
            >{{ g }}</span>
          </div>

          <div v-if="book.metadata.tags?.length" class="chips-row">
            <span
              v-for="t in book.metadata.tags"
              :key="t"
              class="chip chip-muted"
            >{{ t }}</span>
          </div>

          <div v-if="progress" class="progress-row">
            <div class="progress-bar-wrap">
              <div class="progress-bar" :style="{ width: `${Math.round(progress.percentage * 100)}%` }" />
            </div>
            <span class="progress-label">{{ Math.round(progress.percentage * 100) }}% read</span>
          </div>
        </div>
      </div>

      <!-- Action row -->
      <div class="modal-actions">
        <button class="btn-danger" @click="onDelete">Delete</button>
        <div class="right-actions">
          <button class="btn-ghost" @click="$emit('add-to-list', book)">+ Add to list</button>
          <button
            v-if="book.file_type !== 'url'"
            class="btn-ghost"
            :disabled="refreshing"
            @click="onRefreshMetadata"
          >{{ refreshing ? 'Refreshing…' : 'Refresh metadata' }}</button>
          <button class="btn-ghost" @click="$emit('edit', book)">Edit</button>
          <button class="btn-primary" @click="$emit('open-reader', book)">Read →</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useBookCover } from '@/composables/useBookCover'
import { useBooksStore } from '@/stores/books'
import type { Book, ReadingProgress } from '@/types'
import RatingStars from './RatingStars.vue'

const props = defineProps<{
  book: Book
  progress?: ReadingProgress | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'open-reader', book: Book): void
  (e: 'edit', book: Book): void
  (e: 'deleted'): void
  (e: 'add-to-list', book: Book): void
}>()

const booksStore = useBooksStore()

const localRating = ref<number | undefined>(props.progress?.rating)
const refreshing = ref(false)

watch(() => props.progress?.rating, (val) => {
  localRating.value = val
})

const coverSrc = useBookCover(
  () => props.book.id,
  () => Boolean(props.book.cover_path),
)

const initials = computed(() => {
  const words = props.book.title.split(' ').slice(0, 2)
  return words.map((w) => w[0]?.toUpperCase() ?? '').join('')
})

async function onRate(n: number) {
  localRating.value = n
  await booksStore.rateBook(props.book.id, n)
}

async function onRefreshMetadata() {
  refreshing.value = true
  try {
    await booksStore.refreshBookMetadata(props.book.id)
  } finally {
    refreshing.value = false
  }
}

async function onDelete() {
  if (!confirm(`Delete "${props.book.title}"? This cannot be undone.`)) return
  await booksStore.deleteBook(props.book.id)
  emit('deleted')
}

function formatDate(iso: string) {
  try {
    return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'long' })
  } catch {
    return iso
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
  z-index: 200;
  padding: 1rem;
}
.modal {
  width: 100%;
  max-width: 760px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}
.close-btn {
  position: absolute;
  top: 0.75rem;
  right: 0.75rem;
  padding: 0.3rem 0.6rem;
  font-size: 0.85rem;
  z-index: 1;
}
.modal-body {
  display: flex;
  gap: 1.5rem;
  padding: 1.5rem;
  overflow-y: auto;
  flex: 1;
}
.left-col {
  flex-shrink: 0;
  width: 140px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}
.cover-wrap {
  width: 140px;
  aspect-ratio: 2 / 3;
  overflow: hidden;
  background: var(--surface-hover);
  border-radius: var(--radius);
}
.cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.cover-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 2.5rem;
  font-weight: 700;
  color: var(--accent-dim);
  font-family: var(--font-body);
}
.rating {
  font-size: 1.3rem;
}
.rating-label {
  font-size: 0.75rem;
  color: var(--text-muted);
}
.rating-label.unrated {
  font-style: italic;
}
.right-col {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.title {
  font-size: 1.3rem;
  font-weight: 700;
  line-height: 1.3;
}
.author {
  color: var(--text-muted);
  font-size: 0.95rem;
}
.description {
  font-size: 0.9rem;
  line-height: 1.6;
  color: var(--text);
  max-height: 120px;
  overflow-y: auto;
}
.meta-grid {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 0.3rem 1rem;
  font-size: 0.85rem;
}
.meta-label {
  color: var(--text-muted);
  white-space: nowrap;
}
.chips-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}
.chip {
  background: var(--surface-hover);
  border: 1px solid var(--border);
  border-radius: 99px;
  padding: 0.2rem 0.7rem;
  font-size: 0.8rem;
  color: var(--accent);
}
.chip-muted {
  color: var(--text-muted);
}
.progress-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.progress-bar-wrap {
  flex: 1;
  height: 4px;
  background: var(--border);
  border-radius: 2px;
  overflow: hidden;
}
.progress-bar {
  height: 100%;
  background: var(--accent);
  border-radius: 2px;
}
.progress-label {
  font-size: 0.8rem;
  color: var(--text-muted);
  white-space: nowrap;
}
.modal-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border);
  gap: 0.75rem;
}
.right-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

@media (max-width: 600px) {
  .modal-body { flex-direction: column; padding: 1rem; gap: 1rem; }
  .left-col { flex-direction: row; width: 100%; gap: 1rem; align-items: flex-start; }
  .cover-wrap { width: 90px; flex-shrink: 0; }
  .modal-actions { flex-wrap: wrap; padding: 0.75rem 1rem; gap: 0.5rem; }
  .right-actions { flex-wrap: wrap; gap: 0.5rem; }
}
</style>
