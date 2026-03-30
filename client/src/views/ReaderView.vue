<template>
  <div class="reader-page">
    <NavBar />
    <div v-if="loading" class="state-msg">Loading…</div>
    <div v-else-if="book" class="reader-content">
      <div class="reader-header">
        <button class="btn-ghost back-btn" @click="$router.back()">← Library</button>
        <div class="book-info">
          <span class="book-title">{{ book.title }}</span>
          <span class="book-author" v-if="book.author">by {{ book.author }}</span>
        </div>
      </div>

      <!-- EPUB reader — epub.js integration arrives in v0.5 -->
      <div class="reader-placeholder">
        <div class="placeholder-card card">
          <p class="coming-soon">📖 Full in-browser reader coming in v0.5</p>
          <p class="placeholder-desc">
            epub.js rendering, page-turn controls, and position tracking will be added in the next milestone.
          </p>
          <a :href="contentUrl" class="btn-primary download-link" download>
            Download {{ book.file_type.toUpperCase() }}
          </a>
          <div v-if="progress" class="progress-info">
            <span>Last read position: {{ Math.round(progress.percentage * 100) }}% through</span>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="state-msg">Book not found.</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import { useBooksStore } from '@/stores/books'
import { contentUrl as getContentUrl } from '@/api/books'

const route = useRoute()
const books = useBooksStore()

const loading = ref(true)
const book = computed(() => books.currentBook)
const progress = computed(() => books.currentProgress)
const contentUrl = computed(() => getContentUrl(route.params.id as string))

onMounted(async () => {
  const id = route.params.id as string
  try {
    await books.fetchBook(id)
    await books.fetchProgress(id)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.reader-page {
  min-height: 100vh;
  background: var(--bg);
}
.state-msg {
  text-align: center;
  color: var(--text-muted);
  padding: 4rem;
}
.reader-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 2rem;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
}
.back-btn { flex-shrink: 0; }
.book-info {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}
.book-title { font-weight: 600; }
.book-author { font-size: 0.85rem; color: var(--text-muted); }

.reader-content { height: calc(100vh - 56px); display: flex; flex-direction: column; }
.reader-placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}
.placeholder-card {
  max-width: 480px;
  width: 100%;
  padding: 2.5rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  text-align: center;
}
.coming-soon { font-size: 1.1rem; font-weight: 600; }
.placeholder-desc { color: var(--text-muted); font-size: 0.9rem; line-height: 1.6; }
.download-link {
  display: inline-block;
  padding: 0.6rem 1.4rem;
  border-radius: var(--radius);
  background: var(--accent);
  color: #0f0f13;
  font-weight: 600;
  font-size: 0.9rem;
}
.progress-info {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin-top: 0.5rem;
}
</style>
