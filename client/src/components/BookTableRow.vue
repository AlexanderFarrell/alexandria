<template>
  <tr class="book-row" :class="{ 'is-selected': selected }" @click="$emit('show-detail', book)">
    <td class="col-check" @click.stop>
      <input
        type="checkbox"
        :checked="selected"
        @change="$emit('toggle-select', book.id)"
      />
    </td>
    <td class="col-cover">
      <div class="thumb-wrap">
        <img v-if="coverSrc" :src="coverSrc" alt="" class="cover-thumb" />
        <div v-else class="cover-thumb cover-placeholder">
          <span>{{ initials }}</span>
        </div>
      </div>
    </td>
    <td class="col-title">
      <span class="book-title">{{ book.title }}</span>
      <span class="book-author">{{ book.author }}</span>
    </td>
    <td class="col-genres">
      <span v-for="g in visibleGenres" :key="g" class="genre-chip">{{ g }}</span>
      <span v-if="extraGenres > 0" class="genre-more">+{{ extraGenres }}</span>
    </td>
    <td class="col-publisher">{{ book.metadata?.publisher || '—' }}</td>
    <td class="col-year">{{ year || '—' }}</td>
    <td class="col-type">
      <span class="type-badge">{{ book.file_type }}</span>
    </td>
    <td class="col-size">{{ formattedSize }}</td>
  </tr>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Book } from '@/types'
import { useBookCover } from '@/composables/useBookCover'

const props = defineProps<{
  book: Book
  selected: boolean
}>()

defineEmits<{
  (e: 'toggle-select', id: string): void
  (e: 'show-detail', book: Book): void
}>()

const coverSrc = useBookCover(
  () => props.book.id,
  () => !!props.book.cover_path,
)

const initials = computed(() => {
  const words = props.book.title.split(/\s+/).filter(Boolean)
  return words
    .slice(0, 2)
    .map((w) => w[0].toUpperCase())
    .join('')
})

const year = computed(() => {
  const d = props.book.metadata?.published_at
  if (!d) return null
  const y = new Date(d).getFullYear()
  return isNaN(y) ? null : y
})

const allGenres = computed(() => props.book.metadata?.genres ?? [])
const visibleGenres = computed(() => allGenres.value.slice(0, 2))
const extraGenres = computed(() => Math.max(0, allGenres.value.length - 2))

const formattedSize = computed(() => {
  const bytes = props.book.file_size
  if (!bytes) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
})
</script>

<style scoped>
.book-row {
  cursor: pointer;
  border-bottom: 1px solid var(--border);
  transition: background 0.1s;
}
.book-row:hover { background: var(--surface-hover); }
.book-row.is-selected { background: color-mix(in srgb, var(--accent) 10%, transparent); }

td {
  padding: 0.55rem 0.85rem;
  vertical-align: middle;
  font-size: 0.875rem;
  color: var(--text);
}

.col-check { width: 2.5rem; text-align: center; }
.col-check input[type='checkbox'] {
  cursor: pointer;
  width: 1rem;
  height: 1rem;
  accent-color: var(--accent);
}

.col-cover { width: 3.5rem; padding: 0.35rem 0.6rem; }
.thumb-wrap {
  width: 2.5rem;
  height: 3.75rem;
  overflow: hidden;
  border-radius: 3px;
}
.cover-thumb { width: 100%; height: 100%; object-fit: cover; display: block; }
.cover-placeholder {
  width: 100%;
  height: 100%;
  background: var(--surface-hover);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.65rem;
  font-weight: 700;
  color: var(--text-muted);
}

.col-title { min-width: 14rem; }
.book-title { display: block; font-weight: 600; }
.book-author { display: block; font-size: 0.8rem; color: var(--text-muted); margin-top: 0.15rem; }

.col-genres { min-width: 8rem; }
.genre-chip {
  display: inline-block;
  background: color-mix(in srgb, var(--accent) 18%, transparent);
  color: var(--accent);
  border-radius: 3px;
  padding: 0.15rem 0.45rem;
  font-size: 0.72rem;
  font-weight: 600;
  margin-right: 0.25rem;
}
.genre-more { font-size: 0.75rem; color: var(--text-muted); font-weight: 500; }

.col-publisher, .col-year, .col-size {
  color: var(--text-muted);
  white-space: nowrap;
  font-size: 0.85rem;
}

.type-badge {
  display: inline-block;
  background: var(--surface-hover);
  border: 1px solid var(--border);
  border-radius: 3px;
  padding: 0.15rem 0.5rem;
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-muted);
}
</style>
