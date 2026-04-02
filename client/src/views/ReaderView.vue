<template>
  <div class="reader-page">
    <NavBar compact-mobile />

    <div class="reader-body">
      <div
        class="reader-header"
        :class="{ 'is-epub-reader': ttsSupported && book?.file_type === 'epub' }"
        v-show="!loading && !epubError"
      >
        <button class="btn-ghost back-btn" @click="$router.back()">← Library</button>

        <div class="book-info">
          <span class="book-title">{{ book?.title }}</span>
          <span class="book-author" v-if="book?.author">{{ book.author }}</span>
        </div>

        <div v-if="ttsSupported && book?.file_type === 'epub'" class="tts-toolbar">
          <span class="tts-status" :class="ttsStatusClass">
            <span class="tts-status-full">{{ ttsStatusLabel }}</span>
            <span class="tts-status-compact">{{ ttsStatusCompactLabel }}</span>
          </span>
          <button
            class="btn-ghost tts-btn tts-control"
            :class="{ 'is-active': ttsMode === 'speaking' || ttsMode === 'paused' }"
            :disabled="!canStartTts"
            :aria-label="ttsBtnLabel"
            :title="ttsBtnLabel"
            @click="toggleTts"
          >
            <span class="tts-control-icon" aria-hidden="true">{{ ttsBtnIcon }}</span>
            <span class="tts-control-label">{{ ttsBtnLabel }}</span>
          </button>
          <button
            class="btn-ghost tts-control"
            :disabled="ttsMode === 'idle'"
            aria-label="Stop"
            title="Stop"
            @click="stopTts"
          >
            <span class="tts-control-icon" aria-hidden="true">■</span>
            <span class="tts-control-label">Stop</span>
          </button>
          <button
            class="btn-ghost tts-control"
            :class="{ 'is-active': ttsMode === 'selecting' }"
            :disabled="!canChooseParagraph"
            :aria-label="chooseParagraphLabel"
            :title="chooseParagraphLabel"
            @click="toggleParagraphPicker"
          >
            <span class="tts-control-icon" aria-hidden="true">{{ chooseParagraphIcon }}</span>
            <span class="tts-control-label">{{ chooseParagraphLabel }}</span>
          </button>
          <button
            class="btn-ghost tts-control"
            :class="{ 'is-active': ttsSettingsOpen }"
            :aria-label="settingsBtnLabel"
            :title="settingsBtnLabel"
            @click="ttsSettingsOpen = !ttsSettingsOpen"
          >
            <span class="tts-control-icon" aria-hidden="true">⚙</span>
            <span class="tts-control-label">{{ settingsBtnLabel }}</span>
          </button>

          <div v-if="selectedStartBlockIndex >= 0" class="tts-start-chip">
            <span>Start: paragraph {{ selectedStartBlockIndex + 1 }}</span>
            <button class="btn-ghost btn-inline" @click="clearStartBlock">Clear</button>
          </div>
        </div>
      </div>

      <div
        v-if="ttsSupported && ttsSettingsOpen && !loading && !epubError && book?.file_type === 'epub'"
        class="tts-settings-panel"
      >
        <label class="tts-field">
          <span class="tts-field-label">Voice</span>
          <select v-model="selectedVoiceUri" class="tts-select" @change="onVoiceChange">
            <option value="">Default system voice</option>
            <option v-for="voice in voiceOptions" :key="voice.voiceURI" :value="voice.voiceURI">
              {{ formatVoiceLabel(voice) }}
            </option>
          </select>
        </label>

        <label class="tts-field">
          <span class="tts-field-label">Rate</span>
          <div class="tts-slider-wrap">
            <span class="tts-slider-value">{{ ttsPrefs.rate.toFixed(1) }}×</span>
            <input
              v-model.number="ttsPrefs.rate"
              class="tts-slider"
              type="range"
              min="0.5"
              max="2"
              step="0.1"
              @change="onSpeechSettingChange"
            />
          </div>
        </label>

        <label class="tts-field">
          <span class="tts-field-label">Pitch</span>
          <div class="tts-slider-wrap">
            <span class="tts-slider-value">{{ ttsPrefs.pitch.toFixed(1) }}</span>
            <input
              v-model.number="ttsPrefs.pitch"
              class="tts-slider"
              type="range"
              min="0"
              max="2"
              step="0.1"
              @change="onSpeechSettingChange"
            />
          </div>
        </label>

        <label class="tts-field">
          <span class="tts-field-label">Volume</span>
          <div class="tts-slider-wrap">
            <span class="tts-slider-value">{{ Math.round(ttsPrefs.volume * 100) }}%</span>
            <input
              v-model.number="ttsPrefs.volume"
              class="tts-slider"
              type="range"
              min="0"
              max="1"
              step="0.05"
              @change="onSpeechSettingChange"
            />
          </div>
        </label>

        <label class="tts-checkbox">
          <input v-model="ttsPrefs.autoAdvance" type="checkbox" @change="onAutoAdvanceChange" />
          <span>Auto-scroll while reading</span>
        </label>

        <label class="tts-checkbox">
          <input v-model="readerPrefs.darkMode" type="checkbox" @change="onReaderThemeChange" />
          <span>Dark reader mode</span>
        </label>
      </div>

      <p
        v-if="ttsError && !loading && !epubError && book?.file_type === 'epub'"
        class="tts-feedback"
      >
        {{ ttsError }}
      </p>

      <div v-if="loading" class="state-overlay">
        <div class="spinner"></div>
        <span>Loading book…</span>
      </div>

      <div v-else-if="epubError" class="state-overlay error">
        <p>{{ epubError }}</p>
        <button class="btn-ghost" @click="$router.back()">← Back to Library</button>
      </div>

      <div v-else-if="book && book.file_type === 'pdf'" class="pdf-viewer">
        <iframe
          v-if="pdfBlobUrl"
          :src="pdfBlobUrl"
          class="pdf-iframe"
          title="PDF viewer"
        />
        <div v-else class="state-overlay">
          <div class="spinner"></div>
          <span>Loading PDF…</span>
        </div>
        <div class="pdf-toolbar">
          <p v-if="downloadError" class="download-error">{{ downloadError }}</p>
          <button class="btn-ghost" :disabled="downloading" @click="downloadBook">
            {{ downloading ? 'Downloading…' : 'Download PDF' }}
          </button>
        </div>
      </div>

      <div v-else-if="book && book.file_type !== 'epub'" class="state-overlay">
        <div class="card fallback-card">
          <p>In-browser reading is only supported for EPUB files.</p>
          <p v-if="downloadError" class="download-error">{{ downloadError }}</p>
          <button class="btn-primary" :disabled="downloading" @click="downloadBook">
            {{ downloading ? `Downloading ${book.file_type.toUpperCase()}…` : `Download ${book.file_type.toUpperCase()}` }}
          </button>
        </div>
      </div>

      <div
        ref="epubContainer"
        class="epub-container"
        :style="{
          visibility: (loading || epubError || (book && book.file_type !== 'epub')) ? 'hidden' : 'visible',
          background: readerPrefs.darkMode ? '#111318' : '#ffffff',
        }"
      ></div>

      <div class="reader-footer" v-show="!loading && !epubError && book?.file_type === 'epub'">
        <button class="btn-ghost nav-btn" @click="prevPage">↑ Up</button>
        <div class="progress-wrap">
          <div class="progress-track">
            <div class="progress-fill" :style="{ width: currentPercentage + '%' }"></div>
          </div>
          <span class="progress-label">{{ currentPercentage }}%</span>
        </div>
        <button class="btn-ghost nav-btn" @click="nextPage">↓ Down</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, shallowRef } from 'vue'
