<template>
  <div class="reader-page">
    <NavBar />

    <div class="reader-body">
      <!-- Header toolbar — hidden until ready, but always in DOM for layout -->
      <div class="reader-header" v-show="!loading && !epubError">
        <button class="btn-ghost back-btn" @click="$router.back()">← Library</button>
        <div class="book-info">
          <span class="book-title">{{ book?.title }}</span>
          <span class="book-author" v-if="book?.author">{{ book.author }}</span>
        </div>

        <div v-if="ttsSupported" class="tts-controls">
          <span v-if="ttsActive && !ttsPaused" class="tts-indicator">▶ Speaking</span>
          <button class="btn-ghost tts-btn" @click="toggleTts">{{ ttsBtnLabel }}</button>
          <button v-if="ttsActive || ttsPaused" class="btn-ghost" @click="stopTts">Stop</button>
          <label class="tts-rate-wrap">
            <span class="tts-rate-label">{{ ttsRate.toFixed(1) }}×</span>
            <input
              type="range"
              min="0.5"
              max="2"
              step="0.1"
              v-model.number="ttsRate"
              @change="onRateChange"
              class="tts-rate-slider"
            />
          </label>
          <label class="tts-auto-label">
            <input type="checkbox" v-model="ttsAutoAdvance" />
            <span>Auto-advance</span>
          </label>
        </div>
      </div>

      <!-- Loading overlay -->
      <div v-if="loading" class="state-overlay">
        <div class="spinner"></div>
        <span>Loading book…</span>
      </div>

      <!-- Error overlay -->
      <div v-else-if="epubError" class="state-overlay error">
        <p>{{ epubError }}</p>
        <button class="btn-ghost" @click="$router.back()">← Back to Library</button>
      </div>

      <!-- Non-EPUB fallback -->
      <div v-else-if="book && book.file_type !== 'epub'" class="state-overlay">
        <div class="card fallback-card">
          <p>In-browser reading is only supported for EPUB files.</p>
          <a :href="downloadUrl" class="btn-primary" download>Download {{ book.file_type.toUpperCase() }}</a>
        </div>
      </div>

      <!-- epub.js renders here — always in DOM so renderTo() can run immediately -->
      <div
        ref="epubContainer"
        class="epub-container"
        :style="{ visibility: (loading || epubError || (book && book.file_type !== 'epub')) ? 'hidden' : 'visible' }"
      ></div>

      <!-- Footer nav — hidden until ready -->
      <div class="reader-footer" v-show="!loading && !epubError && book?.file_type === 'epub'">
        <button class="btn-ghost nav-btn" @click="prevPage">← Prev</button>
        <div class="progress-wrap">
          <div class="progress-track">
            <div class="progress-fill" :style="{ width: currentPercentage + '%' }"></div>
          </div>
          <span class="progress-label">{{ currentPercentage }}%</span>
        </div>
        <button class="btn-ghost nav-btn" @click="nextPage">Next →</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import Epub from 'epubjs'
import type { Book as EpubBook, Rendition, Contents, Location } from 'epubjs'
import NavBar from '@/components/NavBar.vue'
import { useBooksStore } from '@/stores/books'
import client from '@/api/client'
import { contentUrl } from '@/api/books'

const route = useRoute()
const books = useBooksStore()

// DOM ref
const epubContainer = ref<HTMLDivElement | null>(null)

// Non-reactive epubjs instances
let epubBook: EpubBook | null = null
let rendition: Rendition | null = null
let pageText = ''
let saveTimer: ReturnType<typeof setTimeout> | null = null

// Reactive UI state
const loading = ref(true)
const epubError = ref<string | null>(null)
const currentPercentage = ref(0)
const currentCfi = ref<string | null>(null)

// TTS state
const ttsSupported = ref(false)
const ttsActive = ref(false)
const ttsPaused = ref(false)
const ttsAutoAdvance = ref(false)
const ttsRate = ref(1.0)

