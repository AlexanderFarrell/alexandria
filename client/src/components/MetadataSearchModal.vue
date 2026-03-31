<template>
  <div class="search-overlay" @click.self="$emit('close')">
    <div class="search-modal card">
      <div class="search-header">
        <h2>Search Online</h2>
        <button class="btn-ghost btn-icon" type="button" @click="$emit('close')">✕</button>
      </div>

      <div class="search-bar">
        <input
          v-model="query.title"
          type="text"
          placeholder="Title"
          @keydown.enter="doSearch"
        />
        <input
          v-model="query.author"
          type="text"
          placeholder="Author"
          @keydown.enter="doSearch"
        />
        <input
          v-model="query.isbn"
          type="text"
          placeholder="ISBN"
          @keydown.enter="doSearch"
        />
        <button class="btn-primary" type="button" :disabled="searching" @click="doSearch">
          {{ searching ? 'Searching…' : 'Search' }}
        </button>
      </div>

      <div v-if="providerErrors.length" class="provider-errors">
        <span
          v-for="pe in providerErrors"
          :key="pe.provider"
          class="provider-error-badge"
          :title="pe.message"
        >
          {{ pe.provider }}: unavailable
        </span>
      </div>

      <div v-if="searchError" class="error-msg">{{ searchError }}</div>

      <div v-if="searching" class="spinner-wrap">
        <div class="spinner" />
      </div>

      <div v-else-if="results.length" class="results-grid">
        <button
          v-for="(result, idx) in results"
          :key="idx"
          type="button"
          class="result-card"
          :class="{ selected: selected === result }"
          @click="selected = selected === result ? null : result"
        >
          <img
            v-if="result.cover_url"
            :src="result.cover_url"
            class="result-cover"
            alt=""
            loading="lazy"
          />
          <div v-else class="result-cover result-cover-placeholder">📖</div>
          <div class="result-info">
            <div class="result-title">{{ result.title }}</div>
            <div class="result-authors">{{ result.authors.join(', ') }}</div>
            <div class="result-meta">
              <span v-if="result.publisher">{{ result.publisher }}</span>
              <span v-if="result.publisher && result.published_date"> · </span>
              <span v-if="result.published_date">{{ result.published_date.slice(0, 4) }}</span>
            </div>
            <span class="source-badge">{{ result.source.name }}</span>
          </div>
        </button>
      </div>

      <div v-else-if="searched && !searching" class="empty-state">
        No results found. Try adjusting your search.
      </div>

      <div v-if="selected" class="preview-panel">
        <div class="preview-body">
          <strong>{{ selected.title }}</strong>
          <span v-if="selected.authors.length"> — {{ selected.authors.join(', ') }}</span>
          <p v-if="selected.description" class="preview-desc">
            {{ selected.description.slice(0, 200) }}{{ selected.description.length > 200 ? '…' : '' }}
          </p>
        </div>
        <label v-if="selected.cover_url" class="cover-checkbox">
          <input v-model="replaceCover" type="checkbox" />
          Replace cover image
        </label>
      </div>

      <div class="search-actions">
        <button type="button" class="btn-ghost" @click="$emit('close')">Cancel</button>
        <button
          type="button"
          class="btn-primary"
          :disabled="!selected"
          @click="onApply"
        >
          Apply to form
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { searchMetadata } from '@/api/metadata'
import type { MetadataResult } from '@/types'

const props = defineProps<{
  initialTitle?: string
  initialAuthor?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'apply', result: MetadataResult, replaceCover: boolean): void
}>()

const query = reactive({
  title: props.initialTitle ?? '',
  author: props.initialAuthor ?? '',
  isbn: '',
})

const results = ref<MetadataResult[]>([])
const providerErrors = ref<Array<{ provider: string; message: string }>>([])
const searchError = ref('')
const searching = ref(false)
const searched = ref(false)
const selected = ref<MetadataResult | null>(null)
const replaceCover = ref(false)

async function doSearch() {
  if (!query.title && !query.author && !query.isbn) return
  searching.value = true
  searchError.value = ''
  providerErrors.value = []
  selected.value = null
  try {
    const res = await searchMetadata({
      title: query.title || undefined,
      author: query.author || undefined,
      isbn: query.isbn || undefined,
    })
    results.value = res.results
    providerErrors.value = res.errors ?? []
    searched.value = true
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    searchError.value = msg ?? 'Search failed. Please try again.'
  } finally {
    searching.value = false
  }
}

function onApply() {
  if (!selected.value) return
  emit('apply', selected.value, replaceCover.value)
}

onMounted(() => {
  if (query.title || query.author) {
    doSearch()
  }
})
</script>

<style scoped>
.search-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 400;
  padding: 1rem;
}
.search-modal {
  width: 100%;
  max-width: 680px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.5rem;
  overflow: hidden;
}
.search-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.search-header h2 {
  font-size: 1.1rem;
  margin: 0;
}
.btn-icon {
  font-size: 1rem;
  padding: 0.25rem 0.5rem;
}
.search-bar {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.search-bar input {
  flex: 1;
  min-width: 100px;
}
.provider-errors {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
}
.provider-error-badge {
  font-size: 0.72rem;
  padding: 0.15rem 0.5rem;
  background: var(--color-warning, #7a4f00);
  color: #fff;
  border-radius: 999px;
  cursor: default;
}
.spinner-wrap {
  display: flex;
  justify-content: center;
  padding: 2rem;
}
.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border, #333);
  border-top-color: var(--color-primary, #7c6af7);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
.results-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 0.75rem;
  overflow-y: auto;
  max-height: 320px;
}
.result-card {
  display: flex;
  gap: 0.75rem;
  padding: 0.6rem;
  border: 1px solid var(--border, #333);
  border-radius: 8px;
  background: var(--card-bg, #1e1e1e);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s;
  width: 100%;
}
.result-card:hover {
  border-color: var(--color-primary, #7c6af7);
}
.result-card.selected {
  border-color: var(--color-primary, #7c6af7);
  background: color-mix(in srgb, var(--color-primary, #7c6af7) 12%, var(--card-bg, #1e1e1e));
}
.result-cover {
  width: 52px;
  height: 72px;
  object-fit: cover;
  border-radius: 4px;
  flex-shrink: 0;
  background: var(--border, #333);
}
.result-cover-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.5rem;
}
.result-info {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
}
.result-title {
  font-size: 0.88rem;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.result-authors {
  font-size: 0.78rem;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.result-meta {
  font-size: 0.75rem;
  color: var(--text-muted);
}
.source-badge {
  font-size: 0.68rem;
  padding: 0.1rem 0.4rem;
  background: var(--border, #333);
  border-radius: 999px;
  margin-top: auto;
  align-self: flex-start;
}
.empty-state {
  text-align: center;
  color: var(--text-muted);
  padding: 1.5rem;
  font-size: 0.9rem;
}
.preview-panel {
  border: 1px solid var(--border, #333);
  border-radius: 8px;
  padding: 0.75rem;
  font-size: 0.85rem;
}
.preview-desc {
  margin-top: 0.35rem;
  color: var(--text-muted);
  font-size: 0.8rem;
}
.cover-checkbox {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  margin-top: 0.5rem;
  font-size: 0.82rem;
  cursor: pointer;
}
.search-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}
</style>