import { useRoute } from 'vue-router'
import Epub from 'epubjs'
import type { Book as EpubBook, Contents, Location, Rendition } from 'epubjs'
import NavBar from '@/components/NavBar.vue'
import { useBooksStore } from '@/stores/books'
import client from '@/api/client'
import { getContentBlob } from '@/api/books'

const READABLE_BLOCK_SELECTOR = 'p, li, blockquote, dd, dt, figcaption, h1, h2, h3, h4, h5, h6'
const TTS_PREFS_KEY = 'alexandria.reader.tts'
const READER_PREFS_KEY = 'alexandria.reader.display'
const TTS_STYLE_KEY = 'alexandria-reader-tts'
const TTS_BLOCK_ATTR = 'data-tts-block-id'
const TTS_SELECTED_ATTR = 'data-tts-selected'
const TTS_ACTIVE_ATTR = 'data-tts-active'
const TTS_SELECTING_CLASS = 'alexandria-tts-selecting'
const READER_THEME_LIGHT = 'alexandria-reader-light'
const READER_THEME_DARK = 'alexandria-reader-dark'
const TTS_VIEWPORT_OVERSCAN_PX = 260
const TTS_HORIZONTAL_OVERSCAN_PX = 24
const EPUB_SECTION_OVERSCAN_PX = 1200
const EPUB_SECTION_OVERSCAN_DELTA_PX = 480
const TTS_IFRAME_STYLES = `
  [${TTS_BLOCK_ATTR}] {
    transition: background-color 0.16s ease, box-shadow 0.16s ease, outline-color 0.16s ease;
    border-radius: 0.3rem;
  }

  body.${TTS_SELECTING_CLASS} [${TTS_BLOCK_ATTR}] {
    cursor: pointer;
  }

  body.${TTS_SELECTING_CLASS} [${TTS_BLOCK_ATTR}]:hover {
    background: rgba(200, 169, 110, 0.14);
    outline: 1px solid rgba(200, 169, 110, 0.45);
  }

  [${TTS_SELECTED_ATTR}] {
    background: rgba(200, 169, 110, 0.12);
    outline: 1px solid rgba(200, 169, 110, 0.4);
  }

  [${TTS_ACTIVE_ATTR}] {
    background: rgba(200, 169, 110, 0.22);
    box-shadow: inset 0 0 0 1px rgba(200, 169, 110, 0.65);
  }
`

type TtsMode = 'idle' | 'speaking' | 'paused' | 'selecting'

interface TtsPreferences {
  voiceURI: string
  rate: number
  pitch: number
  volume: number
  autoAdvance: boolean
}

interface ReadableBlock {
  cfi: string
  text: string
  sectionIndex: number
  element: HTMLElement
  contents: Contents
  inViewport: boolean
}

interface ReaderPreferences {
  darkMode: boolean
}

interface StopTtsOptions {
  preserveSelected?: boolean
}

type ReaderRenditionOptions = NonNullable<Parameters<EpubBook['renderTo']>[1]> & {
  method: 'blobUrl'
  offset: number
  offsetDelta: number
}

const route = useRoute()
const books = useBooksStore()

const epubContainer = ref<HTMLDivElement | null>(null)

let epubBook: EpubBook | null = null
let rendition: Rendition | null = null
let currentLocationState: Location | null = null
let saveTimer: ReturnType<typeof setTimeout> | null = null
let refreshBlocksFrame: number | null = null
let autoAdvanceTimer: ReturnType<typeof setTimeout> | null = null
let speechToken = 0
let pendingPageTurnResume = false
let pageTurnPreviousLocation: string | null = null

const contentCleanups = new Map<Document, () => void>()

const loading = ref(true)
const epubError = ref<string | null>(null)
const downloadError = ref<string | null>(null)
const pdfBlobUrl = ref<string | null>(null)
const ttsError = ref<string | null>(null)
const currentPercentage = ref(0)
const currentCfi = ref<string | null>(null)
const downloading = ref(false)

const ttsSupported = ref(false)
const ttsMode = ref<TtsMode>('idle')
const ttsSettingsOpen = ref(false)
const visibleBlocks = shallowRef<ReadableBlock[]>([])
const fallbackPageText = ref('')
const availableVoices = shallowRef<SpeechSynthesisVoice[]>([])
const selectedStartBlockCfi = ref<string | null>(null)
const currentSpokenBlockCfi = ref<string | null>(null)
const ttsUsingFallback = ref(false)
const ttsPrefs = reactive<TtsPreferences>(loadTtsPreferences())
const readerPrefs = reactive<ReaderPreferences>(loadReaderPreferences())

const book = computed(() => books.currentBook)
const progress = computed(() => books.currentProgress)
const preferredLanguage = computed(() => book.value?.metadata?.language ?? navigator.language ?? 'en')

const voiceOptions = computed(() => {
  return [...availableVoices.value].sort((a, b) => compareVoices(a, b, preferredLanguage.value))
})

const selectedVoiceUri = computed({
  get: () => {
    const match = availableVoices.value.some((voice) => voice.voiceURI === ttsPrefs.voiceURI)
    return match ? ttsPrefs.voiceURI : ''
  },
  set: (voiceURI: string) => {
    ttsPrefs.voiceURI = voiceURI
  },
})

const selectedStartBlockIndex = computed(() => {
  if (!selectedStartBlockCfi.value) return -1
  return visibleBlocks.value.findIndex((block) => block.cfi === selectedStartBlockCfi.value)
})

const currentSpokenBlockIndex = computed(() => {
  if (!currentSpokenBlockCfi.value) return -1
  return visibleBlocks.value.findIndex((block) => block.cfi === currentSpokenBlockCfi.value)
})

const hasVisibleBlocks = computed(() => visibleBlocks.value.length > 0)
const hasReadableText = computed(() => hasVisibleBlocks.value || fallbackPageText.value.length > 0)
const canStartTts = computed(() => ttsMode.value !== 'selecting' && hasReadableText.value)
const canChooseParagraph = computed(() => hasVisibleBlocks.value)
const TTS_DEBUG = false