// Store data
const book = computed(() => books.currentBook)
const progress = computed(() => books.currentProgress)
const downloadUrl = computed(() => contentUrl(route.params.id as string))

const ttsBtnLabel = computed(() => {
  if (ttsActive.value && !ttsPaused.value) return 'Pause'
  if (ttsPaused.value) return 'Resume'
  return 'Read Aloud'
})

// ── epub.js ──────────────────────────────────────────────────────────────────

async function initEpub(arrayBuffer: ArrayBuffer): Promise<void> {
  if (!epubContainer.value) throw new Error('epub container not mounted')

  // ArrayBuffer overload exists at runtime but the TS type only declares string
  epubBook = Epub(arrayBuffer as unknown as string)

  rendition = epubBook.renderTo(epubContainer.value, {
    flow: 'paginated',
    width: '100%',
    height: '100%',
    allowScriptedContent: false,
  })

  rendition.themes.default({
    body: {
      background: '#ffffff',
      color: '#1a1a1a',
      'font-family': 'Georgia, serif',
      'line-height': '1.7',
      padding: '0 2rem',
    },
  })

  // Capture page text for TTS whenever content renders
  rendition.hooks.content.register((contents: Contents) => {
    pageText = (contents.document.body as HTMLElement)?.innerText ?? ''
  })

  // Track reading position
  rendition.on('relocated', (location: Location) => {
    currentCfi.value = location?.start?.cfi ?? null
    currentPercentage.value = Math.round((location?.start?.percentage ?? 0) * 100)
    scheduleSaveProgress()
  })

  await rendition.display(progress.value?.cfi ?? undefined)
  loading.value = false
}

// ── Progress ─────────────────────────────────────────────────────────────────

function scheduleSaveProgress(): void {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    flushProgress()
  }, 5000)
}

function flushProgress(): void {
  if (saveTimer) { clearTimeout(saveTimer); saveTimer = null }
  const id = route.params.id as string
  const cfi = currentCfi.value
  if (cfi) {
    books.saveProgress(id, cfi, currentPercentage.value / 100)
  }
}

// ── Navigation ───────────────────────────────────────────────────────────────

async function prevPage(): Promise<void> {
  stopTts()
  await rendition?.prev()
}

async function nextPage(): Promise<void> {
  stopTts()
  await rendition?.next()
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'ArrowLeft') prevPage()
  else if (e.key === 'ArrowRight') nextPage()
}

// ── TTS ──────────────────────────────────────────────────────────────────────

function speakCurrentPage(): void {
  if (!ttsSupported.value || !pageText.trim()) return

  const utt = new SpeechSynthesisUtterance(pageText)
  utt.rate = ttsRate.value
  utt.lang = book.value?.metadata?.language ?? 'en'

  utt.onstart = () => {
    ttsActive.value = true
    ttsPaused.value = false
  }

  utt.onend = () => {
    ttsActive.value = false
    ttsPaused.value = false
    if (ttsAutoAdvance.value) {
      rendition?.next().then(() => {
        setTimeout(() => {
          if (ttsAutoAdvance.value) speakCurrentPage()
        }, 300)
      })
    }
  }

  utt.onerror = (e) => {
    if (e.error !== 'interrupted') {
      epubError.value = `TTS error: ${e.error}`
    }
    ttsActive.value = false
    ttsPaused.value = false
  }

  window.speechSynthesis.cancel()
  window.speechSynthesis.speak(utt)
}

function toggleTts(): void {
  if (!ttsActive.value && !ttsPaused.value) {
    speakCurrentPage()
  } else if (ttsActive.value && !ttsPaused.value) {
    window.speechSynthesis.pause()
    ttsPaused.value = true
  } else {
    window.speechSynthesis.resume()
    ttsPaused.value = false
  }
}

function stopTts(): void {
  ttsAutoAdvance.value = false
  window.speechSynthesis.cancel()
  ttsActive.value = false
  ttsPaused.value = false
}

