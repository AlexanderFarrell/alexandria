<template>
  <div class="reader-page" :style="shellStyle">
    <NavBar compact-mobile />

    <div class="reader-body">
      <div
        class="reader-header"
        :class="{ 'is-epub-reader': ttsSupported && book?.file_type === 'epub' }"
        v-show="!loading && !epubError"
      >
        <div class="reader-heading">
          <button class="btn-ghost back-btn" @click="$router.back()">← Library</button>

          <div class="book-info">
            <span class="book-title" :title="book?.title">{{ book?.title }}</span>
            <div
              v-if="book?.author || (book?.file_type === 'epub' && currentSection)"
              class="book-meta"
            >
              <span v-if="book?.author" class="book-author" :title="book.author">{{ book.author }}</span>
              <span
                v-if="book?.author && book?.file_type === 'epub' && currentSection"
                class="book-meta-divider"
              >
                ·
              </span>
              <span
                v-if="book?.file_type === 'epub' && currentSection"
                class="book-section"
                :title="currentSection.title"
              >
                {{ currentSection.title }}
              </span>
            </div>
          </div>
        </div>

        <div v-if="book?.file_type === 'epub'" class="reader-toolbar">
          <button class="btn-ghost toolbar-btn nav-toggle" @click="toggleNav">
            <span class="label-desktop">{{ navButtonLabel }}</span>
            <span class="label-mobile">{{ navButtonCompactLabel }}</span>
          </button>

          <button
            v-if="ttsSupported"
            class="btn-ghost toolbar-btn"
            :class="{ 'is-active': ttsMode === 'speaking' || ttsMode === 'paused' }"
            :disabled="!canStartTts"
            :title="ttsButtonLabel"
            @click="toggleTts"
          >
            <span class="label-desktop">{{ ttsButtonLabel }}</span>
            <span class="label-mobile">{{ ttsButtonCompactLabel }}</span>
          </button>
          <button
            v-if="ttsSupported"
            class="btn-ghost toolbar-btn toolbar-btn-secondary"
            :disabled="ttsMode === 'idle'"
            @click="stopTts"
          >
            Stop
          </button>
          <button
            v-if="ttsSupported"
            class="btn-ghost toolbar-btn toolbar-btn-secondary"
            :class="{ 'is-active': ttsMode === 'selecting' }"
            :disabled="!canChooseParagraph"
            @click="toggleParagraphPicker"
          >
            {{ chooseParagraphLabel }}
          </button>

          <div class="progress-chip" :title="progressChipTitle">
            <span class="progress-chip-value">{{ currentPercentage }}%</span>
            <span v-if="currentSectionPositionLabel" class="progress-chip-detail">
              {{ currentSectionPositionLabel }}
            </span>
          </div>

          <button
            class="btn-ghost toolbar-btn"
            :class="{ 'is-active': settingsOpen }"
            @click="toggleSettings"
          >
            Settings
          </button>

          <button class="btn-ghost toolbar-btn" @click="copyBookLink">
            {{ linkCopied ? 'Copied!' : 'Copy link' }}
          </button>
        </div>
      </div>

      <div
        v-if="book?.file_type === 'epub' && settingsOpen && !loading && !epubError"
        class="reader-settings-shell"
      >
        <button
          class="reader-settings-backdrop"
          type="button"
          aria-label="Close settings"
          @click="settingsOpen = false"
        ></button>

        <div class="reader-settings" role="dialog" aria-modal="true" aria-label="Reader settings">
          <div class="reader-settings-header">
            <div>
              <span class="settings-kicker">Reader</span>
              <h2>Settings</h2>
            </div>
            <button class="btn-ghost settings-close" @click="settingsOpen = false">Close</button>
          </div>

          <section v-if="ttsSupported" class="settings-group reader-mobile-actions">
            <h3>Quick actions</h3>
            <div class="settings-action-row">
              <button class="btn-ghost toolbar-btn" :disabled="ttsMode === 'idle'" @click="stopTts">
                Stop
              </button>
              <button
                class="btn-ghost toolbar-btn"
                :class="{ 'is-active': ttsMode === 'selecting' }"
                :disabled="!canChooseParagraph"
                @click="toggleParagraphPickerFromSettings"
              >
                {{ chooseParagraphLabel }}
              </button>
            </div>
          </section>

          <section class="settings-group">
            <h3>Theme</h3>
            <div class="theme-preset-row">
              <button
                v-for="preset in themePresets"
                :key="preset"
                class="theme-chip"
                :class="{ 'is-active': readerPrefs.themePreset === preset }"
                @click="setThemePreset(preset)"
              >
                {{ themePresetLabel(preset) }}
              </button>
            </div>

            <div v-if="readerPrefs.themePreset === 'custom'" class="theme-grid">
              <label v-for="field in themeFields" :key="field.key" class="theme-field">
                <span>{{ field.label }}</span>
                <input
                  v-model="readerPrefs.customTheme[field.key]"
                  type="color"
                  @change="persistReaderPreferencesAndApplyTheme"
                />
              </label>
            </div>
          </section>

          <section v-if="ttsSupported" class="settings-group">
            <h3>Read Aloud</h3>

            <label class="tts-field">
              <span>Voice</span>
              <select v-model="selectedVoiceUri" class="tts-select" @change="onVoiceChange">
                <option value="">Default system voice</option>
                <option v-for="voice in voiceOptions" :key="voice.voiceURI" :value="voice.voiceURI">
                  {{ formatVoiceLabel(voice) }}
                </option>
              </select>
            </label>

            <label class="tts-field">
              <span>Rate</span>
              <input
                v-model.number="ttsPrefs.rate"
                class="tts-slider"
                type="range"
                min="0.5"
                max="2"
                step="0.1"
                @change="onSpeechSettingChange"
              />
              <strong>{{ ttsPrefs.rate.toFixed(1) }}×</strong>
            </label>

            <label class="tts-field">
              <span>Pitch</span>
              <input
                v-model.number="ttsPrefs.pitch"
                class="tts-slider"
                type="range"
                min="0"
                max="2"
                step="0.1"
                @change="onSpeechSettingChange"
              />
              <strong>{{ ttsPrefs.pitch.toFixed(1) }}</strong>
            </label>

            <label class="tts-field">
              <span>Volume</span>
              <input
                v-model.number="ttsPrefs.volume"
                class="tts-slider"
                type="range"
                min="0"
                max="1"
                step="0.05"
                @change="onSpeechSettingChange"
              />
              <strong>{{ Math.round(ttsPrefs.volume * 100) }}%</strong>
            </label>

            <label class="tts-checkbox">
              <input v-model="ttsPrefs.autoAdvance" type="checkbox" @change="onAutoAdvanceChange" />
              <span>Auto-advance to the next section</span>
            </label>
          </section>
        </div>
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

      <div v-else class="reader-shell">
        <div class="reader-scrim" :class="{ 'is-visible': navOpen }" @click="navOpen = false"></div>

        <aside class="reader-sidebar" :class="{ 'is-open': navOpen }">
          <div class="sidebar-header">
            <h2>Contents</h2>
            <button class="btn-ghost sidebar-close" @click="navOpen = false">Close</button>
          </div>
          <ReaderNavTree
            v-if="manifest"
            :items="manifest.nav"
            :active-section-id="currentSectionId"
            :active-fragment="currentFragment"
            @navigate="navigateToSection"
          />
          <p v-else class="sidebar-empty">No section navigation found.</p>
        </aside>

        <main class="reader-main">
          <div class="reader-frame-wrap">
            <div v-if="sectionLoading" class="frame-loading">
              <div class="spinner"></div>
            </div>
            <iframe
              ref="readerFrame"
              class="reader-frame"
              sandbox="allow-same-origin"
              :srcdoc="currentSection?.html ?? ''"
              title="EPUB section reader"
              @load="onFrameLoad"
            />
          </div>

          <div class="reader-footer">
            <button
              class="btn-ghost nav-btn"
              :disabled="!currentSection?.prev_section_id || sectionLoading"
              @click="goToPreviousSection"
            >
              ← Previous
            </button>

            <div class="progress-wrap">
              <div class="progress-track">
                <div class="progress-fill" :style="{ width: `${currentPercentage}%` }"></div>
              </div>
              <span class="progress-label">
                {{ currentPercentage }}% · {{ currentSectionPositionLabel }}
              </span>
            </div>

            <button
              class="btn-ghost nav-btn"
              :disabled="!currentSection?.next_section_id || sectionLoading"
              @click="goToNextSection"
            >
              Next →
            </button>
          </div>
        </main>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, shallowRef, watch } from 'vue'