const ttsBtnLabel = computed(() => {
  if (ttsMode.value === 'speaking') return 'Pause'
  if (ttsMode.value === 'paused') return 'Resume'
  return 'Read Aloud'
})

const ttsBtnIcon = computed(() => {
  if (ttsMode.value === 'speaking') return '⏸'
  return '▶'
})

const chooseParagraphLabel = computed(() => {
  return ttsMode.value === 'selecting' ? 'Cancel' : 'Choose paragraph'
})

const chooseParagraphIcon = computed(() => {
  return ttsMode.value === 'selecting' ? '✕' : '¶'
})

const settingsBtnLabel = computed(() => {
  return ttsSettingsOpen.value ? 'Hide settings' : 'Settings'
})

const ttsStatusLabel = computed(() => {
  if (!hasReadableText.value) return 'No readable text'
  if (ttsMode.value === 'selecting') return 'Choose a paragraph'
  if (ttsMode.value === 'paused') {
    return currentSpokenBlockIndex.value >= 0
      ? `Paused on paragraph ${currentSpokenBlockIndex.value + 1}`
      : 'Paused'
  }
  if (ttsMode.value === 'speaking') {
    return currentSpokenBlockIndex.value >= 0
      ? `Reading paragraph ${currentSpokenBlockIndex.value + 1}`
      : 'Reading aloud'
  }
  return 'Ready'
})

const ttsStatusCompactLabel = computed(() => {
  if (!hasReadableText.value) return 'No text'
  if (ttsMode.value === 'selecting') return 'Pick start'
  if (ttsMode.value === 'paused') return 'Paused'
  if (ttsMode.value === 'speaking') return 'Reading'
  return 'Ready'
})

const ttsStatusClass = computed(() => {
  if (!hasReadableText.value) return 'is-disabled'
  return `is-${ttsMode.value}`
})

function ttsDebugState(): Record<string, unknown> {
  return {
    mode: ttsMode.value,
    currentCfi: currentCfi.value,
    currentLocation: currentLocationSignature(),
    pendingPageTurnResume,
    pageTurnPreviousLocation,
    visibleBlockCount: visibleBlocks.value.length,
    fallbackTextLength: fallbackPageText.value.length,
    currentSpokenBlockCfi: currentSpokenBlockCfi.value,
    selectedStartBlockCfi: selectedStartBlockCfi.value,
    usingFallback: ttsUsingFallback.value,
    autoAdvance: ttsPrefs.autoAdvance,
  }
}

function logTtsDebug(event: string, details: Record<string, unknown> = {}): void {
  if (!TTS_DEBUG) return
  console.debug(`[ReaderView TTS] ${event}`, {
    ...details,
    state: ttsDebugState(),
  })
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function normalizeText(value: string): string {
  return value.replace(/\s+/g, ' ').trim()
}

function parseNumberPreference(value: unknown, fallback: number): number {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string') {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) return parsed
  }
  return fallback
}

function loadTtsPreferences(): TtsPreferences {
  const defaults: TtsPreferences = {
    voiceURI: '',
    rate: 1,
    pitch: 1,
    volume: 1,
    autoAdvance: false,
  }

  if (typeof window === 'undefined') return defaults

  try {
    const raw = window.localStorage.getItem(TTS_PREFS_KEY)
    if (!raw) return defaults

    const parsed = JSON.parse(raw) as Partial<TtsPreferences>
    return {
      voiceURI: typeof parsed.voiceURI === 'string' ? parsed.voiceURI : '',
      rate: clamp(parseNumberPreference(parsed.rate, defaults.rate), 0.5, 2),
      pitch: clamp(parseNumberPreference(parsed.pitch, defaults.pitch), 0, 2),
      volume: clamp(parseNumberPreference(parsed.volume, defaults.volume), 0, 1),
      autoAdvance: Boolean(parsed.autoAdvance),
    }
  } catch {
    return defaults
  }
}

function persistTtsPreferences(): void {
  try {
    window.localStorage.setItem(TTS_PREFS_KEY, JSON.stringify(ttsPrefs))
  } catch {
    // Ignore localStorage failures and keep the in-memory settings.
  }
}

function loadReaderPreferences(): ReaderPreferences {
  const defaults: ReaderPreferences = {
    darkMode: false,
  }

  if (typeof window === 'undefined') return defaults

  try {
    const raw = window.localStorage.getItem(READER_PREFS_KEY)
    if (!raw) return defaults

    const parsed = JSON.parse(raw) as Partial<ReaderPreferences>
    return {
      darkMode: Boolean(parsed.darkMode),
    }
  } catch {
    return defaults
  }
}

function persistReaderPreferences(): void {
  try {
    window.localStorage.setItem(READER_PREFS_KEY, JSON.stringify(readerPrefs))
  } catch {
    // Ignore localStorage failures and keep the in-memory settings.
  }
}

function readerThemeRules(darkMode: boolean): Record<string, Record<string, string>> {
  const bodyRules = {
    'font-family': 'Georgia, serif',
    'font-size': 'clamp(1rem, 0.98rem + 0.2vw, 1.05rem)',
    'line-height': '1.7',
    padding: '0 clamp(0.95rem, 4vw, 2rem) 1.6rem',
  }

  if (darkMode) {
    return {
      body: {
        ...bodyRules,
        background: '#111318',
        color: '#ece4d9',
      },
      a: {
        color: '#d7b777',
      },
      'h1, h2, h3, h4, h5, h6, strong, b': {
        color: '#f6efe5',
      },
      'img, svg, video, canvas': {
        'max-width': '100%',
        height: 'auto',
      },
      pre: {
        'white-space': 'pre-wrap',
      },
    }
  }

  return {
    body: {
      ...bodyRules,
      background: '#ffffff',
      color: '#1a1a1a',
    },
    a: {
      color: '#8b6a2f',
    },
    'h1, h2, h3, h4, h5, h6, strong, b': {
      color: '#1a1a1a',
    },
    'img, svg, video, canvas': {
      'max-width': '100%',
      height: 'auto',
    },
    pre: {
      'white-space': 'pre-wrap',
    },
  }
}

function registerReaderThemes(): void {
  if (!rendition) return
  rendition.themes.register(READER_THEME_LIGHT, readerThemeRules(false))
  rendition.themes.register(READER_THEME_DARK, readerThemeRules(true))
}

function applyReaderTheme(): void {
  if (!rendition) return
  rendition.themes.select(readerPrefs.darkMode ? READER_THEME_DARK : READER_THEME_LIGHT)
}

function normalizeLanguageTag(value: string | undefined): string {
  return (value ?? '').toLowerCase()
}

function baseLanguageTag(value: string | undefined): string {
  return normalizeLanguageTag(value).split('-')[0]
}

