<template>
  <div class="rating-stars" :class="{ interactive }" :aria-label="`Rating: ${rating ?? 0} of 5`">
    <button
      v-for="n in 5"
      :key="n"
      class="star"
      :class="{ filled: n <= (hovered ?? rating ?? 0) }"
      :disabled="!interactive"
      :aria-label="`Rate ${n} star${n !== 1 ? 's' : ''}`"
      @click="interactive && $emit('rate', n)"
      @mouseenter="interactive && (hovered = n)"
      @mouseleave="interactive && (hovered = null)"
    >★</button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

defineProps<{
  rating?: number
  interactive?: boolean
}>()

defineEmits<{ (e: 'rate', n: number): void }>()

const hovered = ref<number | null>(null)
</script>

<style scoped>
.rating-stars {
  display: inline-flex;
  gap: 0.1rem;
}
.star {
  background: none;
  border: none;
  padding: 0;
  font-size: 1rem;
  line-height: 1;
  cursor: default;
  color: var(--text-muted);
  transition: color 0.1s;
}
.star.filled {
  color: var(--accent);
}
.interactive .star {
  cursor: pointer;
}
.interactive .star:hover,
.interactive .star:focus {
  outline: none;
  color: var(--accent-dim);
}
.interactive .star.filled {
  color: var(--accent);
}
</style>