function onRateChange(): void {
  if (ttsActive.value || ttsPaused.value) {
    const wasAutoAdvance = ttsAutoAdvance.value
    stopTts()
    ttsAutoAdvance.value = wasAutoAdvance
    speakCurrentPage()
  }
}

// ── Lifecycle ────────────────────────────────────────────────────────────────

onMounted(async () => {
  ttsSupported.value = 'speechSynthesis' in window
  window.addEventListener('keydown', onKeydown)

  const id = route.params.id as string

  try {
    await Promise.all([books.fetchBook(id), books.fetchProgress(id)])
  } catch {
    epubError.value = 'Failed to load book metadata.'
    loading.value = false
    return
  }

  if (book.value?.file_type !== 'epub') {
    loading.value = false
    return
  }

  let arrayBuffer: ArrayBuffer
  try {
    const response = await client.get<ArrayBuffer>(`/books/${id}/content`, {
      responseType: 'arraybuffer',
    })
    arrayBuffer = response.data
  } catch {
    epubError.value = 'Failed to fetch EPUB file.'
    loading.value = false
    return
  }

  try {
    await initEpub(arrayBuffer)
  } catch (err) {
    epubError.value = `Failed to open EPUB: ${err instanceof Error ? err.message : String(err)}`
    loading.value = false
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  window.speechSynthesis.cancel()
  flushProgress()
  rendition?.destroy()
  epubBook?.destroy()
  rendition = null
  epubBook = null
})
</script>

<style scoped>
.reader-page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg);
}

.reader-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

/* ── Header ────────────────────────────────────────────────────────────────── */
.reader-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem 1.25rem;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
  flex-shrink: 0;
  flex-wrap: wrap;
}

.back-btn { flex-shrink: 0; }

.book-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  min-width: 0;
}

.book-title {
  font-weight: 600;
  font-size: 0.95rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.book-author {
  font-size: 0.8rem;
  color: var(--text-muted);
}

/* ── TTS Controls ──────────────────────────────────────────────────────────── */
.tts-controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.tts-indicator {
  font-size: 0.78rem;
  color: var(--accent);
  font-weight: 600;
  animation: pulse 1.4s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50%       { opacity: 0.45; }
}

.tts-btn { min-width: 5.5rem; }

.tts-rate-wrap {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  cursor: default;
}

.tts-rate-label {
  font-size: 0.8rem;
  color: var(--text-muted);
  min-width: 2.5rem;
  text-align: right;
}

.tts-rate-slider {
  width: 80px;
  accent-color: var(--accent);
  cursor: pointer;
}

.tts-auto-label {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.82rem;
  color: var(--text-muted);
  cursor: pointer;
  white-space: nowrap;
}

.tts-auto-label input[type='checkbox'] {
  accent-color: var(--accent);
  width: auto;
  cursor: pointer;
}

/* ── State overlays ────────────────────────────────────────────────────────── */
.state-overlay {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  color: var(--text-muted);
  font-size: 0.95rem;
}

.state-overlay.error { color: var(--danger); }

.spinner {
  width: 36px;
  height: 36px;
  border: 3px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.fallback-card {
  max-width: 420px;
  padding: 2rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1.25rem;
  text-align: center;
}

/* ── epub container ────────────────────────────────────────────────────────── */
.epub-container {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

/* ── Footer ────────────────────────────────────────────────────────────────── */
.reader-footer {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.5rem 1.25rem;
  border-top: 1px solid var(--border);
  background: var(--surface);
  flex-shrink: 0;
}

.nav-btn { flex-shrink: 0; }

.progress-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
}

.progress-track {
  flex: 1;
  height: 4px;
  background: var(--border);
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 2px;
  transition: width 0.4s ease;
}

.progress-label {
  font-size: 0.8rem;
  color: var(--text-muted);
  min-width: 2.75rem;
  text-align: right;
}
</style>