function compareVoices(
  left: SpeechSynthesisVoice,
  right: SpeechSynthesisVoice,
  preferredLang: string,
): number {
  const preferred = normalizeLanguageTag(preferredLang)
  const preferredBase = baseLanguageTag(preferredLang)

  const score = (voice: SpeechSynthesisVoice) => {
    const voiceLang = normalizeLanguageTag(voice.lang)
    const voiceBase = baseLanguageTag(voice.lang)

    if (voiceLang === preferred) return 0
    if (voiceBase && voiceBase === preferredBase) return 1
    if (voice.default) return 2
    return 3
  }

  return (
    score(left) - score(right) ||
    normalizeLanguageTag(left.lang).localeCompare(normalizeLanguageTag(right.lang)) ||
    left.name.localeCompare(right.name)
  )
}

function formatVoiceLabel(voice: SpeechSynthesisVoice): string {
  const language = voice.lang || 'Unknown language'
  return voice.default ? `${voice.name} (${language}, default)` : `${voice.name} (${language})`
}

function buildDownloadFilename(title: string, fileType: string): string {
  const safeTitle = title
    .trim()
    .replace(/[\\/:*?"<>|]+/g, '-')
    .replace(/\s+/g, ' ')

  return `${safeTitle || 'book'}.${fileType}`
}

async function downloadBook(): Promise<void> {
  if (!book.value || downloading.value) return

  downloadError.value = null
  downloading.value = true

  try {
    const blob = await getContentBlob(book.value.id)
    const objectUrl = URL.createObjectURL(blob)
    const link = document.createElement('a')

    link.href = objectUrl
    link.download = buildDownloadFilename(book.value.title, book.value.file_type)
    document.body.appendChild(link)
    link.click()
    link.remove()

    window.setTimeout(() => URL.revokeObjectURL(objectUrl), 0)
  } catch {
    downloadError.value = 'Failed to download file.'
  } finally {
    downloading.value = false
  }
}

async function initEpub(arrayBuffer: ArrayBuffer): Promise<void> {
  if (!epubContainer.value) throw new Error('epub container not mounted')

  epubBook = Epub(arrayBuffer as unknown as string)
  const renditionOptions: ReaderRenditionOptions = {
    manager: 'continuous',
    flow: 'scrolled-continuous',
    spread: 'none',
    width: '100%',
    height: '100%',
    offset: EPUB_SECTION_OVERSCAN_PX,
    offsetDelta: EPUB_SECTION_OVERSCAN_DELTA_PX,
    allowScriptedContent: false,
    method: 'blobUrl',
  }
  rendition = epubBook.renderTo(epubContainer.value, renditionOptions)

  registerReaderThemes()
  applyReaderTheme()

  rendition.hooks.content.register((contents: Contents) => {
    setupTtsContents(contents)
    scheduleVisibleBlocksRefresh()
  })

  rendition.hooks.unloaded.register((view: { contents?: Contents }) => {
    if (view.contents) teardownTtsContents(view.contents)
  })

  rendition.on('relocated', (location: Location) => {
    currentLocationState = location
    currentCfi.value = location?.start?.cfi ?? null
    currentPercentage.value = Math.round((location?.start?.percentage ?? 0) * 100)
    logTtsDebug('relocated', {
      startCfi: location?.start?.cfi ?? null,
      endCfi: location?.end?.cfi ?? null,
      startPage: location?.start?.displayed?.page ?? null,
      endPage: location?.end?.displayed?.page ?? null,
      atStart: location?.atStart ?? false,
      atEnd: location?.atEnd ?? false,
    })
    scheduleSaveProgress()
    scheduleVisibleBlocksRefresh()
  })

  await rendition.display(progress.value?.cfi ?? undefined)
  loading.value = false
  scheduleVisibleBlocksRefresh()
}

function scheduleSaveProgress(): void {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    flushProgress()
  }, 5000)
}

function flushProgress(): void {
  if (saveTimer) {
    clearTimeout(saveTimer)
    saveTimer = null
  }

  const id = route.params.id as string
  const cfi = currentCfi.value
  if (cfi) {
    books.saveProgress(id, cfi, currentPercentage.value / 100)
  }
}

async function prevPage(): Promise<void> {
  await turnPage('prev')
}

async function nextPage(): Promise<void> {
  await turnPage('next')
}

function clearPendingPageTurnResume(): void {
  logTtsDebug('clear-pending-page-turn-resume')
  pendingPageTurnResume = false
  pageTurnPreviousLocation = null
}

function currentLocationSignature(): string | null {
  const location = currentLocationState
  if (!location?.start) return currentCfi.value

  const index = location.start.index ?? ''
  const displayedPage = location.start.displayed?.page ?? ''
  const displayedTotal = location.start.displayed?.total ?? ''
  const cfi = location.start.cfi ?? ''

  return `${index}:${displayedPage}:${displayedTotal}:${cfi}`
}

function prepareTtsPageTurnResume(): void {
  logTtsDebug('prepare-page-turn-resume')
  cancelSpeechOutput()
  selectedStartBlockCfi.value = null
  currentSpokenBlockCfi.value = null
  ttsUsingFallback.value = false
  pendingPageTurnResume = true
  pageTurnPreviousLocation = currentLocationSignature()
  syncBlockMarkers()
}

async function turnPage(direction: 'next' | 'prev'): Promise<void> {
  const shouldResumeTts = ttsMode.value === 'speaking'
  const atBoundary =
    direction === 'next' ? currentLocationState?.atEnd === true : currentLocationState?.atStart === true

  logTtsDebug('turn-page:start', { direction, shouldResumeTts, atBoundary })

  if (shouldResumeTts && !atBoundary) {
    prepareTtsPageTurnResume()
  } else {
    internalStopTts({ preserveSelected: false })
  }

  try {
    if (direction === 'next') {
      await rendition?.next()
    } else {
      await rendition?.prev()
    }
    logTtsDebug('turn-page:await-complete', { direction })
  } catch {
    logTtsDebug('turn-page:error', { direction })
    if (shouldResumeTts) internalStopTts({ preserveSelected: false })
  }
}

function onKeydown(event: KeyboardEvent): void {
  const target = event.target as HTMLElement | null
  const tagName = target?.tagName?.toLowerCase()
  const isFormControl = ['input', 'textarea', 'select', 'button'].includes(tagName ?? '')

  if (isFormControl) return

  if (event.key === 'ArrowLeft') {
    event.preventDefault()
    void prevPage()
  } else if (event.key === 'ArrowRight') {
    event.preventDefault()
    void nextPage()
  }
}

