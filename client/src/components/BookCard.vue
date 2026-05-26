<template>
  <div class="book-card card" :class="{ 'is-selected': selected }" @click="$emit('show-detail', book)">
    <div class="cover-wrap">
      <img
        v-if="coverSrc"
        :src="coverSrc"
        :alt="book.title"
        class="cover"
        loading="lazy"
      />
      <div v-else class="cover-placeholder">
        <span>{{ initials }}</span>
      </div>
      <div v-if="selectable" class="select-overlay" @click.stop>
        <input
          type="checkbox"
          class="select-checkbox"
          :checked="selected"
          @change="$emit('toggle-select', book.id)"
        />
      </div>
    </div>
    <div class="info">
      <p class="title">{{ book.title }}</p>
      <p class="author">{{ book.author || 'Unknown author' }}</p>
      <div v-if="progress" class="progress-bar-wrap">
        <div class="progress-bar" :style="{ width: `${Math.round(progress.percentage * 100)}%` }" />
        <span class="progress-label">{{ Math.round(progress.percentage * 100) }}%</span>
      </div>
      <RatingStars v-if="progress?.rating" :rating="progress.rating" class="card-rating" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useBookCover } from '@/composables/useBookCover'
import type { Book, ReadingProgress } from '@/types'
import RatingStars from './RatingStars.vue'

const props = defineProps<{
  book: Book
  progress?: ReadingProgress | null
  selectable?: boolean
  selected?: boolean
}>()

defineEmits<{
  (e: 'show-detail', book: Book): void
  (e: 'toggle-select', id: string): void
}>()

const coverSrc = useBookCover(
  () => props.book.id,
  () => Boolean(props.book.cover_path),
)

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
  position: relative;
}

.select-overlay {
  position: absolute;
  top: 0.4rem;
  left: 0.4rem;
  opacity: 0;
  transition: opacity 0.15s;
  z-index: 2;
}
.select-checkbox {
  width: 1.1rem;
  height: 1.1rem;
  cursor: pointer;
  accent-color: var(--accent);
  border-radius: 0.2rem;
}
.book-card:hover .select-overlay,
.book-card.is-selected .select-overlay {
  opacity: 1;
}
.book-card.is-selected {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--accent) 40%, transparent);
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

.card-rating {
  font-size: 0.7rem;
  margin-top: 0.2rem;
}

@media (max-width: 400px) {
  .title {
    font-size: 0.95rem;
  }
  .author {
    font-size: 0.82rem;
  }
}
</style>