import { useRoute } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import ReaderNavTree from '@/components/reader/ReaderNavTree.vue'
import { getContentBlob, getReaderManifest, getReaderSection } from '@/api/books'
import { useBooksStore } from '@/stores/books'
import { useServerConfigStore } from '@/stores/serverConfig'
import type { ReaderManifest, ReaderSection } from '@/types'

const READABLE_BLOCK_SELECTOR = 'p, li, blockquote, dd, dt, figcaption, h1, h2, h3, h4, h5, h6'
const TTS_PREFS_KEY = 'alexandria.reader.tts'
const READER_PREFS_KEY = 'alexandria.reader.display'
const TTS_BLOCK_ATTR = 'data-tts-block-id'
const TTS_SELECTED_ATTR = 'data-tts-selected'
const TTS_ACTIVE_ATTR = 'data-tts-active'
const TTS_SELECTING_CLASS = 'alexandria-tts-selecting'
const THEME_STYLE_ID = 'alexandria-reader-theme'

type TtsMode = 'idle' | 'speaking' | 'paused' | 'selecting'
type ThemePreset = 'light' | 'dark' | 'custom'

interface TtsPreferences {
  voiceURI: string
  rate: number
  pitch: number
  volume: number
  autoAdvance: boolean
}

interface ThemePalette {
  pageBackground: string
  text: string
  muted: string
  accent: string
  link: string
  highlight: string
}

interface ReaderPreferences {
  themePreset: ThemePreset
  customTheme: ThemePalette
}

interface PendingRestoreState {
  sectionProgress: number
  blockIndex: number | null
  fragment: string
}

const themePresets: ThemePreset[] = ['light', 'dark', 'custom']
const themeFields: Array<{ key: keyof ThemePalette; label: string }> = [
  { key: 'pageBackground', label: 'Page' },
  { key: 'text', label: 'Text' },
  { key: 'muted', label: 'Muted' },
  { key: 'accent', label: 'Accent' },
  { key: 'link', label: 'Link' },
  { key: 'highlight', label: 'Highlight' },
]

const presetThemes: Record<Exclude<ThemePreset, 'custom'>, ThemePalette> = {
  light: {
    pageBackground: '#f8f3ea',
    text: '#1c1917',
    muted: '#6d655d',
    accent: '#8a6734',
    link: '#765623',
    highlight: '#d5b989',
  },
  dark: {
    pageBackground: '#111318',
    text: '#efe7db',
    muted: '#b4aaa0',
    accent: '#d7b777',
    link: '#e4c989',
    highlight: '#7f6533',
  },
}

const route = useRoute()
const books = useBooksStore()
const serverConfig = useServerConfigStore()

const readerFrame = ref<HTMLIFrameElement | null>(null)

const loading = ref(true)
const sectionLoading = ref(false)
const epubError = ref<string | null>(null)
const downloadError = ref<string | null>(null)
const pdfBlobUrl = ref<string | null>(null)
const downloading = ref(false)

const manifest = ref<ReaderManifest | null>(null)
const currentSection = ref<ReaderSection | null>(null)
const currentSectionId = ref('')
const currentFragment = ref('')
const currentPercentage = ref(0)
const navOpen = ref(false)
const settingsOpen = ref(false)
const linkCopied = ref(false)

const ttsSupported = ref(false)
const ttsMode = ref<TtsMode>('idle')
const ttsError = ref<string | null>(null)
const availableVoices = shallowRef<SpeechSynthesisVoice[]>([])
const readableBlocks = shallowRef<HTMLElement[]>([])
const fallbackSectionText = ref('')
const selectedStartBlockIndex = ref<number | null>(null)
const currentSpokenBlockIndex = ref<number | null>(null)
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
  set: (value: string) => {
    ttsPrefs.voiceURI = value
  },
})

const hasReadableText = computed(() => readableBlocks.value.length > 0 || fallbackSectionText.value.length > 0)
const canStartTts = computed(() => ttsMode.value !== 'selecting' && hasReadableText.value)
const canChooseParagraph = computed(() => readableBlocks.value.length > 0)
const navButtonLabel = computed(() => (navOpen.value ? 'Hide Contents' : 'Contents'))
const navButtonCompactLabel = computed(() => (navOpen.value ? 'Hide' : 'Contents'))
const ttsButtonLabel = computed(() => {
  if (ttsMode.value === 'speaking') return 'Pause'
  if (ttsMode.value === 'paused') return 'Resume'
  return 'Read Aloud'
})
const ttsButtonCompactLabel = computed(() => {
  if (ttsMode.value === 'speaking') return 'Pause'
  if (ttsMode.value === 'paused') return 'Resume'
  return 'Read'
})
const chooseParagraphLabel = computed(() => (
  ttsMode.value === 'selecting' ? 'Cancel paragraph pick' : 'Choose paragraph'
))
const currentSectionPositionLabel = computed(() => {
  if (!manifest.value || !currentSection.value) return ''
  return `${currentSection.value.section_index + 1}/${manifest.value.sections.length}`
})
const progressChipTitle = computed(() => {
  if (!currentSectionPositionLabel.value) return `${currentPercentage.value}% complete`
  return `${currentPercentage.value}% complete · ${currentSectionPositionLabel.value}`
})