function setupTtsContents(contents: Contents): void {
  void contents.addStylesheetCss(TTS_IFRAME_STYLES, TTS_STYLE_KEY)

  const documentRef = contents.document
  if (!documentRef || contentCleanups.has(documentRef)) {
    syncBlockMarkers()
    return
  }

  const onClick = (event: Event) => {
    if (ttsMode.value !== 'selecting') return

    const target = event.target as HTMLElement | null
    const blockElement = target?.closest<HTMLElement>(`[${TTS_BLOCK_ATTR}]`)
    if (!blockElement) return

    const index = Number(blockElement.getAttribute(TTS_BLOCK_ATTR))
    const block = visibleBlocks.value[index]
    if (!block) return

    event.preventDefault()
    event.stopPropagation()

    selectedStartBlockCfi.value = block.cfi
    currentSpokenBlockCfi.value = null
    ttsUsingFallback.value = false
    ttsMode.value = 'idle'
    syncBlockMarkers()
  }

  documentRef.addEventListener('click', onClick, true)
  contentCleanups.set(documentRef, () => {
    documentRef.removeEventListener('click', onClick, true)
  })

  syncBlockMarkers()
}

function teardownTtsContents(contents: Contents): void {
  const documentRef = contents.document
  if (!documentRef) return

  contentCleanups.get(documentRef)?.()
  contentCleanups.delete(documentRef)
}

function getRenderedContents(): Contents[] {
  if (!rendition) return []
  return ((rendition.getContents() as unknown) as Contents[]).filter(Boolean)
}

function scheduleVisibleBlocksRefresh(): void {
  if (refreshBlocksFrame !== null) window.cancelAnimationFrame(refreshBlocksFrame)

  refreshBlocksFrame = window.requestAnimationFrame(() => {
    refreshBlocksFrame = null
    refreshVisibleBlocks()
  })
}

function clearBlockMarkers(contentsList: Contents[]): void {
  for (const contents of contentsList) {
    const documentRef = contents.document
    const body = documentRef?.body
    if (!body) continue

    body.classList.toggle(TTS_SELECTING_CLASS, ttsMode.value === 'selecting')

    const marked = body.querySelectorAll<HTMLElement>(
      `[${TTS_BLOCK_ATTR}], [${TTS_SELECTED_ATTR}], [${TTS_ACTIVE_ATTR}]`,
    )

    marked.forEach((element) => {
      element.removeAttribute(TTS_BLOCK_ATTR)
      element.removeAttribute(TTS_SELECTED_ATTR)
      element.removeAttribute(TTS_ACTIVE_ATTR)
    })
  }
}

function hasNestedReadableDescendant(element: HTMLElement): boolean {
  const nested = element.querySelector<HTMLElement>(READABLE_BLOCK_SELECTOR)
  if (!nested) return false
  return normalizeText(nested.innerText || nested.textContent || '').length > 0
}

function elementStartRect(element: HTMLElement): DOMRect {
  const clientRects = Array.from(element.getClientRects())
  const firstRect = clientRects.find((rect) => rect.width > 0 && rect.height > 0)
  return firstRect ?? element.getBoundingClientRect()
}

function blockIntersectsViewport(
  element: HTMLElement,
  contentsWindow: Window,
  verticalOverscanPx: number,
  horizontalOverscanPx: number,
): boolean {
  const styles = contentsWindow.getComputedStyle(element)
  if (styles.display === 'none' || styles.visibility === 'hidden' || Number(styles.opacity) === 0) {
    return false
  }

  const rect = elementStartRect(element)
  if (rect.width <= 0 || rect.height <= 0) return false

  const frameElement = contentsWindow.frameElement as Element | null
  const readerRect = epubContainer.value?.getBoundingClientRect()

  if (frameElement && readerRect) {
    const frameRect = frameElement.getBoundingClientRect()
    const absoluteLeft = frameRect.left + rect.left
    const absoluteRight = frameRect.left + rect.right
    const absoluteTop = frameRect.top + rect.top
    const absoluteBottom = frameRect.top + rect.bottom

    const horizontalOverlap =
      Math.min(absoluteRight, readerRect.right + horizontalOverscanPx) -
      Math.max(absoluteLeft, readerRect.left - horizontalOverscanPx)
    const verticalOverlap =
      Math.min(absoluteBottom, readerRect.bottom + verticalOverscanPx) -
      Math.max(absoluteTop, readerRect.top - verticalOverscanPx)

    return horizontalOverlap > 12 && verticalOverlap > 8
  }

  const horizontalOverlap =
    Math.min(rect.right, contentsWindow.innerWidth + horizontalOverscanPx) -
    Math.max(rect.left, -horizontalOverscanPx)
  const verticalOverlap =
    Math.min(rect.bottom, contentsWindow.innerHeight + verticalOverscanPx) -
    Math.max(rect.top, -verticalOverscanPx)

  return horizontalOverlap > 12 && verticalOverlap > 8
}

function isBlockNearViewport(element: HTMLElement, contentsWindow: Window): boolean {
  return blockIntersectsViewport(
    element,
    contentsWindow,
    TTS_VIEWPORT_OVERSCAN_PX,
    TTS_HORIZONTAL_OVERSCAN_PX,
  )
}

function isBlockInViewport(element: HTMLElement, contentsWindow: Window): boolean {
  return blockIntersectsViewport(element, contentsWindow, 0, 0)
}

function blockCfiFromElement(contents: Contents, element: HTMLElement): string | null {
  try {
    return contents.cfiFromNode(element)
  } catch {
    try {
      const range = element.ownerDocument.createRange()
      range.selectNodeContents(element)
      return contents.cfiFromRange(range)
    } catch {
      return null
    }
  }
}

function buildFallbackPageText(): string {
  if (!rendition || !currentLocationState?.start?.cfi || !currentLocationState?.end?.cfi) {
    return normalizeText(
      getRenderedContents()
        .map((contents) => contents.document?.body?.innerText ?? '')
        .join(' '),
    )
  }

  try {
    const startRange = rendition.getRange(currentLocationState.start.cfi)
    const endRange = rendition.getRange(currentLocationState.end.cfi)
    const ownerDocument = startRange?.startContainer.ownerDocument ?? null

    if (startRange && endRange && ownerDocument && ownerDocument === endRange.endContainer.ownerDocument) {
      const range = ownerDocument.createRange()
      range.setStart(startRange.startContainer, startRange.startOffset)
      range.setEnd(endRange.endContainer, endRange.endOffset)

      const text = normalizeText(range.toString())
      if (text) return text
    }
  } catch {
    // Fall back to visible document text below.
  }

  return normalizeText(
    getRenderedContents()
      .map((contents) => contents.document?.body?.innerText ?? '')
      .join(' '),
  )
}

function isBlockInCurrentLocation(cfi: string): boolean {
  const startCfi = currentLocationState?.start?.cfi
  const endCfi = currentLocationState?.end?.cfi
  if (!rendition || !startCfi || !endCfi) return true

  try {
    return rendition.epubcfi.compare(cfi, startCfi) >= 0 && rendition.epubcfi.compare(cfi, endCfi) <= 0
  } catch {
    return true
  }
}

