<template>
  <div class="book-card card" @click="$emit('open', book)">
    <div class="cover-wrap">
      <img
        v-if="book.cover_path"
        :src="coverSrc"
        :alt="book.title"
        class="cover"
        loading="lazy"
      />
      <div v-else class="cover-placeholder">
        <span>{{ initials }}</span>
      </div>
    </div>
    <div class="info">
      <p class="title">{{ book.title }}</p>
      <p class="author">{{ book.author || 'Unknown author' }}</p>
      <div v-if="progress" class="progress-bar-wrap">
        <div class="progress-bar" :style="{ width: `${Math.round(progress.percentage * 100)}%` }" />
        <span class="progress-label">{{ Math.round(progress.percentage * 100) }}%</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { coverUrl } from '@/api/books'
import type { Book, ReadingProgress } from '@/types'

const props = defineProps<{
  book: Book
  progress?: ReadingProgress | null
}>()

defineEmits<{ (e: 'open', book: Book): void }>()

const coverSrc = computed(() => coverUrl(props.book.id))

const initials = computed(() => {
  const words = props.book.title.split(' ').slice(0, 2)
  return words.map((w) => w[0]?.toUpperCase() ?? '').join('')
})
</script>

<style scoped>
.book-card {
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.book-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  border-color: var(--accent-dim);
}

.cover-wrap {
  width: 100%;
  aspect-ratio: 2 / 3;
  overflow: hidden;
  background: var(--surface-hover);
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

.info {
  padding: 0.75rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.title {
  font-size: 0.9rem;
  font-weight: 600;
  line-height: 1.3;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.author {
  font-size: 0.8rem;
  color: var(--text-muted);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.progress-bar-wrap {
  margin-top: auto;
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
.progress-bar {
  height: 3px;
  background: var(--accent);
  border-radius: 2px;
  flex: 1;
  max-width: calc(100% - 32px);
}
.progress-label {
  font-size: 0.7rem;
  color: var(--text-muted);
}
</style>