const activeTheme = computed<ThemePalette>(() => {
  if (readerPrefs.themePreset === 'custom') return { ...readerPrefs.customTheme }
  return { ...presetThemes[readerPrefs.themePreset] }
})

const shellStyle = computed(() => ({
  '--reader-bg': mixColor(activeTheme.value.pageBackground, '#000000', 0.2),
  '--reader-surface': activeTheme.value.pageBackground,
  '--reader-text': activeTheme.value.text,
  '--reader-muted': activeTheme.value.muted,
  '--reader-accent': activeTheme.value.accent,
  '--reader-link': activeTheme.value.link,
  '--reader-highlight': activeTheme.value.highlight,
  '--reader-border': hexToRgba(activeTheme.value.muted, 0.2),
  '--reader-nav-active-bg': hexToRgba(activeTheme.value.highlight, 0.28),
  '--reader-nav-active-text': activeTheme.value.text,
}))

let saveTimer: ReturnType<typeof setTimeout> | null = null
let scrollFrame: number | null = null
let speechToken = 0
let autoAdvanceTimer: ReturnType<typeof setTimeout> | null = null
let currentFrameCleanup: (() => void) | null = null
let pendingRestore: PendingRestoreState | null = null
let pendingTtsResume = false
let currentSectionLoadToken = 0

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
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

function loadReaderPreferences(): ReaderPreferences {
  const defaults: ReaderPreferences = {
    themePreset: 'light',
    customTheme: { ...presetThemes.light },
  }

  if (typeof window === 'undefined') return defaults

  try {
    const raw = window.localStorage.getItem(READER_PREFS_KEY)
    if (!raw) return defaults
    const parsed = JSON.parse(raw) as Partial<ReaderPreferences> & { customTheme?: Partial<ThemePalette> }
    const preset = parsed.themePreset === 'dark' || parsed.themePreset === 'custom' ? parsed.themePreset : 'light'
    return {
      themePreset: preset,
      customTheme: {
        pageBackground: parsed.customTheme?.pageBackground || defaults.customTheme.pageBackground,
        text: parsed.customTheme?.text || defaults.customTheme.text,
        muted: parsed.customTheme?.muted || defaults.customTheme.muted,
        accent: parsed.customTheme?.accent || defaults.customTheme.accent,
        link: parsed.customTheme?.link || defaults.customTheme.link,
        highlight: parsed.customTheme?.highlight || defaults.customTheme.highlight,
      },
    }
  } catch {
    return defaults
  }
}

function persistTtsPreferences(): void {
  try {
    window.localStorage.setItem(TTS_PREFS_KEY, JSON.stringify(ttsPrefs))
  } catch {
    // Ignore storage failures and keep in-memory settings.
  }
}

function persistReaderPreferencesAndApplyTheme(): void {
  try {
    window.localStorage.setItem(READER_PREFS_KEY, JSON.stringify(readerPrefs))
  } catch {
    // Ignore storage failures and keep in-memory settings.
  }
  applyFrameTheme()
}

function setThemePreset(preset: ThemePreset): void {
  readerPrefs.themePreset = preset
  persistReaderPreferencesAndApplyTheme()
}

function themePresetLabel(preset: ThemePreset): string {
  switch (preset) {
    case 'dark':
      return 'Dark'
    case 'custom':
      return 'Custom'
    default:
      return 'Light'
  }
}

function normalizeText(value: string): string {
  return value.replace(/\s+/g, ' ').trim()
}

function currentFrameDocument(): Document | null {
  return readerFrame.value?.contentDocument ?? null
}

function currentFrameWindow(): Window | null {
  return readerFrame.value?.contentWindow ?? null
}

function currentScrollElement(): HTMLElement | null {
  const doc = currentFrameDocument()
  return (doc?.scrollingElement as HTMLElement | null) ?? doc?.documentElement ?? null
}

function currentFrameBody(): HTMLBodyElement | null {
  return (currentFrameDocument()?.body as HTMLBodyElement | null) ?? null
}

function formatVoiceLabel(voice: SpeechSynthesisVoice): string {
  const language = voice.lang || 'Unknown language'
  return voice.default ? `${voice.name} (${language}, default)` : `${voice.name} (${language})`
}

function normalizeLanguageTag(value: string | undefined): string {
  return (value ?? '').toLowerCase()
}

function baseLanguageTag(value: string | undefined): string {
  return normalizeLanguageTag(value).split('-')[0]
}