function refreshVisibleBlocks(): void {
  const contentsList = getRenderedContents().sort((left, right) => left.sectionIndex - right.sectionIndex)
  clearBlockMarkers(contentsList)

  const nextBlocks: ReadableBlock[] = []

  for (const contents of contentsList) {
    const documentRef = contents.document
    const contentsWindow = contents.window
    if (!documentRef?.body || !contentsWindow) continue

    const candidates = Array.from(documentRef.querySelectorAll<HTMLElement>(READABLE_BLOCK_SELECTOR))

    for (const element of candidates) {
      const text = normalizeText(element.innerText || element.textContent || '')
      if (!text) continue
      if (hasNestedReadableDescendant(element)) continue
      if (!isBlockNearViewport(element, contentsWindow)) continue

      const cfi = blockCfiFromElement(contents, element)
      if (!cfi) continue
      if (!isBlockInCurrentLocation(cfi)) continue

      nextBlocks.push({
        cfi,
        text,
        sectionIndex: contents.sectionIndex,
        element,
        contents,
        inViewport: isBlockInViewport(element, contentsWindow),
      })
    }
  }

  visibleBlocks.value = nextBlocks
  fallbackPageText.value = nextBlocks.length > 0 ? '' : buildFallbackPageText()
  logTtsDebug('refresh-visible-blocks', {
    contentsCount: contentsList.length,
    nextBlocks: nextBlocks.length,
    firstBlockCfi: nextBlocks[0]?.cfi ?? null,
    lastBlockCfi: nextBlocks.length > 0 ? nextBlocks[nextBlocks.length - 1].cfi : null,
  })

  if (selectedStartBlockCfi.value && !nextBlocks.some((block) => block.cfi === selectedStartBlockCfi.value)) {
    selectedStartBlockCfi.value = null
  }

  if (ttsMode.value === 'selecting' && nextBlocks.length === 0) {
    ttsMode.value = 'idle'
  }

  if (
    currentSpokenBlockCfi.value &&
    !ttsUsingFallback.value &&
    !nextBlocks.some((block) => block.cfi === currentSpokenBlockCfi.value)
  ) {
    logTtsDebug('current-block-no-longer-visible', {
      currentSpokenBlockCfi: currentSpokenBlockCfi.value,
    })
    if (pendingPageTurnResume) {
      currentSpokenBlockCfi.value = null
    } else {
      internalStopTts({ preserveSelected: true })
    }
  }

  syncBlockMarkers()

  if (
    pendingPageTurnResume &&
    currentLocationSignature() !== pageTurnPreviousLocation &&
    hasReadableText.value
  ) {
    logTtsDebug('resume-after-page-turn')
    clearPendingPageTurnResume()
    if (visibleBlocks.value.length > 0) {
      speakBlockAtIndex(activeBlockIndex())
    } else {
      speakFallbackPageText()
    }
  }
}

function syncBlockMarkers(): void {
  const contentsList = getRenderedContents()

  for (const contents of contentsList) {
    contents.document?.body?.classList.toggle(TTS_SELECTING_CLASS, ttsMode.value === 'selecting')
  }

  visibleBlocks.value.forEach((block, index) => {
    block.element.setAttribute(TTS_BLOCK_ATTR, String(index))

    if (block.cfi === selectedStartBlockCfi.value) {
      block.element.setAttribute(TTS_SELECTED_ATTR, 'true')
    } else {
      block.element.removeAttribute(TTS_SELECTED_ATTR)
    }

    const isActiveBlock =
      !ttsUsingFallback.value &&
      (ttsMode.value === 'speaking' || ttsMode.value === 'paused') &&
      block.cfi === currentSpokenBlockCfi.value

    if (isActiveBlock) {
      block.element.setAttribute(TTS_ACTIVE_ATTR, 'true')
    } else {
      block.element.removeAttribute(TTS_ACTIVE_ATTR)
    }
  })
}

function clearAutoAdvanceTimer(): void {
  if (autoAdvanceTimer) {
    clearTimeout(autoAdvanceTimer)
    autoAdvanceTimer = null
  }
}

function cancelSpeechOutput(): void {
  logTtsDebug('cancel-speech-output')
  speechToken += 1
  clearAutoAdvanceTimer()
  if (ttsSupported.value) window.speechSynthesis.cancel()
}

function resolveVoice(): SpeechSynthesisVoice | null {
  return availableVoices.value.find((voice) => voice.voiceURI === ttsPrefs.voiceURI) ?? null
}

function applyUtteranceSettings(utterance: SpeechSynthesisUtterance): void {
  utterance.rate = ttsPrefs.rate
  utterance.pitch = ttsPrefs.pitch
  utterance.volume = ttsPrefs.volume

  const voice = resolveVoice()
  utterance.lang = voice?.lang || preferredLanguage.value || 'en'
  if (voice) utterance.voice = voice
}

function activeBlockIndex(): number {
  if (currentSpokenBlockIndex.value >= 0) return currentSpokenBlockIndex.value
  if (selectedStartBlockIndex.value >= 0) return selectedStartBlockIndex.value
  const firstViewportIndex = visibleBlocks.value.findIndex((block) => block.inViewport)
  if (firstViewportIndex >= 0) return firstViewportIndex
  return 0
}

function internalStopTts(options: StopTtsOptions = {}): void {
  logTtsDebug('internal-stop-tts', { preserveSelected: options.preserveSelected ?? false })
  cancelSpeechOutput()
  clearPendingPageTurnResume()
  ttsMode.value = 'idle'
  ttsUsingFallback.value = false
  currentSpokenBlockCfi.value = null
  if (!options.preserveSelected) selectedStartBlockCfi.value = null
  syncBlockMarkers()
}

function stopTts(): void {
  logTtsDebug('stop-tts')
  internalStopTts({ preserveSelected: true })
}

function handleTtsError(error: string): void {
  logTtsDebug('tts-error', { error })
  if (pendingPageTurnResume && (error === 'interrupted' || error === 'canceled')) {
    return
  }

  internalStopTts({ preserveSelected: true })
  if (error !== 'interrupted' && error !== 'canceled') {
    ttsError.value = `TTS error: ${error}`
  }
}

function speakFallbackPageText(): void {
  const text = fallbackPageText.value
  if (!ttsSupported.value || !text) return

  const token = speechToken + 1
  logTtsDebug('speak-fallback:start', {
    token,
    textLength: text.length,
  })
  cancelSpeechOutput()
  speechToken = token

  const utterance = new SpeechSynthesisUtterance(text)
  applyUtteranceSettings(utterance)

  ttsMode.value = 'speaking'
  ttsUsingFallback.value = true
  currentSpokenBlockCfi.value = null
  ttsError.value = null
  syncBlockMarkers()

  utterance.onend = () => {
    if (token !== speechToken) return

    logTtsDebug('speak-fallback:end', { token })
    ttsUsingFallback.value = false
    if (ttsPrefs.autoAdvance) {
      void queueAutoAdvance()
    } else {
      ttsMode.value = 'idle'
      syncBlockMarkers()
    }
  }

  utterance.onerror = (event) => {
    if (token !== speechToken) return
    logTtsDebug('speak-fallback:error', { token, error: event.error })
    handleTtsError(event.error)
  }

  window.speechSynthesis.speak(utterance)
}