function compareVoices(left: SpeechSynthesisVoice, right: SpeechSynthesisVoice, preferredLang: string): number {
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

async function loadReaderState(bookId: string): Promise<void> {
  loading.value = true
  epubError.value = null
  currentSection.value = null
  manifest.value = null
  currentSectionId.value = ''
  currentFragment.value = ''
  navOpen.value = false
  settingsOpen.value = false
  cleanupCurrentFrame()
  internalStopTts({ preserveSelected: false })

  try {
    await Promise.all([books.fetchBook(bookId), books.fetchProgress(bookId)])
  } catch {
    epubError.value = 'Failed to load book metadata.'
    loading.value = false
    return
  }

  if (pdfBlobUrl.value) {
    URL.revokeObjectURL(pdfBlobUrl.value)
    pdfBlobUrl.value = null
  }

  if (book.value?.file_type === 'pdf') {
    loading.value = false
    try {
      const blob = await getContentBlob(bookId)
      pdfBlobUrl.value = URL.createObjectURL(blob)
    } catch {
      // Non-fatal.
    }
    return
  }

  if (book.value?.file_type !== 'epub') {
    loading.value = false
    return
  }

  try {
    const response = await getReaderManifest(bookId)
    manifest.value = response.manifest
  } catch {
    epubError.value = 'Failed to load EPUB structure.'
    loading.value = false
    return
  }

  const initialSectionId = resolveInitialSectionId()
  pendingRestore = {
    sectionProgress: shouldRestoreSavedProgress(initialSectionId) ? progress.value?.section_progress ?? 0 : 0,
    blockIndex: shouldRestoreSavedProgress(initialSectionId) ? progress.value?.block_index ?? null : null,
    fragment: '',
  }

  await loadSection(initialSectionId)
  loading.value = false
}

function shouldRestoreSavedProgress(sectionId: string): boolean {
  return Boolean(progress.value?.section_id) && progress.value?.section_id === sectionId
}

function resolveInitialSectionId(): string {
  const sections = manifest.value?.sections ?? []
  if (sections.length === 0) return ''
  if (progress.value?.section_id && sections.some((section) => section.id === progress.value?.section_id)) {
    return progress.value.section_id
  }
  return manifest.value?.first_section_id || sections[0].id
}

async function loadSection(
  sectionId: string,
  options: { fragment?: string; resumeTts?: boolean; blockIndex?: number | null; sectionProgress?: number } = {},
): Promise<void> {
  if (!book.value || book.value.file_type !== 'epub' || !sectionId) return

  if (currentSectionId.value === sectionId && currentSection.value) {
    pendingRestore = {
      sectionProgress: options.sectionProgress ?? 0,
      blockIndex: options.blockIndex ?? null,
      fragment: options.fragment ?? '',
    }
    if (pendingRestore.fragment) {
      restoreFramePosition()
    }
    return
  }

  currentSectionLoadToken += 1
  const token = currentSectionLoadToken
  sectionLoading.value = true
  if (!options.resumeTts) {
    internalStopTts({ preserveSelected: false })
  } else {
    cancelSpeechOutput()
    pendingTtsResume = true
  }

  try {
    const response = await getReaderSection(book.value.id, sectionId)
    if (token !== currentSectionLoadToken) return

    currentSection.value = response.section
    // In Tauri, srcdoc iframes resolve relative URLs against tauri://localhost, not the
    // configured server. Rewrite /api/ paths to absolute URLs so images load correctly.
    const serverBase = serverConfig.serverUrl
    if (serverBase && currentSection.value) {
      const html = currentSection.value.html
      currentSection.value = {
        ...currentSection.value,
        html: html
          .replace(/="\/api\//g, `="${serverBase}/api/`)
          .replace(/='\/api\//g, `='${serverBase}/api/`)
          .replace(/url\(\/api\//g, `url(${serverBase}/api/`),
      }
    }
    currentSectionId.value = response.section.id
    currentFragment.value = ''
    pendingRestore = {
      sectionProgress: options.sectionProgress ?? 0,
      blockIndex: options.blockIndex ?? null,
      fragment: options.fragment ?? '',
    }
    navOpen.value = false
  } catch {
    if (token !== currentSectionLoadToken) return
    epubError.value = 'Failed to load EPUB section.'
  } finally {
    if (token === currentSectionLoadToken) {
      sectionLoading.value = false
    }
  }
}

function navigateToSection(sectionId: string, fragment?: string) {
  void loadSection(sectionId, { fragment })
}

function toggleNav() {
  const next = !navOpen.value
  if (next) settingsOpen.value = false
  navOpen.value = next
}

function toggleSettings() {
  const next = !settingsOpen.value
  if (next) navOpen.value = false
  settingsOpen.value = next
}

async function copyBookLink() {
  await navigator.clipboard.writeText(window.location.href)
  linkCopied.value = true
  setTimeout(() => { linkCopied.value = false }, 2000)
}

function toggleParagraphPickerFromSettings() {
  const enteringSelection = ttsMode.value !== 'selecting'
  if (enteringSelection) settingsOpen.value = false
  toggleParagraphPicker()
}

function goToPreviousSection() {
  if (!currentSection.value?.prev_section_id) return
  void loadSection(currentSection.value.prev_section_id)
}

function goToNextSection() {
  if (!currentSection.value?.next_section_id) return
  void loadSection(currentSection.value.next_section_id)
}

function onFrameLoad() {
  cleanupCurrentFrame()
  applyFrameTheme()
  setupFrameInteractions()

  window.requestAnimationFrame(() => {
    collectReadableBlocks()
    restoreFramePosition()
    syncProgressFromFrame()

    if (pendingTtsResume) {
      pendingTtsResume = false
      if (readableBlocks.value.length > 0) {
        speakBlockAtIndex(activeBlockIndex())
      } else {
        speakFallbackText()
      }
    }
  })
}

function cleanupCurrentFrame() {
  if (currentFrameCleanup) {
    currentFrameCleanup()
    currentFrameCleanup = null
  }
  readableBlocks.value = []
  fallbackSectionText.value = ''
}

function setupFrameInteractions() {
  const doc = currentFrameDocument()
  const win = currentFrameWindow()
  if (!doc || !win || !doc.body) return

  const onScroll = () => {
    if (scrollFrame !== null) window.cancelAnimationFrame(scrollFrame)
    scrollFrame = window.requestAnimationFrame(() => {
      scrollFrame = null
      syncProgressFromFrame()
    })
  }

  const onClick = (event: MouseEvent) => {
    const target = event.target as HTMLElement | null
    const anchor = target?.closest<HTMLAnchorElement>('a')
    if (anchor?.dataset.readerSectionId) {
      event.preventDefault()
      const sectionId = anchor.dataset.readerSectionId
      if (!sectionId) return
      const fragment = anchor.dataset.readerFragment || undefined
      void loadSection(sectionId, { fragment })
      return
    }

    if (ttsMode.value !== 'selecting') return
    const blockElement = target?.closest<HTMLElement>(`[${TTS_BLOCK_ATTR}]`)
    if (!blockElement) return
    const index = Number(blockElement.getAttribute(TTS_BLOCK_ATTR))
    if (!Number.isFinite(index)) return
    event.preventDefault()
    event.stopPropagation()
    selectedStartBlockIndex.value = index
    currentSpokenBlockIndex.value = null
    ttsMode.value = 'idle'
    syncBlockMarkers()
  }

  doc.addEventListener('click', onClick, true)
  win.addEventListener('scroll', onScroll, { passive: true })

  currentFrameCleanup = () => {
    doc.removeEventListener('click', onClick, true)
    win.removeEventListener('scroll', onScroll)
  }
}

function restoreFramePosition() {
  const restore = pendingRestore
  pendingRestore = null
  if (!restore) return

  if (restore.fragment) {
    scrollToFragment(restore.fragment)
    return
  }

  if (restore.blockIndex !== null) {
    scrollToBlockIndex(restore.blockIndex)
    return
  }

  if (restore.sectionProgress > 0) {
    const scrollElement = currentScrollElement()
    if (!scrollElement) return
    const maxScroll = Math.max(scrollElement.scrollHeight - scrollElement.clientHeight, 0)
    scrollElement.scrollTop = maxScroll * restore.sectionProgress
    return
  }

  const scrollElement = currentScrollElement()
  if (scrollElement) scrollElement.scrollTop = 0
}

function collectReadableBlocks() {
  const doc = currentFrameDocument()
  if (!doc?.body) {
    readableBlocks.value = []
    fallbackSectionText.value = ''
    return
  }

  const candidates = Array.from(doc.querySelectorAll<HTMLElement>(READABLE_BLOCK_SELECTOR))
  const nextBlocks = candidates.filter((element) => {
    if (!element.isConnected) return false
    const text = normalizeText(element.innerText || element.textContent || '')
    if (!text) return false
    const nested = element.querySelector<HTMLElement>(READABLE_BLOCK_SELECTOR)
    if (nested) {
      const nestedText = normalizeText(nested.innerText || nested.textContent || '')
      if (nestedText) return false
    }
    return true
  })

  nextBlocks.forEach((element, index) => {
    element.setAttribute(TTS_BLOCK_ATTR, String(index))
  })

  readableBlocks.value = nextBlocks
  fallbackSectionText.value = normalizeText(doc.body.innerText || '')

  if (selectedStartBlockIndex.value !== null && selectedStartBlockIndex.value >= nextBlocks.length) {
    selectedStartBlockIndex.value = null
  }
  if (currentSpokenBlockIndex.value !== null && currentSpokenBlockIndex.value >= nextBlocks.length) {
    currentSpokenBlockIndex.value = null
  }

  syncBlockMarkers()
}

function syncProgressFromFrame() {
  updateCurrentFragment()
  updateCurrentPercentage()
  collectReadableBlocks()
  scheduleSaveProgress()
}

function updateCurrentFragment() {
  const doc = currentFrameDocument()
  if (!doc) {
    currentFragment.value = ''
    return
  }

  const tracked = Array.from(doc.querySelectorAll<HTMLElement>('[id], a[name]'))
  let nextFragment = ''
  for (const element of tracked) {
    const rect = element.getBoundingClientRect()
    if (rect.top > 100) break
    nextFragment = element.id || element.getAttribute('name') || ''
  }
  currentFragment.value = nextFragment
}

function updateCurrentPercentage() {
  if (!manifest.value || !currentSection.value) {
    currentPercentage.value = 0
    return
  }

  const scrollElement = currentScrollElement()
  const maxScroll = scrollElement ? Math.max(scrollElement.scrollHeight - scrollElement.clientHeight, 0) : 0
  const sectionProgress = scrollElement ? (maxScroll > 0 ? scrollElement.scrollTop / maxScroll : 1) : 0
  const totalSections = Math.max(manifest.value.sections.length, 1)
  const percentage = clamp((currentSection.value.section_index + clamp(sectionProgress, 0, 1)) / totalSections, 0, 1)
  currentPercentage.value = Math.round(percentage * 100)
}

function scheduleSaveProgress() {
  if (!book.value || book.value.file_type !== 'epub' || !currentSection.value || !manifest.value) return
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    void flushProgress()
  }, 1500)
}

async function flushProgress() {
  if (saveTimer) {
    clearTimeout(saveTimer)
    saveTimer = null
  }

  if (!book.value || book.value.file_type !== 'epub' || !currentSection.value || !manifest.value) return

  const scrollElement = currentScrollElement()
  const maxScroll = scrollElement ? Math.max(scrollElement.scrollHeight - scrollElement.clientHeight, 0) : 0
  const sectionProgress = scrollElement ? clamp(maxScroll > 0 ? scrollElement.scrollTop / maxScroll : 1, 0, 1) : 0
  const percentage = clamp((currentSection.value.section_index + sectionProgress) / Math.max(manifest.value.sections.length, 1), 0, 1)

  const payload = {
    section_id: currentSection.value.id,
    section_progress: sectionProgress,
    block_index: nearestVisibleBlockIndex(),
    percentage,
  }

  await books.saveProgress(book.value.id, payload)
}

function nearestVisibleBlockIndex(): number | null {
  if (readableBlocks.value.length === 0) return null

  let lastAbove = 0
  for (let index = 0; index < readableBlocks.value.length; index += 1) {
    const element = readableBlocks.value[index]
    const rect = element.getBoundingClientRect()
    if (rect.bottom >= 96 && rect.top <= window.innerHeight) return index
    if (rect.top <= 96) lastAbove = index
  }
  return lastAbove
}

function scrollToBlockIndex(index: number) {
  const element = readableBlocks.value[index]
  if (!element) return
  element.scrollIntoView({ block: 'start', behavior: 'auto' })
}

function scrollToFragment(fragment: string) {
  const doc = currentFrameDocument()
  if (!doc) return

  const byID = doc.getElementById(fragment)
  if (byID) {
    byID.scrollIntoView({ block: 'start', behavior: 'auto' })
    return
  }

  const byName = Array.from(doc.querySelectorAll<HTMLElement>('a[name]'))
    .find((element) => element.getAttribute('name') === fragment)
  if (byName) {
    byName.scrollIntoView({ block: 'start', behavior: 'auto' })
  }
}

function applyFrameTheme() {
  const doc = currentFrameDocument()
  if (!doc) return

  let head = doc.head
  if (!head) {
    head = doc.createElement('head')
    doc.documentElement?.prepend(head)
  }

  let style = head.querySelector<HTMLStyleElement>(`#${THEME_STYLE_ID}`)
  if (!style) {
    style = doc.createElement('style')
    style.id = THEME_STYLE_ID
    head.appendChild(style)
  }

  style.textContent = frameThemeStyles(activeTheme.value)
  if (doc.body) {
    doc.body.classList.toggle(TTS_SELECTING_CLASS, ttsMode.value === 'selecting')
  }
  syncBlockMarkers()
}

function frameThemeStyles(theme: ThemePalette): string {
  return `
    html, body {
      margin: 0;
      min-height: 100%;
      background: ${theme.pageBackground} !important;
      color: ${theme.text} !important;
    }

    body {
      max-width: min(860px, 100%);
      margin: 0 auto;
      padding: 0 1.2rem 2.8rem;
      font-family: "Iowan Old Style", Georgia, serif;
      font-size: clamp(1rem, 0.98rem + 0.2vw, 1.06rem);
      line-height: 1.72;
      word-break: break-word;
    }

    a {
      color: ${theme.link} !important;
    }

    h1, h2, h3, h4, h5, h6, strong, b {
      color: ${theme.text} !important;
    }

    img, svg, video, canvas, object, embed, iframe, audio {
      max-width: 100% !important;
      height: auto;
    }

    table {
      display: block;
      max-width: 100%;
      overflow-x: auto;
      border-collapse: collapse;
    }

    pre {
      white-space: pre-wrap;
    }

    [${TTS_BLOCK_ATTR}] {
      border-radius: 0.28rem;
      transition: background-color 0.16s ease, box-shadow 0.16s ease;
    }

    body.${TTS_SELECTING_CLASS} [${TTS_BLOCK_ATTR}] {
      cursor: pointer;
    }

    body.${TTS_SELECTING_CLASS} [${TTS_BLOCK_ATTR}]:hover {
      background: ${hexToRgba(theme.highlight, 0.16)};
      box-shadow: inset 0 0 0 1px ${hexToRgba(theme.highlight, 0.55)};
    }

    [${TTS_SELECTED_ATTR}] {
      background: ${hexToRgba(theme.highlight, 0.14)};
      box-shadow: inset 0 0 0 1px ${hexToRgba(theme.highlight, 0.48)};
    }

    [${TTS_ACTIVE_ATTR}] {
      background: ${hexToRgba(theme.highlight, 0.28)};
      box-shadow: inset 0 0 0 1px ${hexToRgba(theme.highlight, 0.74)};
    }
  `
}

function syncBlockMarkers() {
  const body = currentFrameBody()
  if (body) {
    body.classList.toggle(TTS_SELECTING_CLASS, ttsMode.value === 'selecting')
  }

  readableBlocks.value.forEach((element, index) => {
    if (selectedStartBlockIndex.value === index) {
      element.setAttribute(TTS_SELECTED_ATTR, 'true')
    } else {
      element.removeAttribute(TTS_SELECTED_ATTR)
    }

    if (currentSpokenBlockIndex.value === index && (ttsMode.value === 'speaking' || ttsMode.value === 'paused')) {
      element.setAttribute(TTS_ACTIVE_ATTR, 'true')
    } else {
      element.removeAttribute(TTS_ACTIVE_ATTR)
    }
  })
}

function activeBlockIndex(): number {
  if (currentSpokenBlockIndex.value !== null) return currentSpokenBlockIndex.value
  if (selectedStartBlockIndex.value !== null) return selectedStartBlockIndex.value
  return nearestVisibleBlockIndex() ?? 0
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

function cancelSpeechOutput() {
  speechToken += 1
  if (autoAdvanceTimer) {
    clearTimeout(autoAdvanceTimer)
    autoAdvanceTimer = null
  }
  if (ttsSupported.value) window.speechSynthesis.cancel()
}

function internalStopTts(options: { preserveSelected: boolean }) {
  cancelSpeechOutput()
  ttsMode.value = 'idle'
  ttsError.value = null
  currentSpokenBlockIndex.value = null
  pendingTtsResume = false
  if (!options.preserveSelected) selectedStartBlockIndex.value = null
  syncBlockMarkers()
}

function stopTts() {
  internalStopTts({ preserveSelected: true })
}

function readableBlockText(index: number): string {
  const element = readableBlocks.value[index]
  return element ? normalizeText(element.innerText || element.textContent || '') : ''
}

function speakFallbackText() {
  const text = fallbackSectionText.value
  if (!ttsSupported.value || !text) return

  const token = speechToken + 1
  cancelSpeechOutput()
  speechToken = token

  const utterance = new SpeechSynthesisUtterance(text)
  applyUtteranceSettings(utterance)

  ttsMode.value = 'speaking'
  currentSpokenBlockIndex.value = null
  ttsError.value = null
  syncBlockMarkers()

  utterance.onend = () => {
    if (token !== speechToken) return
    if (ttsPrefs.autoAdvance) {
      void queueAutoAdvance()
    } else {
      ttsMode.value = 'idle'
      syncBlockMarkers()
    }
  }
  utterance.onerror = (event) => {
    if (token !== speechToken) return
    handleTtsError(event.error)
  }

  window.speechSynthesis.speak(utterance)
}

function speakBlockAtIndex(index: number) {
  const text = readableBlockText(index)
  if (!text || !ttsSupported.value) {
    internalStopTts({ preserveSelected: true })
    return
  }

  const token = speechToken + 1
  cancelSpeechOutput()
  speechToken = token

  const utterance = new SpeechSynthesisUtterance(text)
  applyUtteranceSettings(utterance)

  currentSpokenBlockIndex.value = index
  ttsMode.value = 'speaking'
  ttsError.value = null
  scrollToBlockIndex(index)
  syncBlockMarkers()

  utterance.onend = () => {
    if (token !== speechToken) return
    const nextIndex = index + 1
    if (nextIndex < readableBlocks.value.length) {
      speakBlockAtIndex(nextIndex)
      return
    }
    currentSpokenBlockIndex.value = null
    if (ttsPrefs.autoAdvance) {
      void queueAutoAdvance()
    } else {
      ttsMode.value = 'idle'
      syncBlockMarkers()
    }
  }
  utterance.onerror = (event) => {
    if (token !== speechToken) return
    handleTtsError(event.error)
  }

  window.speechSynthesis.speak(utterance)
}

function handleTtsError(error: string) {
  internalStopTts({ preserveSelected: true })
  if (error !== 'interrupted' && error !== 'canceled') {
    ttsError.value = `TTS error: ${error}`
  }
}

async function queueAutoAdvance() {
  if (!currentSection.value?.next_section_id) {
    internalStopTts({ preserveSelected: true })
    return
  }

  autoAdvanceTimer = setTimeout(() => {
    autoAdvanceTimer = null
    void loadSection(currentSection.value?.next_section_id ?? '', { resumeTts: true, blockIndex: 0 })
  }, 180)
}

async function startTts() {
  if (!ttsSupported.value || !hasReadableText.value || ttsMode.value === 'selecting') return
  if (readableBlocks.value.length > 0) {
    speakBlockAtIndex(activeBlockIndex())
    return
  }
  speakFallbackText()
}

function pauseTts() {
  if (ttsMode.value !== 'speaking') return
  cancelSpeechOutput()
  ttsMode.value = 'paused'
  syncBlockMarkers()
}

async function resumeTts() {
  if (ttsMode.value !== 'paused') return
  if (readableBlocks.value.length > 0) {
    speakBlockAtIndex(activeBlockIndex())
    return
  }
  speakFallbackText()
}

async function toggleTts() {
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

function toggleParagraphPicker() {
  if (!canChooseParagraph.value) return
  if (ttsMode.value === 'selecting') {
    ttsMode.value = 'idle'
    syncBlockMarkers()
    return
  }

  internalStopTts({ preserveSelected: true })
  ttsMode.value = 'selecting'
  syncBlockMarkers()
}

function restartCurrentPlayback() {
  if (ttsMode.value !== 'speaking') return
  if (readableBlocks.value.length > 0) {
    speakBlockAtIndex(activeBlockIndex())
  } else {
    speakFallbackText()
  }
}

function onVoiceChange() {
  persistTtsPreferences()
  restartCurrentPlayback()
}

function onSpeechSettingChange() {
  persistTtsPreferences()
  restartCurrentPlayback()
}

function onAutoAdvanceChange() {
  persistTtsPreferences()
}

function loadVoices() {
  if (!ttsSupported.value) return
  availableVoices.value = window.speechSynthesis.getVoices()
}

function hexToRgba(hex: string, alpha: number): string {
  const normalized = hex.replace('#', '')
  const expanded = normalized.length === 3
    ? normalized.split('').map((part) => part + part).join('')
    : normalized
  const value = Number.parseInt(expanded, 16)
  if (!Number.isFinite(value)) return `rgba(0, 0, 0, ${alpha})`
  const r = (value >> 16) & 255
  const g = (value >> 8) & 255
  const b = value & 255
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

function mixColor(base: string, overlay: string, overlayAlpha: number): string {
  const baseRgb = parseHex(base)
  const overlayRgb = parseHex(overlay)
  return `rgb(${Math.round(baseRgb.r * (1 - overlayAlpha) + overlayRgb.r * overlayAlpha)}, ${Math.round(baseRgb.g * (1 - overlayAlpha) + overlayRgb.g * overlayAlpha)}, ${Math.round(baseRgb.b * (1 - overlayAlpha) + overlayRgb.b * overlayAlpha)})`
}

function parseHex(hex: string): { r: number; g: number; b: number } {
  const normalized = hex.replace('#', '')
  const expanded = normalized.length === 3
    ? normalized.split('').map((part) => part + part).join('')
    : normalized
  const value = Number.parseInt(expanded, 16)
  if (!Number.isFinite(value)) {
    return { r: 0, g: 0, b: 0 }
  }
  return {
    r: (value >> 16) & 255,
    g: (value >> 8) & 255,
    b: value & 255,
  }
}

watch(activeTheme, () => {
  applyFrameTheme()
})

onMounted(async () => {
  ttsSupported.value = 'speechSynthesis' in window && 'SpeechSynthesisUtterance' in window
  if (ttsSupported.value) {
    loadVoices()
    window.speechSynthesis.addEventListener('voiceschanged', loadVoices)
  }
  await loadReaderState(route.params.id as string)
})

onUnmounted(() => {
  cleanupCurrentFrame()
  if (ttsSupported.value) {
    window.speechSynthesis.removeEventListener('voiceschanged', loadVoices)
    cancelSpeechOutput()
  }
  if (scrollFrame !== null) {
    window.cancelAnimationFrame(scrollFrame)
    scrollFrame = null
  }
  void flushProgress()
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
  display: flex;
  flex-direction: column;
  background: var(--reader-bg);
  color: var(--reader-text);
}

.reader-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.reader-header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.85rem 1rem;
  padding-top: env(safe-area-inset-top, 0px);
  padding-left: max(1rem, env(safe-area-inset-left, 0px));
  padding-right: max(1rem, env(safe-area-inset-right, 0px));
  padding-bottom: 0.65rem;
  border-bottom: 1px solid var(--reader-border);
  background: color-mix(in srgb, var(--reader-surface) 94%, transparent);
  backdrop-filter: blur(10px);
}

.reader-heading {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
}

.back-btn {
  flex-shrink: 0;
  border-radius: 999px;
  padding: 0.5rem 0.9rem;
  background: color-mix(in srgb, var(--reader-surface) 82%, transparent);
  border-color: color-mix(in srgb, var(--reader-muted) 20%, transparent);
}

.book-info {
  min-width: 0;
  display: grid;
  gap: 0.18rem;
}

.book-title {
  font-weight: 700;
  font-size: 1.02rem;
  line-height: 1.2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.book-meta {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  min-width: 0;
  color: var(--reader-muted);
  font-size: 0.82rem;
}

.book-meta-divider {
  flex-shrink: 0;
  color: color-mix(in srgb, var(--reader-muted) 80%, transparent);
}

.book-author,
.book-section {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.reader-toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.5rem;
  flex-wrap: wrap;
  min-width: 0;
}

.toolbar-btn,
.settings-close,
.nav-btn {
  border-radius: 999px;
  padding: 0.5rem 0.85rem;
  background: color-mix(in srgb, var(--reader-surface) 82%, transparent);
  border-color: color-mix(in srgb, var(--reader-muted) 20%, transparent);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  line-height: 1.15;
}

.label-mobile {
  display: none;
}

.progress-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.38rem 0.72rem;
  border-radius: 999px;
  border: 1px solid color-mix(in srgb, var(--reader-muted) 20%, transparent);
  background: color-mix(in srgb, var(--reader-surface) 78%, transparent);
  color: var(--reader-muted);
  font-size: 0.8rem;
  white-space: nowrap;
}

.progress-chip-value {
  color: var(--reader-text);
  font-weight: 700;
}

.progress-chip-detail {
  color: inherit;
}

.toolbar-btn.is-active {
  border-color: color-mix(in srgb, var(--reader-accent) 50%, transparent);
  background: var(--reader-nav-active-bg);
}

.reader-settings-shell {
  position: relative;
}

.reader-settings-backdrop {
  display: none;
}

.reader-settings {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
  padding: 0.85rem 1rem 1rem;
  border-bottom: 1px solid var(--reader-border);
  background: color-mix(in srgb, var(--reader-surface) 97%, transparent);
}

.reader-settings-header {
  grid-column: 1 / -1;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.settings-kicker {
  display: inline-block;
  font-size: 0.72rem;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--reader-muted);
}

.reader-settings-header h2 {
  margin: 0.2rem 0 0;
  font-size: 1rem;
}

.reader-mobile-actions {
  display: none;
}

.settings-action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
}

.settings-group {
  background: color-mix(in srgb, var(--reader-surface) 96%, transparent);
  border: 1px solid var(--reader-border);
  border-radius: 1rem;
  padding: 1rem;
  display: grid;
  gap: 0.85rem;
}

.settings-group h3 {
  margin: 0;
  font-size: 0.95rem;
}

.theme-preset-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
}

.theme-chip {
  border: 1px solid var(--reader-border);
  background: transparent;
  color: inherit;
  border-radius: 999px;
  padding: 0.42rem 0.78rem;
  cursor: pointer;
}

.theme-chip.is-active {
  background: var(--reader-nav-active-bg);
  border-color: color-mix(in srgb, var(--reader-accent) 50%, transparent);
}

.theme-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.theme-field,
.tts-field {
  display: grid;
  gap: 0.35rem;
}

.theme-field input[type='color'] {
  width: 100%;
  height: 2.6rem;
  border: 0;
  background: transparent;
}

.tts-select,
.tts-slider {
  width: 100%;
}

.tts-checkbox {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.tts-feedback {
  margin: 0;
  padding: 0.75rem 1.2rem;
  color: #ffb3b3;
  background: rgba(120, 24, 24, 0.2);
  border-bottom: 1px solid rgba(120, 24, 24, 0.35);
}

.reader-shell {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  position: relative;
}

.reader-sidebar {
  border-right: 1px solid var(--reader-border);
  background: color-mix(in srgb, var(--reader-surface) 94%, transparent);
  padding: 1rem;
  overflow-y: auto;
  min-height: 0;
  position: relative;
  z-index: 2;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.sidebar-header h2 {
  margin: 0;
  font-size: 0.98rem;
}

.sidebar-close {
  display: none;
}

.sidebar-empty {
  margin: 0;
  color: var(--reader-muted);
}

.reader-main {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
}

.reader-frame-wrap {
  position: relative;
  min-height: 0;
  background: var(--reader-surface);
}

.reader-frame {
  width: 100%;
  height: 100%;
  border: 0;
  background: var(--reader-surface);
}

.frame-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.14);
  z-index: 1;
}

.reader-footer {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-top: 1px solid var(--reader-border);
  background: color-mix(in srgb, var(--reader-surface) 96%, transparent);
}

.nav-btn {
  flex-shrink: 0;
}

.progress-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.8rem;
}

.progress-track {
  flex: 1;
  height: 0.32rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--reader-accent), var(--reader-link));
}