function speakBlockAtIndex(index: number): void {
  const block = visibleBlocks.value[index]
  if (!block) {
    logTtsDebug('speak-block:missing', { index })
    internalStopTts({ preserveSelected: true })
    return
  }

  const token = speechToken + 1
  logTtsDebug('speak-block:start', {
    index,
    token,
    blockCfi: block.cfi,
    textLength: block.text.length,
    preview: block.text.slice(0, 80),
  })
  cancelSpeechOutput()
  speechToken = token

  const utterance = new SpeechSynthesisUtterance(block.text)
  applyUtteranceSettings(utterance)

  ttsMode.value = 'speaking'
  ttsUsingFallback.value = false
  currentSpokenBlockCfi.value = block.cfi
  ttsError.value = null
  syncBlockMarkers()

  utterance.onend = () => {
    if (token !== speechToken) return

    const nextIndex = index + 1
    logTtsDebug('speak-block:end', {
      index,
      token,
      nextIndex,
      visibleBlockCount: visibleBlocks.value.length,
    })
    if (nextIndex < visibleBlocks.value.length) {
      speakBlockAtIndex(nextIndex)
      return
    }

    currentSpokenBlockCfi.value = null
    if (ttsPrefs.autoAdvance) {
      void queueAutoAdvance()
    } else {
      ttsMode.value = 'idle'
      syncBlockMarkers()
    }
  }

  utterance.onerror = (event) => {
    if (token !== speechToken) return
    logTtsDebug('speak-block:error', { index, token, error: event.error, blockCfi: block.cfi })
    handleTtsError(event.error)
  }

  window.speechSynthesis.speak(utterance)
}

async function queueAutoAdvance(): Promise<void> {
  logTtsDebug('queue-auto-advance:start', {
    atEnd: currentLocationState?.atEnd ?? false,
  })
  if (!rendition || currentLocationState?.atEnd) {
    internalStopTts({ preserveSelected: true })
    return
  }

  prepareTtsPageTurnResume()

  autoAdvanceTimer = setTimeout(async () => {
    autoAdvanceTimer = null

    try {
      logTtsDebug('queue-auto-advance:fire')
      await rendition?.next()
    } catch {
      logTtsDebug('queue-auto-advance:error')
      internalStopTts({ preserveSelected: false })
    }
  }, 250)
}

async function startTts(): Promise<void> {
  logTtsDebug('start-tts')
  if (!ttsSupported.value || !hasReadableText.value || ttsMode.value === 'selecting') return

  ttsError.value = null

  if (visibleBlocks.value.length > 0) {
    speakBlockAtIndex(activeBlockIndex())
    return
  }

  speakFallbackPageText()
}

function pauseTts(): void {
  if (ttsMode.value !== 'speaking') return

  logTtsDebug('pause-tts')
  cancelSpeechOutput()
  clearPendingPageTurnResume()
  ttsMode.value = 'paused'
  syncBlockMarkers()
}

async function resumeTts(): Promise<void> {
  logTtsDebug('resume-tts')
  if (ttsMode.value !== 'paused') return

  if (ttsUsingFallback.value || visibleBlocks.value.length === 0) {
    speakFallbackPageText()
    return
  }

  speakBlockAtIndex(activeBlockIndex())
}

async function toggleTts(): Promise<void> {
  if (ttsMode.value === 'speaking') {
    pauseTts()
    return
  }

  if (ttsMode.value === 'paused') {
    await resumeTts()
    return
  }

  await startTts()
}

function toggleParagraphPicker(): void {
  if (!canChooseParagraph.value) return

  if (ttsMode.value === 'selecting') {
    ttsMode.value = 'idle'
    syncBlockMarkers()
    return
  }

  internalStopTts({ preserveSelected: true })
  ttsMode.value = 'selecting'
  ttsError.value = null
  syncBlockMarkers()
}

function clearStartBlock(): void {
  selectedStartBlockCfi.value = null
  syncBlockMarkers()
}

function restartCurrentPlayback(): void {
  if (ttsMode.value !== 'speaking') return

  logTtsDebug('restart-current-playback')
  if (ttsUsingFallback.value || visibleBlocks.value.length === 0) {
    speakFallbackPageText()
    return
  }

  speakBlockAtIndex(activeBlockIndex())
}

function onVoiceChange(): void {
  persistTtsPreferences()
  restartCurrentPlayback()
}

function onSpeechSettingChange(): void {
  persistTtsPreferences()
  restartCurrentPlayback()
}

function onAutoAdvanceChange(): void {
  persistTtsPreferences()
}

function onReaderThemeChange(): void {
  persistReaderPreferences()
  applyReaderTheme()
}

function loadVoices(): void {
  if (!ttsSupported.value) return
  availableVoices.value = window.speechSynthesis.getVoices()
}

onMounted(async () => {
  ttsSupported.value = 'speechSynthesis' in window && 'SpeechSynthesisUtterance' in window
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('resize', scheduleVisibleBlocksRefresh)

  if (ttsSupported.value) {
    loadVoices()
    window.speechSynthesis.addEventListener('voiceschanged', loadVoices)
  }

  const id = route.params.id as string

  try {
    await Promise.all([books.fetchBook(id), books.fetchProgress(id)])
  } catch {
    epubError.value = 'Failed to load book metadata.'
    loading.value = false
    return
  }

  if (book.value?.file_type === 'pdf') {
    loading.value = false
    try {
      const blob = await getContentBlob(id)
      pdfBlobUrl.value = URL.createObjectURL(blob)
    } catch {
      // Non-fatal — the toolbar still shows a download button
    }
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
  } catch (error) {
    epubError.value = `Failed to open EPUB: ${error instanceof Error ? error.message : String(error)}`
    loading.value = false
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize', scheduleVisibleBlocksRefresh)

  if (ttsSupported.value) {
    window.speechSynthesis.removeEventListener('voiceschanged', loadVoices)
    cancelSpeechOutput()
  }

  if (refreshBlocksFrame !== null) {
    window.cancelAnimationFrame(refreshBlocksFrame)
    refreshBlocksFrame = null
  }

  contentCleanups.forEach((cleanup) => cleanup())
  contentCleanups.clear()

  flushProgress()
  rendition?.destroy()
  epubBook?.destroy()
  rendition = null
  epubBook = null

  if (pdfBlobUrl.value) {
    URL.revokeObjectURL(pdfBlobUrl.value)
    pdfBlobUrl.value = null
  }
})
</script>

<style scoped>
.reader-page {
  height: 100vh;
  height: 100dvh;
  min-height: 100vh;
  min-height: 100dvh;
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

.back-btn {
  flex-shrink: 0;
  white-space: nowrap;
}

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
  line-height: 1.35;
}

.tts-toolbar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  justify-content: flex-end;
  min-width: min(100%, 34rem);
}