.progress-label {
  color: var(--reader-muted);
  font-size: 0.8rem;
  white-space: nowrap;
}

.reader-scrim {
  display: none;
}

.state-overlay,
.pdf-viewer {
  flex: 1;
  min-height: 0;
}

.state-overlay {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 1rem;
}

.spinner {
  width: 2rem;
  height: 2rem;
  border-radius: 999px;
  border: 2px solid rgba(255, 255, 255, 0.14);
  border-top-color: var(--reader-accent);
  animation: spin 0.9s linear infinite;
}

.pdf-viewer {
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
}

.pdf-iframe {
  width: 100%;
  height: 100%;
  border: 0;
}

.pdf-toolbar {
  padding: 0.8rem 1.2rem;
  border-top: 1px solid var(--reader-border);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 1rem;
}

.download-error {
  color: #ffb3b3;
}

.fallback-card {
  max-width: 420px;
  padding: 1.5rem;
  text-align: center;
  display: grid;
  gap: 0.8rem;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 980px) {
  .reader-header {
    grid-template-columns: minmax(0, 1fr);
  }

  .reader-toolbar {
    justify-content: flex-start;
  }

  .reader-shell {
    grid-template-columns: minmax(0, 1fr);
  }

  .reader-sidebar {
    position: absolute;
    top: 0;
    bottom: 0;
    left: 0;
    width: min(84vw, 320px);
    transform: translateX(-100%);
    transition: transform 0.2s ease;
    box-shadow: 0 24px 60px rgba(0, 0, 0, 0.35);
  }

  .reader-sidebar.is-open {
    transform: translateX(0);
  }

  .reader-scrim {
    display: block;
    position: absolute;
    inset: 0;
    background: rgba(0, 0, 0, 0.35);
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.2s ease;
    z-index: 1;
  }

  .reader-scrim.is-visible {
    opacity: 1;
    pointer-events: auto;
  }

  .sidebar-close {
    display: inline-flex;
  }
}

@media (max-width: 720px) {
  .reader-header {
    padding-inline: 0.85rem;
    padding-bottom: 0.6rem;
    gap: 0.7rem;
  }

  .reader-heading {
    gap: 0.65rem;
  }

  .back-btn {
    padding: 0.46rem 0.78rem;
    font-size: 0.82rem;
  }

  .book-title {
    font-size: 0.96rem;
  }

  .book-meta {
    display: none;
  }

  .reader-toolbar {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    align-items: stretch;
    width: 100%;
    gap: 0.45rem;
  }

  .toolbar-btn {
    min-width: 0;
    padding: 0.5rem 0.55rem;
    font-size: 0.8rem;
  }

  .toolbar-btn-secondary {
    display: none;
  }

  .label-desktop {
    display: none;
  }

  .label-mobile {
    display: inline;
  }

  .progress-chip {
    justify-content: center;
    min-width: 0;
    padding: 0.36rem 0.55rem;
  }

  .reader-settings-shell {
    position: fixed;
    inset: 0;
    display: flex;
    align-items: flex-end;
    z-index: 40;
  }

  .reader-settings-backdrop {
    display: block;
    position: absolute;
    inset: 0;
    border: 0;
    background: rgba(8, 10, 14, 0.45);
  }

  .reader-settings {
    position: relative;
    z-index: 1;
    width: 100%;
    max-height: min(78vh, 680px);
    overflow-y: auto;
    padding: 0.85rem 0.9rem 1rem;
    grid-template-columns: 1fr;
    border: 1px solid var(--reader-border);
    border-bottom: 0;
    border-radius: 1.2rem 1.2rem 0 0;
    box-shadow: 0 -24px 48px rgba(0, 0, 0, 0.28);
  }

  .reader-mobile-actions {
    display: grid;
  }

  .theme-grid {
    grid-template-columns: 1fr;
  }

  .reader-footer {
    padding: 0.72rem 0.85rem;
    gap: 0.55rem;
    flex-wrap: wrap;
  }

  .nav-btn {
    padding: 0.5rem 0.85rem;
    font-size: 0.82rem;
  }

  .progress-wrap {
    min-width: 0;
    order: 3;
    width: 100%;
    gap: 0.65rem;
  }
}

@media (max-width: 480px) {
  .reader-header {
    padding-inline: 0.72rem;
    padding-bottom: 0.58rem;
  }

  .reader-heading {
    gap: 0.55rem;
  }

  .back-btn {
    padding: 0.44rem 0.68rem;
  }

  .book-title {
    font-size: 0.93rem;
  }

  .toolbar-btn {
    padding: 0.46rem 0.4rem;
    font-size: 0.76rem;
  }

  .progress-chip {
    padding: 0.34rem 0.42rem;
    font-size: 0.76rem;
  }

  .progress-chip-detail {
    display: none;
  }

  .reader-settings {
    padding: 0.8rem 0.75rem 0.95rem;
  }

  .reader-footer {
    padding: 0.65rem 0.72rem;
  }

  .progress-label {
    font-size: 0.76rem;
  }
}
</style>