.tts-status {
  display: inline-flex;
  align-items: center;
  min-height: 2rem;
  max-width: 100%;
  padding: 0 0.7rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.03);
  color: var(--text-muted);
  font-size: 0.8rem;
  font-weight: 600;
  white-space: nowrap;
}

.tts-status-compact {
  display: none;
}

.tts-status.is-speaking {
  color: var(--accent);
  border-color: rgba(200, 169, 110, 0.4);
  background: rgba(200, 169, 110, 0.08);
}

.tts-status.is-paused,
.tts-status.is-selecting {
  color: var(--text);
}

.tts-status.is-disabled {
  opacity: 0.7;
}

.tts-btn {
  min-width: 6.25rem;
}

.tts-control {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
}

.tts-control-icon {
  font-size: 0.98rem;
  line-height: 1;
}

.tts-control.is-active {
  border-color: rgba(200, 169, 110, 0.45);
  color: var(--accent);
  background: rgba(200, 169, 110, 0.08);
}

.tts-start-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.2rem 0.25rem 0.2rem 0.7rem;
  border-radius: 999px;
  border: 1px solid rgba(200, 169, 110, 0.3);
  background: rgba(200, 169, 110, 0.08);
  color: var(--text);
  font-size: 0.8rem;
}

.btn-inline {
  padding: 0.3rem 0.6rem;
  font-size: 0.78rem;
}

.tts-settings-panel {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.85rem 1rem;
  padding: 0.9rem 1.25rem 1rem;
  border-bottom: 1px solid var(--border);
  background: rgba(26, 26, 36, 0.95);
}

.tts-field {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  min-width: 0;
}

.tts-field-label {
  font-size: 0.78rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.tts-select {
  width: 100%;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--text);
  font-family: inherit;
  font-size: 0.92rem;
  padding: 0.6rem 0.8rem;
  outline: none;
}

.tts-select:focus {
  border-color: var(--accent);
}

.tts-slider-wrap {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.tts-slider-value {
  width: 3.5rem;
  color: var(--text);
  font-size: 0.84rem;
  text-align: right;
}

.tts-slider {
  flex: 1;
  accent-color: var(--accent);
}

.tts-checkbox {
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  color: var(--text);
  font-size: 0.88rem;
  align-self: end;
}

.tts-checkbox input[type='checkbox'] {
  width: auto;
  accent-color: var(--accent);
}

.tts-feedback {
  padding: 0.75rem 1.25rem 0;
  color: var(--danger);
  font-size: 0.85rem;
  line-height: 1.35;
}

.download-error {
  color: var(--danger);
  font-size: 0.85rem;
  line-height: 1.35;
}

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

.state-overlay.error {
  color: var(--danger);
}

.spinner {
  width: 36px;
  height: 36px;
  border: 3px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
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

.epub-container {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.pdf-viewer {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.pdf-iframe {
  flex: 1;
  min-height: 0;
  width: 100%;
  border: none;
}

.pdf-toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 0.5rem 1rem calc(0.5rem + env(safe-area-inset-bottom));
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.reader-footer {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.5rem 1.25rem calc(0.5rem + env(safe-area-inset-bottom));
  border-top: 1px solid var(--border);
  background: var(--surface);
  flex-shrink: 0;
}

.nav-btn {
  flex-shrink: 0;
  min-width: 6rem;
}

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

@media (max-width: 900px) {
  .reader-header {
    padding-inline: 1rem;
  }

  .tts-toolbar {
    width: 100%;
    justify-content: flex-start;
  }

  .tts-settings-panel {
    padding-inline: 1rem;
  }

  .tts-feedback {
    padding-inline: 1rem;
  }

  .reader-footer {
    padding-inline: 1rem;
  }
}

@media (max-width: 760px) {
  .reader-header {
    padding: 0.7rem 0.85rem;
  }

  .reader-header.is-epub-reader {
    padding: 0.55rem 0.75rem;
  }

  .reader-header.is-epub-reader .back-btn,
  .reader-header.is-epub-reader .book-info {
    display: none;
  }

  .reader-header.is-epub-reader .tts-toolbar {
    width: 100%;
    min-width: 0;
    justify-content: flex-start;
    gap: 0.45rem;
  }

  .reader-header.is-epub-reader .tts-status {
    min-height: 0;
    padding: 0.35rem 0.65rem;
    font-size: 0.72rem;
  }

  .reader-header.is-epub-reader .tts-status-full {
    display: none;
  }

  .reader-header.is-epub-reader .tts-status-compact {
    display: inline;
  }

  .reader-header.is-epub-reader .tts-control {
    width: 2.7rem;
    min-width: 2.7rem;
    height: 2.7rem;
    padding: 0;
    border-radius: 0.85rem;
  }

  .reader-header.is-epub-reader .tts-control-label {
    display: none;
  }

  .reader-header.is-epub-reader .tts-control-icon {
    font-size: 1.02rem;
  }

  .reader-header.is-epub-reader .tts-start-chip {
    width: 100%;
    justify-content: space-between;
    flex-wrap: wrap;
    padding: 0.4rem 0.5rem 0.4rem 0.7rem;
  }

  .tts-settings-panel {
    grid-template-columns: 1fr;
    gap: 0.75rem;
    padding: 0.85rem 0.9rem 1rem;
  }

  .tts-slider-wrap {
    gap: 0.6rem;
  }

  .tts-slider-value {
    width: 3rem;
  }

  .tts-checkbox {
    align-self: start;
  }

  .tts-feedback {
    padding: 0.75rem 0.9rem 0;
  }

  .pdf-toolbar {
    flex-wrap: wrap;
    justify-content: stretch;
    padding: 0.65rem 0.9rem calc(0.65rem + env(safe-area-inset-bottom));
  }

  .pdf-toolbar > .btn-ghost,
  .pdf-toolbar > .download-error {
    width: 100%;
  }

  .reader-footer {
    display: none;
  }
}

@media (max-width: 480px) {
  .reader-header {
    padding: 0.6rem 0.75rem;
  }

  .reader-header.is-epub-reader {
    padding: 0.5rem 0.65rem;
  }

  .tts-start-chip {
    font-size: 0.76rem;
  }

  .btn-inline {
    padding: 0.28rem 0.55rem;
  }

  .reader-header.is-epub-reader .tts-control {
    width: 2.55rem;
    min-width: 2.55rem;
    height: 2.55rem;
  }
}
</style>
